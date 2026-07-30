package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/generate"
	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// GenerateRun is the outcome of one WF-C pass.
type GenerateRun struct {
	Considered int      `json:"considered"`
	Generated  int      `json:"generated"`
	Failed     int      `json:"failed"`
	Model      string   `json:"model"`
	JobIDs     []string `json:"job_ids"`
	// Interrupted / RateLimited carry the same meaning as in ScoreRun: the run
	// stopped early and the untouched jobs are still in Scored.
	Interrupted bool `json:"interrupted,omitempty"`
	RateLimited bool `json:"rate_limited,omitempty"`
}

// DefaultGenerateBatch is deliberately small. Generation is the most expensive
// stage — two documents of prose per job — and on a free tier metered per day
// a large batch would exhaust the allowance in one run.
const DefaultGenerateBatch = 3

// MaxGenerateBatch caps it.
const MaxGenerateBatch = 20

// GenerateForScored is WF-C: take jobs sitting in `Scored`, write a tailored CV
// and cover letter for each, and move them to `AwaitingApproval`.
//
// The status walk is Scored -> CVGenerated -> AwaitingApproval, both legal
// edges, done in that order so a crash between the two leaves the job in
// CVGenerated with its documents intact rather than in a state that claims
// nothing was produced.
//
// Unlike scoring there is no free fallback: writing prose needs a model. With
// none configured the endpoint reports that rather than pretending.
func (s *Service) GenerateForScored(ctx context.Context, limit int) (GenerateRun, error) {
	client, defaultModel, err := s.llmClient()
	if err != nil {
		return GenerateRun{}, err
	}

	if limit <= 0 {
		limit = DefaultGenerateBatch
	}
	if limit > MaxGenerateBatch {
		limit = MaxGenerateBatch
	}

	profile, err := s.GetProfile(ctx)
	if err != nil {
		return GenerateRun{}, err
	}
	if strings.TrimSpace(profile.MasterCV) == "" {
		return GenerateRun{}, fmt.Errorf("%w: profile.master_cv is empty; there is nothing to tailor from", ErrInvalidInput)
	}

	// Per-stage override, so generation can use a stronger model than scoring —
	// and, where quota is metered per model, a separate daily allowance.
	model := strings.TrimSpace(profile.GenerationModel)
	if model == "" {
		model = defaultModel
	}

	jobs, err := s.jobsWithStatus(ctx, models.StatusScored, limit)
	if err != nil {
		return GenerateRun{}, err
	}

	run := GenerateRun{Considered: len(jobs), Model: model, JobIDs: make([]string, 0, len(jobs))}

	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			s.log.Warn("generation run cut short", "reason", err, "done", run.Generated)
			run.Interrupted = true
			break
		}

		docs, err := generate.Generate(ctx, client, model, profile, job)
		if errors.Is(err, scoring.ErrRateLimited) {
			// Transient and provider-wide: stop, leave the rest in Scored.
			s.log.Warn("provider rate limit reached during generation", "done", run.Generated)
			run.Interrupted, run.RateLimited = true, true
			break
		}
		if err != nil {
			run.Failed++
			s.log.Error("generation failed", "job_id", job.ID, "title", job.Title, "error", err)
			// The job stays in Scored: a later run retries it. There is no
			// GenerationFailed state, and inventing one would mean a status the
			// database CHECK rejects.
			s.logErrorDetached(ctx, "WF-C", &job.ID, fmt.Sprintf("generation failed: %v", err))
			continue
		}

		if err := s.storeDocuments(ctx, job, docs); err != nil {
			run.Failed++
			s.log.Error("could not store generated documents", "job_id", job.ID, "error", err)
			s.logErrorDetached(ctx, "WF-C", &job.ID, fmt.Sprintf("storing documents failed: %v", err))
			continue
		}

		run.Generated++
		run.JobIDs = append(run.JobIDs, job.ID)
	}

	s.log.Info("generation run complete",
		"considered", run.Considered, "generated", run.Generated,
		"failed", run.Failed, "model", model)
	return run, nil
}

// storeDocuments writes the documents and walks the job to AwaitingApproval.
func (s *Service) storeDocuments(ctx context.Context, job models.Job, docs generate.Documents) error {
	cvURL := "/jobs/" + job.ID + "/cv"
	letterURL := "/jobs/" + job.ID + "/cover-letter"

	if _, err := s.db.Pool.Exec(ctx, queries.StoreDocuments,
		job.ID, docs.CV, docs.CoverLetter, docs.Model, cvURL, letterURL); err != nil {
		return fmt.Errorf("write documents: %w", err)
	}

	// Two legal transitions, in order. Doing them through UpdateJob rather than
	// a raw UPDATE keeps the state machine the single gatekeeper.
	for _, st := range []string{models.StatusCVGenerated, models.StatusAwaitingApproval} {
		status := st
		if _, err := s.UpdateJob(ctx, job.ID, models.JobUpdate{Status: &status}); err != nil {
			return fmt.Errorf("advance to %s: %w", st, err)
		}
	}

	s.log.Info("documents generated", "job_id", job.ID, "title", job.Title,
		"cv_chars", len(docs.CV), "letter_chars", len(docs.CoverLetter), "model", docs.Model)
	return nil
}

// Documents returns the generated CV and cover letter for a job.
func (s *Service) Documents(ctx context.Context, id string) (cv, letter string, generatedAt *time.Time, err error) {
	err = s.db.Pool.QueryRow(ctx, queries.GetDocuments, id).Scan(&cv, &letter, &generatedAt)
	if err != nil {
		// Matching on the sentinel and the SQLSTATE, not on message text.
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			return "", "", nil, ErrNotFound
		}
		return "", "", nil, fmt.Errorf("read documents: %w", err)
	}
	return cv, letter, generatedAt, nil
}

// llmClient resolves the model client, or explains why there is not one.
//
// It reads the dedicated client rather than reaching into the scorer, so
// generation works with SCORING_MODE=keyword — free scoring alongside LLM
// generation is a supported combination, not an accident.
func (s *Service) llmClient() (scoring.Client, string, error) {
	if s.llm == nil {
		return nil, "", fmt.Errorf("%w: generation needs a model — set LLM_API_KEY (and LLM_PROVIDER) and restart", ErrInvalidInput)
	}
	return s.llm, s.llmModel, nil
}

// logErrorDetached records a failure on a context that cannot already be dead,
// so the explanation survives a cancelled run.
func (s *Service) logErrorDetached(ctx context.Context, workflow string, jobID *string, msg string) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	s.LogError(writeCtx, workflow, jobID, msg, nil)
}
