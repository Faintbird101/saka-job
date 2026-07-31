package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// ---------- profile ----------

// GetProfile reads the singleton settings row.
func (s *Service) GetProfile(ctx context.Context) (models.Profile, error) {
	var p models.Profile
	var titles, skills, locations []byte

	err := s.db.Pool.QueryRow(ctx, queries.GetProfile).Scan(
		&p.MasterCV, &titles, &skills,
		&p.MinScoreThreshold, &p.MaxJobsPerRun,
		&p.ScoringModel, &p.GenerationModel, &p.CoverLetterNotes,
		&p.ManualApplyGraceDays, &p.NotifyEmail, &p.InboxAutoConfidence,
		&p.FollowUpAfterDays, &p.FollowUpCloseDays,
		&p.PushOnApproval, &p.PushOnReply, &p.PushOnFollowUp, &p.PushOnFailure,
		&locations, &p.RemotePreference, &p.SalaryFloor, &p.SalaryCurrency,
		&p.WeightSkills, &p.WeightSeniority, &p.WeightDomain, &p.WeightLocation, &p.WeightPay,
		&p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		// 0001_init.sql seeds id=1, so this means the row was deleted by hand.
		return models.Profile{}, fmt.Errorf("profile row missing: the singleton (id=1) was deleted; re-run migrations")
	}
	if err != nil {
		return models.Profile{}, fmt.Errorf("read profile: %w", err)
	}

	p.SearchTitles = jsonOr(titles, "[]")
	p.PreferredSkills = jsonOr(skills, "[]")
	p.PreferredLocations = jsonOr(locations, "[]")
	return p, nil
}

// UpdateProfile patches the singleton.
//
// The two numeric settings are validated here rather than trusted, because
// they are the pipeline's safety rails: max_jobs_per_run is what keeps the
// twice-daily cron inside the API quota, and a threshold outside 0–100 would
// silently park every job in LowMatch.
func (s *Service) UpdateProfile(ctx context.Context, patch models.ProfileUpdate) (models.Profile, error) {
	if patch.MinScoreThreshold != nil && (*patch.MinScoreThreshold < 0 || *patch.MinScoreThreshold > 100) {
		return models.Profile{}, fmt.Errorf("%w: min_score_threshold must be between 0 and 100", ErrInvalidInput)
	}
	if patch.ManualApplyGraceDays != nil && (*patch.ManualApplyGraceDays < 0 || *patch.ManualApplyGraceDays > 365) {
		return models.Profile{}, fmt.Errorf("%w: manual_apply_grace_days must be between 0 and 365 (0 disables expiry)", ErrInvalidInput)
	}
	if patch.MaxJobsPerRun != nil && (*patch.MaxJobsPerRun < 1 || *patch.MaxJobsPerRun > 100) {
		return models.Profile{}, fmt.Errorf("%w: max_jobs_per_run must be between 1 and 100", ErrInvalidInput)
	}
	if err := validateJSONArray("search_titles", patch.SearchTitles); err != nil {
		return models.Profile{}, err
	}
	if err := validateJSONArray("preferred_skills", patch.PreferredSkills); err != nil {
		return models.Profile{}, err
	}

	var p models.Profile
	var titles, skills, locations2 []byte

	err := s.db.Pool.QueryRow(ctx, queries.UpdateProfile,
		patch.MasterCV,
		rawOrNil(patch.SearchTitles),
		rawOrNil(patch.PreferredSkills),
		patch.MinScoreThreshold,
		patch.MaxJobsPerRun,
		patch.ScoringModel,
		patch.GenerationModel,
		patch.CoverLetterNotes,
		patch.ManualApplyGraceDays,
		patch.NotifyEmail,
		patch.InboxAutoConfidence,
		patch.FollowUpAfterDays,
		patch.FollowUpCloseDays,
		patch.PushOnApproval,
		patch.PushOnReply,
		patch.PushOnFollowUp,
		patch.PushOnFailure,
		rawOrNil(patch.PreferredLocations),
		patch.RemotePreference,
		patch.SalaryFloor,
		patch.SalaryCurrency,
		patch.WeightSkills,
		patch.WeightSeniority,
		patch.WeightDomain,
		patch.WeightLocation,
		patch.WeightPay,
	).Scan(&p.MasterCV, &titles, &skills, &p.MinScoreThreshold, &p.MaxJobsPerRun,
		&p.ScoringModel, &p.GenerationModel, &p.CoverLetterNotes,
		&p.ManualApplyGraceDays, &p.NotifyEmail, &p.InboxAutoConfidence, &p.FollowUpAfterDays, &p.FollowUpCloseDays,
		&p.PushOnApproval, &p.PushOnReply, &p.PushOnFollowUp, &p.PushOnFailure,
		&locations2, &p.RemotePreference, &p.SalaryFloor, &p.SalaryCurrency,
		&p.WeightSkills, &p.WeightSeniority, &p.WeightDomain, &p.WeightLocation, &p.WeightPay,
		&p.UpdatedAt)
	if err != nil {
		return models.Profile{}, fmt.Errorf("update profile: %w", err)
	}

	p.SearchTitles = jsonOr(titles, "[]")
	p.PreferredSkills = jsonOr(skills, "[]")
	p.PreferredLocations = jsonOr(locations2, "[]")
	return p, nil
}

// validateJSONArray rejects a settings field that isn't a JSON array of
// strings. n8n reads these to build its search query, so a malformed value
// here would break the ingest workflow at 07:00 rather than at edit time.
func validateJSONArray(field string, raw *json.RawMessage) error {
	if raw == nil || len(*raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(*raw, &out); err != nil {
		return fmt.Errorf("%w: %s must be a JSON array of strings", ErrInvalidInput, field)
	}
	return nil
}

// ---------- errors ----------

// LogError appends to the application error log.
//
// It deliberately swallows its own failure: this is called from failure paths,
// and a logging error must not replace the original error the caller is
// already handling.
func (s *Service) LogError(ctx context.Context, workflow string, jobID *string, message string, detail json.RawMessage) {
	var id string
	var at time.Time
	err := s.db.Pool.QueryRow(ctx, queries.InsertError,
		nullIfEmpty(workflow), jobID, message, jsonbOrNull(detail),
	).Scan(&id, &at)
	if err != nil {
		s.log.Error("could not write to errors table", "error", err, "original_message", message)
	}
}

// RecordError is the /internal/errors endpoint's entry point: same write, but
// it reports failure so n8n knows the log didn't take.
func (s *Service) RecordError(ctx context.Context, rec models.ErrorRecord) (models.ErrorRecord, error) {
	if rec.Message == "" {
		return models.ErrorRecord{}, fmt.Errorf("%w: message is required", ErrInvalidInput)
	}

	err := s.db.Pool.QueryRow(ctx, queries.InsertError,
		nullIfEmpty(rec.Workflow), rec.JobID, rec.Message, jsonbOrNull(rec.Context),
	).Scan(&rec.ID, &rec.CreatedAt)
	if err != nil {
		return models.ErrorRecord{}, fmt.Errorf("insert error record: %w", err)
	}
	return rec, nil
}

// ListErrors returns the newest error rows.
func (s *Service) ListErrors(ctx context.Context, limit int) ([]models.ErrorRecord, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}

	rows, err := s.db.Pool.Query(ctx, queries.ListErrors, limit)
	if err != nil {
		return nil, fmt.Errorf("list errors: %w", err)
	}
	defer rows.Close()

	out := make([]models.ErrorRecord, 0, limit)
	for rows.Next() {
		var e models.ErrorRecord
		var ctxJSON []byte
		if err := rows.Scan(&e.ID, &e.Workflow, &e.JobID, &e.Message, &ctxJSON, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan error record: %w", err)
		}
		e.Context = jsonOr(ctxJSON, "{}")
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------- fetch log ----------

// ListFetchLogs returns recent ingest runs, newest first.
func (s *Service) ListFetchLogs(ctx context.Context, limit int) ([]models.FetchLog, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}

	rows, err := s.db.Pool.Query(ctx, queries.ListFetchLogs, limit)
	if err != nil {
		return nil, fmt.Errorf("list fetch logs: %w", err)
	}
	defer rows.Close()

	out := make([]models.FetchLog, 0, limit)
	for rows.Next() {
		var f models.FetchLog
		if err := rows.Scan(&f.ID, &f.FetchedAt, &f.QueryTitle,
			&f.ReturnedCount, &f.InsertedCount, &f.SkippedCount, &f.Notes); err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ---------- stats ----------

// Stats builds the dashboard payload: one count per pipeline stage, plus the
// last ingest run and a 24-hour error count.
func (s *Service) Stats(ctx context.Context) (models.Stats, error) {
	var out models.Stats

	rows, err := s.db.Pool.Query(ctx, queries.StatusCounts)
	if err != nil {
		return out, fmt.Errorf("status counts: %w", err)
	}
	counts := make(map[string]int, len(models.AllStatuses))
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return out, fmt.Errorf("scan status count: %w", err)
		}
		counts[status] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("iterate status counts: %w", err)
	}

	// Emit every status in pipeline order, including the zeros, so the app can
	// render a stable set of bars instead of a chart that changes shape as
	// jobs move through.
	out.ByStatus = make([]models.StatusCount, 0, len(models.AllStatuses))
	for _, st := range models.AllStatuses {
		out.ByStatus = append(out.ByStatus, models.StatusCount{Status: st, Count: counts[st]})
	}

	if err := s.db.Pool.QueryRow(ctx, queries.CountAllJobs).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("count jobs: %w", err)
	}
	if err := s.db.Pool.QueryRow(ctx, queries.CountRecentErrors).Scan(&out.RecentErrors); err != nil {
		return out, fmt.Errorf("count recent errors: %w", err)
	}

	var f models.FetchLog
	err = s.db.Pool.QueryRow(ctx, queries.LatestFetchLog).Scan(
		&f.ID, &f.FetchedAt, &f.QueryTitle,
		&f.ReturnedCount, &f.InsertedCount, &f.SkippedCount, &f.Notes,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// No ingest has ever run. Not an error — LastFetch stays null.
	case err != nil:
		return out, fmt.Errorf("latest fetch log: %w", err)
	default:
		out.LastFetch = &f
	}

	return out, nil
}

// ClearErrors empties the application error log and reports how many rows went.
//
// The 24-hour error count on the dashboard is only meaningful if resolved
// noise can be cleared; otherwise a burst of failures from one bad afternoon
// makes the badge permanently red and you stop looking at it.
func (s *Service) ClearErrors(ctx context.Context, olderThan *time.Time) (int64, error) {
	var tag pgconn.CommandTag
	var err error
	if olderThan != nil {
		tag, err = s.db.Pool.Exec(ctx, queries.ClearErrorsBefore, *olderThan)
	} else {
		tag, err = s.db.Pool.Exec(ctx, queries.ClearErrors)
	}
	if err != nil {
		return 0, fmt.Errorf("clear errors: %w", err)
	}
	return tag.RowsAffected(), nil
}
