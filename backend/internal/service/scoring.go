package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// ScoreRun is the outcome of one WF-B pass.
type ScoreRun struct {
	Considered int    `json:"considered"`
	Scored     int    `json:"scored"`
	LowMatch   int    `json:"low_match"`
	Failed     int    `json:"failed"`
	Threshold  int    `json:"threshold"`
	Scorer     string `json:"scorer"`
	// Interrupted reports that the run stopped early. The unscored jobs are
	// still in `New`, so the next run continues rather than losing them.
	Interrupted bool `json:"interrupted,omitempty"`
	// RateLimited distinguishes "the provider throttled us" from other early
	// exits. On a free tier this is a normal outcome, not a failure: the batch
	// simply resumes on the next scheduled run.
	RateLimited bool     `json:"rate_limited,omitempty"`
	JobIDs      []string `json:"job_ids"`
}

// DefaultScoreBatch bounds one run when the caller doesn't say.
const DefaultScoreBatch = 10

// MaxScoreBatch caps it. Each job is a separate model call, so an unbounded
// batch is an unbounded bill and a request that outlives its own timeout.
const MaxScoreBatch = 50

// ScoreNewJobs is WF-B: take jobs sitting in `New`, score each against the
// profile, and move it to `Scored` or `LowMatch`.
//
// Notes on the shape of this:
//
//   - One model call per job, sequentially. Scoring in parallel would be
//     faster but makes rate-limit handling and partial failure much harder to
//     reason about, for a batch that is 10 jobs twice a day.
//
//   - A failure on one job never aborts the batch. It is recorded on that row
//     as ScoreFailed and in the errors table, and the run continues. One
//     malformed reply should not deny the other nine jobs their scores.
//
//   - Nothing is retried in-process. ScoreFailed is a resting state the next
//     scheduled run can pick up, which is the same state-machine argument the
//     rest of the pipeline uses.
func (s *Service) ScoreNewJobs(ctx context.Context, limit int) (ScoreRun, error) {
	if s.scorer == nil {
		return ScoreRun{}, fmt.Errorf("%w: scoring is not configured — set SCORING_MODE=keyword for the free scorer, or supply LLM_API_KEY for SCORING_MODE=llm", ErrInvalidInput)
	}
	if limit <= 0 {
		limit = DefaultScoreBatch
	}
	if limit > MaxScoreBatch {
		limit = MaxScoreBatch
	}

	profile, err := s.GetProfile(ctx)
	if err != nil {
		return ScoreRun{}, err
	}

	// Refuse rather than score against an empty CV. Every job would come back
	// near zero, every one of them would land in LowMatch, and the run would
	// look successful while quietly poisoning the pipeline — and it would bill
	// a model call per job to do it.
	if strings.TrimSpace(profile.MasterCV) == "" {
		return ScoreRun{}, fmt.Errorf("%w: profile.master_cv is empty; set it before scoring or every job will be parked in LowMatch", ErrInvalidInput)
	}

	jobs, err := s.jobsWithStatus(ctx, models.StatusNew, limit)
	if err != nil {
		return ScoreRun{}, err
	}

	run := ScoreRun{
		Considered: len(jobs),
		Threshold:  profile.MinScoreThreshold,
		Scorer:     s.scorer.Name(),
		JobIDs:     make([]string, 0, len(jobs)),
	}

	for _, job := range jobs {
		// Stop cleanly if the request is going away (timeout, client hang-up,
		// shutdown). Continuing would mark every remaining job ScoreFailed for
		// a reason that has nothing to do with the job — the rows stay in `New`
		// and the next run picks them up instead.
		if err := ctx.Err(); err != nil {
			s.log.Warn("scoring run cut short",
				"reason", err, "scored_so_far", run.Scored+run.LowMatch,
				"remaining", run.Considered-(run.Scored+run.LowMatch+run.Failed))
			run.Interrupted = true
			break
		}

		status, err := s.scoreOne(ctx, profile, job)

		// A rate limit is transient and affects every remaining call in the
		// window, so stop rather than burning through the rest of the batch
		// marking jobs ScoreFailed for a reason that is not their fault. They
		// stay in New and the next scheduled run continues.
		if errors.Is(err, scoring.ErrRateLimited) {
			s.log.Warn("provider rate limit reached, ending run early",
				"scored_so_far", run.Scored+run.LowMatch, "error", err)
			run.Interrupted = true
			run.RateLimited = true
			break
		}

		switch {
		case err != nil:
			run.Failed++
			s.log.Error("scoring failed", "job_id", job.ID, "title", job.Title, "error", err)
		case status == models.StatusScored:
			run.Scored++
			run.JobIDs = append(run.JobIDs, job.ID)
		default:
			run.LowMatch++
		}
	}

	s.log.Info("scoring run complete",
		"considered", run.Considered, "scored", run.Scored,
		"low_match", run.LowMatch, "failed", run.Failed, "threshold", run.Threshold)

	return run, nil
}

// scoreOne scores a single job and writes the outcome. It returns the status
// the job ended in.
func (s *Service) scoreOne(ctx context.Context, profile models.Profile, job models.Job) (string, error) {
	result, err := s.scorer.Score(ctx, profile, job)
	if err == nil {
		// Only the LLM path has a prompt worth auditing; the keyword scorer is
		// deterministic, so its inputs are already the stored job row.
		audit := ""
		if l, ok := s.scorer.(*scoring.LLMScorer); ok {
			audit = l.Prompt(profile, job)
		}
		return s.applyScore(ctx, profile, job, result, audit)
	}

	// A rate limit leaves the row untouched: it is a provider condition, not a
	// property of this job, and ScoreFailed would misattribute it.
	if errors.Is(err, scoring.ErrRateLimited) {
		return "", err
	}

	// Either the call or the parse failed. Park the job so the next run can
	// retry it, and record why — an unexplained ScoreFailed row is useless
	// when you come back to it a day later.
	s.log.Warn("marking job ScoreFailed", "job_id", job.ID, "error", err)

	// Detach from the request context for the bookkeeping writes. If the reason
	// for failing was the context itself, using it again loses the explanation
	// and the status update — which is how a timeout became a silent
	// ScoreFailed with nothing in the errors table.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	s.LogError(writeCtx, "WF-B", &job.ID, fmt.Sprintf("scoring failed: %v", err), nil)

	failed := models.StatusScoreFailed
	if _, uerr := s.UpdateJob(writeCtx, job.ID, models.JobUpdate{Status: &failed}); uerr != nil {
		return "", fmt.Errorf("scoring failed (%v) and the status update also failed: %w", err, uerr)
	}
	return failed, err
}

// applyScore writes a successful score and the resulting status.
func (s *Service) applyScore(ctx context.Context, profile models.Profile, job models.Job, r scoring.Result, prompt string) (string, error) {
	// The threshold decision lives here, not in the prompt. The model reports
	// a score; the profile decides what counts as good enough. Changing the
	// threshold in the app must not require re-scoring anything.
	status := models.StatusLowMatch
	if r.Score >= profile.MinScoreThreshold {
		status = models.StatusScored
	}

	matched, _ := json.Marshal(r.MatchedSkills)
	missing, _ := json.Marshal(r.MissingSkills)
	matchedRaw := json.RawMessage(matched)
	missingRaw := json.RawMessage(missing)

	update := models.JobUpdate{
		Status:        &status,
		Score:         &r.Score,
		MatchedSkills: &matchedRaw,
		MissingSkills: &missingRaw,
		AISummary:     &r.Summary,
	}
	// The breakdown is what lets the app show its work rather than presenting
	// a bare number.
	if axesJSON := r.Axes.JSON(); len(axesJSON) > 0 && string(axesJSON) != "null" {
		update.ScoreAxes = &axesJSON
	}
	// prompt_used is the audit trail for "why did this get 42?". Only the LLM
	// path has one; leaving it nil in keyword mode keeps the column honest
	// rather than storing a reconstruction that was never actually sent.
	if prompt != "" {
		update.PromptUsed = &prompt
	}

	if _, err := s.UpdateJob(ctx, job.ID, update); err != nil {
		return "", fmt.Errorf("write score for job %s: %w", job.ID, err)
	}

	s.log.Info("job scored", "job_id", job.ID, "score", r.Score, "status", status,
		"scorer", s.scorer.Name(), "title", job.Title)
	return status, nil
}

// jobsWithStatus reads a batch of the work queue for a pipeline stage.
func (s *Service) jobsWithStatus(ctx context.Context, status string, limit int) ([]models.Job, error) {
	rows, err := s.db.Pool.Query(ctx, queries.ListJobsByStatus, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list %s jobs: %w", status, err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s jobs: %w", status, err)
	}
	return out, nil
}

// ErrScoringUnavailable is returned when the handler is reached but no model
// client was configured at boot.
var ErrScoringUnavailable = errors.New("scoring is not configured")

// RescoreJob re-runs scoring for a single job, whatever state it is in.
//
// This exists because tuning is iterative: after changing the CV, the
// threshold, or the scorer weights, you want to see the effect on one specific
// job without resetting the whole table by hand. It moves the job back to New
// and scores it immediately.
func (s *Service) RescoreJob(ctx context.Context, id string) (models.Job, error) {
	if s.scorer == nil {
		return models.Job{}, fmt.Errorf("%w: scoring is not configured", ErrInvalidInput)
	}

	profile, err := s.GetProfile(ctx)
	if err != nil {
		return models.Job{}, err
	}
	if strings.TrimSpace(profile.MasterCV) == "" {
		return models.Job{}, fmt.Errorf("%w: profile.master_cv is empty", ErrInvalidInput)
	}

	job, err := s.GetJob(ctx, id)
	if err != nil {
		return models.Job{}, err
	}

	// Refuse past the approval gate. Re-scoring an Applied job would rewrite
	// history for something already sent, and the transition back is illegal
	// anyway — failing here gives a clearer reason than a 409 from the state
	// machine three calls later.
	switch job.Status {
	case models.StatusApplied, models.StatusFollowUpSent, models.StatusClosed, models.StatusManualApply:
		return models.Job{}, fmt.Errorf("%w: job is %s — re-scoring something already applied for would rewrite history",
			ErrConflict, job.Status)
	}

	if _, err := s.forceStatus(ctx, id, models.StatusNew); err != nil {
		return models.Job{}, err
	}
	job.Status = models.StatusNew

	if _, err := s.scoreOne(ctx, profile, job); err != nil {
		return models.Job{}, err
	}
	return s.GetJob(ctx, id)
}

// forceStatus sets a status without a transition check.
//
// It is used only by RescoreJob, to reset a job to New so it can be scored
// again. Every other write goes through UpdateJob so the state machine stays
// the single gatekeeper — this is the deliberate, narrow exception, and it is
// why it is unexported.
func (s *Service) forceStatus(ctx context.Context, id, status string) (models.Job, error) {
	return scanJob(s.db.Pool.QueryRow(ctx, queries.ForceStatus, id, status))
}
