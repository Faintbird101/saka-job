package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/inbox"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// ScanResult is the outcome of one WF-F pass.
type ScanResult struct {
	Received  int `json:"received"`
	Matched   int `json:"matched"`
	Unmatched int `json:"unmatched"`
	Ambiguous int `json:"ambiguous"`
	Duplicate int `json:"duplicate"`
	// AutoApplied counts status changes made without asking — only ever the
	// safe, non-terminal ones.
	AutoApplied int `json:"auto_applied"`
	// Suggested counts consequential classifications parked for confirmation.
	Suggested int `json:"suggested"`
	// EarliestRelevant tells the caller how far back to fetch next time.
	EarliestRelevant *time.Time        `json:"earliest_relevant,omitempty"`
	Events           []models.JobEvent `json:"events"`
}

// MaxScanBatch caps one scan. IMAP can hand over a large backlog on first run.
const MaxScanBatch = 200

// ScanInbox is WF-F: attribute inbound email to applications and record what it
// says.
//
// The safety rule is the whole design: only an acknowledgement is applied
// automatically, because "they got it" is not a decision. Rejections, interview
// invitations, and offers are recorded as SUGGESTIONS for you to confirm,
// because acting on a misread would either abandon a live opportunity or claim
// progress that did not happen.
func (s *Service) ScanInbox(ctx context.Context, msgs []inbox.Message) (ScanResult, error) {
	if len(msgs) > MaxScanBatch {
		return ScanResult{}, fmt.Errorf("%w: %d messages exceeds the maximum of %d", ErrInvalidInput, len(msgs), MaxScanBatch)
	}

	profile, err := s.GetProfile(ctx)
	if err != nil {
		return ScanResult{}, err
	}

	candidates, err := s.jobsAwaitingReply(ctx)
	if err != nil {
		return ScanResult{}, err
	}

	res := ScanResult{Received: len(msgs), Events: make([]models.JobEvent, 0, len(msgs))}

	// Reported back so the next scan can narrow its IMAP window. Nothing older
	// than the first application can be a reply to one.
	if earliest, err := s.earliestApplication(ctx); err == nil && earliest != nil {
		res.EarliestRelevant = earliest
	}

	for _, msg := range msgs {
		if err := ctx.Err(); err != nil {
			s.log.Warn("inbox scan cut short", "reason", err, "done", res.Matched+res.Unmatched)
			break
		}

		match := inbox.FindMatch(msg, candidates)
		class := inbox.Classify(ctx, s.llm, s.inboxModel(profile), msg)

		event, stored, err := s.recordEvent(ctx, profile, msg, match, class)
		if err != nil {
			s.log.Error("could not record inbox event", "subject", msg.Subject, "error", err)
			continue
		}
		if !stored {
			// Already seen: IMAP re-delivers after a restart or an overlapping
			// window, and recording it twice would suggest the same rejection
			// twice.
			res.Duplicate++
			continue
		}

		switch {
		case match.JobID == "":
			res.Unmatched++
		case match.Ambiguous:
			res.Ambiguous++
		default:
			res.Matched++
		}
		if event.AppliedAt != nil {
			res.AutoApplied++
		} else if event.SuggestedStatus != "" {
			res.Suggested++
		}
		res.Events = append(res.Events, event)
	}

	s.log.Info("inbox scan complete",
		"received", res.Received, "matched", res.Matched, "unmatched", res.Unmatched,
		"ambiguous", res.Ambiguous, "duplicate", res.Duplicate,
		"auto_applied", res.AutoApplied, "suggested", res.Suggested)

	return res, nil
}

// recordEvent writes the event and, when it is safe to do so, applies the
// status change. Returns stored=false when the message was already recorded.
func (s *Service) recordEvent(
	ctx context.Context, profile models.Profile,
	msg inbox.Message, match inbox.Match, class inbox.Classification,
) (models.JobEvent, bool, error) {

	suggested := inbox.SuggestedStatus(class.Kind)

	// A suggestion is only meaningful if we know which job it belongs to. An
	// ambiguous or unmatched email is filed for review with no suggestion, so it
	// can never move the wrong application.
	if match.JobID == "" || match.Ambiguous {
		suggested = ""
	}

	var jobID *string
	if match.JobID != "" && !match.Ambiguous {
		jobID = &match.JobID
	}

	// Decide up front whether this is applied now or parked.
	var confirmed *bool
	var appliedAt *time.Time

	autoOK := suggested != "" &&
		!class.Kind.Consequential() &&
		class.Confidence >= profile.InboxAutoConfidence

	if autoOK {
		yes := true
		now := time.Now()
		confirmed, appliedAt = &yes, &now
	}

	var received *time.Time
	if t := parseEmailTime(msg.ReceivedAt); t != nil {
		received = t
	}

	event, err := scanEvent(s.db.Pool.QueryRow(ctx, queries.InsertEvent,
		jobID, string(class.Kind), class.Confidence, class.Classifier,
		nullIfEmpty(msg.From), nullIfEmpty(inbox.SenderDomain(msg.From)),
		nullIfEmpty(trimTo(msg.Subject, 500)), nullIfEmpty(excerptOf(msg, class)),
		received,
		match.Score, nullIfEmpty(match.Reason), nullIfEmpty(suggested),
		confirmed, appliedAt,
		nullIfEmpty(msg.MessageID),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.JobEvent{}, false, nil // duplicate message_id
	}
	if err != nil {
		return models.JobEvent{}, false, fmt.Errorf("insert event: %w", err)
	}

	// Apply the safe transition. Done after the insert so a failure here leaves
	// a recorded event explaining what was attempted.
	if autoOK && jobID != nil {
		st := suggested
		if _, err := s.UpdateJob(ctx, *jobID, models.JobUpdate{Status: &st}); err != nil {
			// Not fatal: an illegal transition (the job has already moved on)
			// just means the event stands as a record without a status change.
			s.log.Warn("inbox event could not update status",
				"job_id", *jobID, "to", st, "error", err)
			event.AppliedAt = nil
		} else {
			s.log.Info("inbox event applied automatically",
				"job_id", *jobID, "status", st, "confidence", class.Confidence)
		}
	}

	return event, true, nil
}

// ConfirmEvent applies or dismisses a parked suggestion.
//
// accept=false records that the classification was wrong, which is deliberately
// kept rather than deleted: it is the only evidence available for improving the
// rules.
func (s *Service) ConfirmEvent(ctx context.Context, eventID string, accept bool) (models.JobEvent, error) {
	event, err := scanEvent(s.db.Pool.QueryRow(ctx, queries.GetEvent, eventID))
	if err != nil {
		return models.JobEvent{}, err
	}
	if event.SuggestedStatus == "" {
		return models.JobEvent{}, fmt.Errorf("%w: this event suggests no status change", ErrInvalidInput)
	}
	if event.Confirmed != nil {
		return models.JobEvent{}, fmt.Errorf("%w: this suggestion was already decided", ErrConflict)
	}
	if event.JobID == nil {
		return models.JobEvent{}, fmt.Errorf("%w: this event is not attached to a job", ErrInvalidInput)
	}

	if accept {
		st := event.SuggestedStatus
		if _, err := s.UpdateJob(ctx, *event.JobID, models.JobUpdate{Status: &st}); err != nil {
			return models.JobEvent{}, err
		}
	}

	updated, err := scanEvent(s.db.Pool.QueryRow(ctx, queries.SetEventConfirmed, eventID, accept))
	if err != nil {
		return models.JobEvent{}, err
	}
	s.log.Info("inbox suggestion decided",
		"event_id", eventID, "job_id", *event.JobID,
		"status", event.SuggestedStatus, "accepted", accept)
	return updated, nil
}

// EventsForJob is the reply timeline for one job.
func (s *Service) EventsForJob(ctx context.Context, jobID string, limit int) ([]models.JobEvent, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return s.listEvents(ctx, queries.ListEventsForJob, jobID, limit)
}

// PendingEvents is every suggestion awaiting a decision.
func (s *Service) PendingEvents(ctx context.Context, limit int) ([]models.JobEvent, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return s.listEvents(ctx, queries.ListPendingEvents, limit)
}

// UnmatchedEvents is mail that could not be attributed.
func (s *Service) UnmatchedEvents(ctx context.Context, limit int) ([]models.JobEvent, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return s.listEvents(ctx, queries.ListUnmatchedEvents, limit)
}

func (s *Service) listEvents(ctx context.Context, q string, args ...any) ([]models.JobEvent, error) {
	rows, err := s.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	out := make([]models.JobEvent, 0, 16)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// jobsAwaitingReply is the candidate set for matching.
func (s *Service) jobsAwaitingReply(ctx context.Context) ([]models.Job, error) {
	rows, err := s.db.Pool.Query(ctx, queries.JobsAwaitingReply)
	if err != nil {
		return nil, fmt.Errorf("list candidate jobs: %w", err)
	}
	defer rows.Close()

	out := make([]models.Job, 0, 32)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Service) earliestApplication(ctx context.Context) (*time.Time, error) {
	var t *time.Time
	if err := s.db.Pool.QueryRow(ctx, queries.EarliestApplication).Scan(&t); err != nil {
		return nil, err
	}
	return t, nil
}

// inboxModel picks the model for escalated classifications, reusing the
// scoring model since both are short classification tasks.
func (s *Service) inboxModel(p models.Profile) string {
	if m := strings.TrimSpace(p.ScoringModel); m != "" {
		return m
	}
	return s.llmModel
}

// excerptOf keeps just enough of the message to justify the classification.
// Storing whole bodies would put entire threads of someone else's
// correspondence in the database for no operational benefit.
func excerptOf(msg inbox.Message, class inbox.Classification) string {
	if class.Evidence != "" {
		return class.Evidence
	}
	return trimTo(strings.Join(strings.Fields(msg.Body), " "), 300)
}

func trimTo(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// parseEmailTime accepts the formats IMAP and n8n hand over.
func parseEmailTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{
		time.RFC3339, time.RFC1123Z, time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// scanEvent reads one job_events row in the exact order of
// queries.EventColumns. Same matched-pair discipline as scanJob.
func scanEvent(r row) (models.JobEvent, error) {
	var e models.JobEvent
	err := r.Scan(
		&e.ID,
		&e.JobID,
		&e.Source,
		&e.Kind,
		&e.Confidence,
		&e.Classifier,
		&e.Sender,
		&e.SenderDomain,
		&e.Subject,
		&e.Excerpt,
		&e.ReceivedAt,
		&e.MatchScore,
		&e.MatchReason,
		&e.SuggestedStatus,
		&e.Confirmed,
		&e.AppliedAt,
		&e.MessageID,
		&e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			return models.JobEvent{}, ErrNotFound
		}
		return models.JobEvent{}, fmt.Errorf("scan job event: %w", err)
	}
	return e, nil
}
