package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// FollowUpRun is the outcome of one WF-E pass.
type FollowUpRun struct {
	Considered  int      `json:"considered"`
	Nudged      int      `json:"nudged"`
	ClosedStale int      `json:"closed_stale"`
	AfterDays   int      `json:"after_days"`
	JobIDs      []string `json:"job_ids"`
	NotifyTo    string   `json:"notify_to,omitempty"`
	// DigestText is the ready-to-send reminder, rendered here so the workflow
	// stays a delivery mechanism rather than a template engine.
	DigestText string `json:"digest_text,omitempty"`
	Count      int    `json:"count"`
}

// MaxFollowUpBatch caps one run.
const MaxFollowUpBatch = 50

// RunFollowUps is WF-E: chase applications that went quiet, and close the ones
// that stayed quiet after being chased.
//
// The important part is what it does NOT do. It never emails an employer — the
// same constraint as WF-D, since no posting carries an address — and it never
// chases a job that actually got a reply. That second check is only possible
// because the inbox scanner records job_events; without it this would be a timer
// that nags people who already answered you.
func (s *Service) RunFollowUps(ctx context.Context, limit int) (FollowUpRun, error) {
	if limit <= 0 || limit > MaxFollowUpBatch {
		limit = MaxFollowUpBatch
	}

	profile, err := s.GetProfile(ctx)
	if err != nil {
		return FollowUpRun{}, err
	}

	after := profile.FollowUpAfterDays
	if after <= 0 {
		after = 7
	}

	run := FollowUpRun{AfterDays: after, JobIDs: make([]string, 0, limit)}

	// Close-out sweep first, so a job that has just passed the close threshold
	// is not nudged again on its way out.
	if profile.FollowUpCloseDays > 0 {
		closed, err := s.expireFollowUps(ctx, profile.FollowUpCloseDays)
		if err != nil {
			s.log.Error("follow-up close sweep failed", "error", err)
		} else {
			run.ClosedStale = closed
		}
	}

	jobs, err := s.jobsNeedingFollowUp(ctx, after, limit)
	if err != nil {
		return FollowUpRun{}, err
	}
	run.Considered = len(jobs)

	var nudged []models.Job
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			s.log.Warn("follow-up run cut short", "reason", err, "done", run.Nudged)
			break
		}

		st := models.StatusFollowUpSent
		if _, err := s.UpdateJob(ctx, job.ID, models.JobUpdate{Status: &st}); err != nil {
			s.log.Error("could not mark FollowUpSent", "job_id", job.ID, "error", err)
			s.logErrorDetached(ctx, "WF-E", &job.ID, fmt.Sprintf("follow-up status update failed: %v", err))
			continue
		}
		if _, err := s.db.Pool.Exec(ctx, queries.MarkFollowUpSent, job.ID); err != nil {
			// The status moved; only the clock failed. The close sweep will skip
			// it rather than closing it early, which is the safe direction.
			s.log.Warn("could not stamp followup_at", "job_id", job.ID, "error", err)
		}

		nudged = append(nudged, job)
		run.Nudged++
		run.JobIDs = append(run.JobIDs, job.ID)
	}

	run.Count = len(nudged)
	run.NotifyTo = strings.TrimSpace(profile.NotifyEmail)
	if len(nudged) > 0 {
		run.DigestText = s.renderFollowUpDigest(nudged, after)
	}

	s.log.Info("follow-up run complete",
		"considered", run.Considered, "nudged", run.Nudged,
		"closed_stale", run.ClosedStale, "after_days", after)
	return run, nil
}

// renderFollowUpDigest writes the reminder, including a draft message per job.
//
// The draft is a template rather than a model call: a follow-up note is three
// formulaic sentences, and spending daily LLM quota on it would take that quota
// away from generation, where there is no free substitute.
func (s *Service) renderFollowUpDigest(jobs []models.Job, afterDays int) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%d application(s) have had no reply in %d+ days.\n", len(jobs), afterDays)
	b.WriteString("Nothing has been sent to these employers — this is a reminder for you.\n")
	b.WriteString(strings.Repeat("=", 64) + "\n\n")

	for i, j := range jobs {
		applied := "an unknown date"
		days := 0
		if j.DateApplied != nil {
			applied = j.DateApplied.Format("2 Jan 2006")
			days = int(time.Since(*j.DateApplied).Hours() / 24)
		}

		fmt.Fprintf(&b, "%d. %s — %s\n", i+1, j.Title, j.Organization)
		fmt.Fprintf(&b, "   applied %s (%d days ago)\n", applied, days)
		if j.URL != "" {
			fmt.Fprintf(&b, "   posting: %s\n", j.URL)
		}
		if j.OrgDomain != "" {
			fmt.Fprintf(&b, "   company site: %s\n", j.OrgDomain)
		}

		b.WriteString("\n   --- draft follow-up ---\n")
		for _, line := range strings.Split(draftFollowUp(j, applied), "\n") {
			b.WriteString("   " + line + "\n")
		}
		b.WriteString("\n" + strings.Repeat("-", 64) + "\n\n")
	}

	b.WriteString("Mark each one in the app once you have chased it, or it will be\n")
	b.WriteString("closed automatically after the grace period.\n")
	return b.String()
}

// draftFollowUp writes the note you can paste. Deliberately short and plain —
// a long follow-up reads worse than a brief one.
func draftFollowUp(j models.Job, applied string) string {
	org := j.Organization
	if org == "" {
		org = "the team"
	}
	return fmt.Sprintf(`Subject: Following up — %s application

Dear %s,

I applied for the %s role on %s and wanted to follow up briefly to
confirm my continued interest in the position.

I would be glad to provide any further information that would help, and
I am happy to talk whenever convenient.

Thank you for your time.

Kind regards,`, j.Title, org, j.Title, applied)
}

func (s *Service) jobsNeedingFollowUp(ctx context.Context, afterDays, limit int) ([]models.Job, error) {
	rows, err := s.db.Pool.Query(ctx, queries.JobsNeedingFollowUp, afterDays, limit)
	if err != nil {
		return nil, fmt.Errorf("list follow-up candidates: %w", err)
	}
	defer rows.Close()

	out := make([]models.Job, 0, limit)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// expireFollowUps closes chased jobs that stayed silent.
func (s *Service) expireFollowUps(ctx context.Context, closeDays int) (int, error) {
	rows, err := s.db.Pool.Query(ctx, queries.ExpireFollowUps, closeDays)
	if err != nil {
		return 0, fmt.Errorf("expire follow-ups: %w", err)
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id, title, org string
		if err := rows.Scan(&id, &title, &org); err != nil {
			return n, fmt.Errorf("scan expired follow-up: %w", err)
		}
		// Logged individually: a job closing itself is the kind of thing you
		// want to be able to find later when you wonder where it went.
		s.log.Info("closing a followed-up job that stayed silent",
			"job_id", id, "title", title, "organization", org, "after_days", closeDays)
		n++
	}
	return n, rows.Err()
}
