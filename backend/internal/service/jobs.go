package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// JobFilter is the query string of GET /jobs, already validated.
//
// The pointer fields distinguish "no filter" from "filter on the zero value" —
// score=0 is a meaningful minimum, and so is the absence of one.
type JobFilter struct {
	Status   string
	MinScore *int
	Country  string
	Search   string
	Limit    int
	Offset   int
}

// DefaultLimit / MaxLimit bound the page size. Unbounded list endpoints are
// how a phone on a slow connection ends up downloading the whole table.
const (
	DefaultLimit = 50
	MaxLimit     = 200
)

// Normalize clamps the paging values and validates the status filter.
func (f *JobFilter) Normalize() error {
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Status != "" && !models.IsValidStatus(f.Status) {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidInput, f.Status)
	}
	if f.MinScore != nil && (*f.MinScore < 0 || *f.MinScore > 100) {
		return fmt.Errorf("%w: min_score must be between 0 and 100", ErrInvalidInput)
	}
	return nil
}

// JobPage is a page of results plus the total matching count, so the app can
// show "showing 50 of 312" without a second request.
type JobPage struct {
	Jobs   []models.Job `json:"jobs"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// ListJobs returns a filtered, paged slice of jobs.
func (s *Service) ListJobs(ctx context.Context, f JobFilter) (JobPage, error) {
	if err := f.Normalize(); err != nil {
		return JobPage{}, err
	}

	args := []any{
		nullIfEmpty(f.Status),
		f.MinScore,
		nullIfEmpty(f.Country),
		nullIfEmpty(f.Search),
	}

	var total int
	if err := s.db.Pool.QueryRow(ctx, queries.CountJobs, args...).Scan(&total); err != nil {
		return JobPage{}, fmt.Errorf("count jobs: %w", err)
	}

	rows, err := s.db.Pool.Query(ctx, queries.ListJobs, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return JobPage{}, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	// Non-nil so the JSON is [] rather than null on an empty result.
	jobs := make([]models.Job, 0, f.Limit)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return JobPage{}, err
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return JobPage{}, fmt.Errorf("iterate jobs: %w", err)
	}

	return JobPage{Jobs: jobs, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

// GetJob returns one job, description text included.
func (s *Service) GetJob(ctx context.Context, id string) (models.Job, error) {
	return scanJob(s.db.Pool.QueryRow(ctx, queries.GetJob, id))
}

// UpdateJob applies a partial patch, enforcing the state machine.
//
// The status check is the reason this isn't just an UPDATE in the handler: the
// database's CHECK constraint validates the *value*, but only this code
// validates the *move*. Skipping AwaitingApproval → Approved would bypass the
// human gate the whole system is built around.
func (s *Service) UpdateJob(ctx context.Context, id string, patch models.JobUpdate) (models.Job, error) {
	if patch.Status != nil {
		if !models.IsValidStatus(*patch.Status) {
			return models.Job{}, fmt.Errorf("%w: unknown status %q", ErrInvalidInput, *patch.Status)
		}

		var current string
		err := s.db.Pool.QueryRow(ctx, queries.GetJobStatus, id).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Job{}, ErrNotFound
		}
		if err != nil {
			return models.Job{}, fmt.Errorf("read current status: %w", err)
		}

		if !models.CanTransition(current, *patch.Status) {
			return models.Job{}, fmt.Errorf("%w: cannot move a job from %s to %s (allowed: %v)",
				ErrConflict, current, *patch.Status, models.NextStatuses(current))
		}
	}

	if patch.Score != nil && (*patch.Score < 0 || *patch.Score > 100) {
		return models.Job{}, fmt.Errorf("%w: score must be between 0 and 100", ErrInvalidInput)
	}

	job, err := scanJob(s.db.Pool.QueryRow(ctx, queries.UpdateJob,
		id,
		patch.Status,
		patch.Score,
		rawOrNil(patch.MatchedSkills),
		rawOrNil(patch.MissingSkills),
		patch.AISummary,
		patch.CVURL,
		patch.CoverLetterURL,
		patch.PromptUsed,
		patch.EmailUsed,
		patch.DateApplied,
		rawOrNil(patch.ScoreAxes),
		rawOrNil(patch.CVEdits),
	))
	if err != nil {
		return models.Job{}, err
	}
	return job, nil
}

// rawOrNil unwraps an optional JSONB patch field into a driver argument.
func rawOrNil(m *json.RawMessage) any {
	if m == nil {
		return nil
	}
	return jsonbOrNull(*m)
}
