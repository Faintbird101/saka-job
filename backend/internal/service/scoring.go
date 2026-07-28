package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// ScoreRun is the outcome of one WF-B pass.
type ScoreRun struct {
	Considered int      `json:"considered"`
	Scored     int      `json:"scored"`
	LowMatch   int      `json:"low_match"`
	Failed     int      `json:"failed"`
	Threshold  int      `json:"threshold"`
	Model      string   `json:"model"`
	JobIDs     []string `json:"job_ids"`
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
		return ScoreRun{}, fmt.Errorf("%w: scoring is not configured — set LLM_API_KEY and restart", ErrInvalidInput)
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
		Model:      s.scorerModel,
		JobIDs:     make([]string, 0, len(jobs)),
	}

	for _, job := range jobs {
		status, err := s.scoreOne(ctx, profile, job)
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
	userPrompt := scoring.BuildUserPrompt(profile, job)

	reply, err := s.scorer.Complete(ctx, scoring.SystemPrompt(), userPrompt)
	if err == nil {
		var result scoring.Result
		result, err = scoring.Parse(reply)
		if err == nil {
			return s.applyScore(ctx, profile, job, result, userPrompt)
		}
	}

	// Either the call or the parse failed. Park the job so the next run can
	// retry it, and record why — an unexplained ScoreFailed row is useless
	// when you come back to it a day later.
	s.log.Warn("marking job ScoreFailed", "job_id", job.ID, "error", err)
	s.LogError(ctx, "WF-B", &job.ID, fmt.Sprintf("scoring failed: %v", err), nil)

	failed := models.StatusScoreFailed
	if _, uerr := s.UpdateJob(ctx, job.ID, models.JobUpdate{Status: &failed}); uerr != nil {
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

	// prompt_used stores the rendered user turn: it is the audit trail for
	// "why did this get 42?", and it is what you diff when a prompt change
	// moves scores.
	audit := prompt

	if _, err := s.UpdateJob(ctx, job.ID, models.JobUpdate{
		Status:        &status,
		Score:         &r.Score,
		MatchedSkills: &matchedRaw,
		MissingSkills: &missingRaw,
		AISummary:     &r.Summary,
		PromptUsed:    &audit,
	}); err != nil {
		return "", fmt.Errorf("write score for job %s: %w", job.ID, err)
	}

	s.log.Info("job scored", "job_id", job.ID, "score", r.Score, "status", status, "title", job.Title)
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
