package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/ingest"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// Ingest is the write half of WF-A: normalise a RapidAPI batch, insert what's
// new, and record the run.
//
// Design notes, because this method is the load-bearing one:
//
//   - Dedup is Postgres's job, not ours. Every insert is a single
//     `ON CONFLICT DO NOTHING RETURNING id`, so two ingest runs racing on the
//     same posting cannot both win. A read-then-write check in Go would have a
//     window between the SELECT and the INSERT; this has none.
//
//   - No transaction wrapping the whole batch. If job 17 of 40 fails, we want
//     the first 16 kept — a partial ingest is fine and re-running the workflow
//     picks up the rest, whereas an all-or-nothing rollback would turn one bad
//     payload into a permanently stuck run.
//
//   - A fetch_log row is written even when nothing was inserted. The point of
//     that table is quota visibility: a run that returned 40 duplicates still
//     consumed an API call, and a run that returned zero results is exactly
//     what you want to notice.
func (s *Service) Ingest(ctx context.Context, batch ingest.Batch) (models.IngestResult, error) {
	prepared, err := ingest.Prepare(batch)
	if err != nil {
		return models.IngestResult{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}

	returned := len(batch.Jobs)
	inserted := make([]string, 0, len(prepared.Keep))
	skipped := prepared.SkippedInBatch + prepared.Invalid
	problems := append([]string(nil), prepared.Problems...)

	for _, job := range prepared.Keep {
		id, err := s.insertJob(ctx, job)
		switch {
		case errors.Is(err, errDuplicate):
			skipped++
		case err != nil:
			// One bad row shouldn't sink the batch, but it must not vanish
			// either — it goes to the errors table and the fetch_log notes.
			skipped++
			problems = append(problems, fmt.Sprintf("source %d: %v", job.SourceJobID, err))
			s.log.Error("ingest insert failed",
				"source_job_id", job.SourceJobID, "title", job.Title, "error", err)
			s.LogError(ctx, "WF-A", nil, fmt.Sprintf("insert failed for source job %d: %v", job.SourceJobID, err), nil)
		default:
			inserted = append(inserted, id)
		}
	}

	logID, err := s.recordFetch(ctx, batch, returned, len(inserted), skipped, problems)
	if err != nil {
		// The jobs are already committed; failing the whole request now would
		// make n8n retry and re-do work that succeeded. Log loudly instead.
		s.log.Error("failed to write fetch_log row", "error", err)
	}

	s.log.Info("ingest complete",
		"query_title", batch.QueryTitle,
		"returned", returned,
		"inserted", len(inserted),
		"skipped", skipped,
	)

	return models.IngestResult{
		Returned:   returned,
		Inserted:   len(inserted),
		Skipped:    skipped,
		JobIDs:     inserted,
		FetchLogID: logID,
	}, nil
}

// errDuplicate is internal: it signals that ON CONFLICT DO NOTHING swallowed
// the insert, i.e. one of the three dedup guards tripped.
var errDuplicate = errors.New("duplicate job")

func (s *Service) insertJob(ctx context.Context, j models.Job) (string, error) {
	var id string
	err := s.db.Pool.QueryRow(ctx, queries.InsertJob,
		j.SourceJobID,
		nullIfZero(j.LinkedInID),
		nullIfEmpty(j.NormalizedURL),
		j.Title,
		nullIfEmpty(j.Organization),
		nullIfEmpty(j.OrganizationURL),
		nullIfEmpty(j.URL),
		nullIfEmpty(j.Source),
		nullIfEmpty(j.SourceDomain),
		nullIfEmpty(j.DescriptionText),
		j.DatePosted,
		j.DateValidThru,
		nullIfEmpty(j.Country),
		nullIfEmpty(j.LocationRaw),
		nullIfEmpty(j.WorkArrangement),
		nullIfEmpty(j.EmploymentType),
		nullIfEmpty(j.Seniority),
		nullIfEmpty(j.ExperienceLevel),
		j.DirectApply,
		jsonbOrNull(j.AIKeySkills),
		jsonbOrNull(j.AIKeywords),
		nullIfEmpty(j.AIRequirementsSummary),
		nullIfEmpty(j.AICoreResponsibilities),
		nullIfEmpty(j.SalaryCurrency),
		j.SalaryMin,
		j.SalaryMax,
		nullIfEmpty(j.SalaryUnit),
		j.Status,
		jsonbOrNull(j.RawPayload),
	).Scan(&id)

	// No returned row is not an error here — it's the conflict clause doing
	// exactly what it's there for.
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errDuplicate
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// recordFetch writes the fetch_log row for this run.
func (s *Service) recordFetch(ctx context.Context, batch ingest.Batch, returned, inserted, skipped int, problems []string) (string, error) {
	notes := strings.TrimSpace(batch.Notes)
	if len(problems) > 0 {
		joined := strings.Join(problems, "; ")
		// Keep the column readable; the full detail is in the errors table.
		if len(joined) > 1000 {
			joined = joined[:1000] + "…"
		}
		if notes != "" {
			notes += " | "
		}
		notes += joined
	}

	var id string
	var at time.Time
	err := s.db.Pool.QueryRow(ctx, queries.InsertFetchLog,
		nullIfEmpty(strings.TrimSpace(batch.QueryTitle)),
		returned, inserted, skipped,
		nullIfEmpty(notes),
	).Scan(&id, &at)
	if err != nil {
		return "", fmt.Errorf("insert fetch_log: %w", err)
	}
	return id, nil
}
