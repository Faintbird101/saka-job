package scoring

import (
	"context"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// Scorer turns a profile and a job into a Result.
//
// Two implementations exist and they are interchangeable by design:
//
//   - KeywordScorer  deterministic, free, no network
//   - LLMScorer      a model call, better judgement, costs money
//
// The service layer holds this interface rather than either concrete type, so
// switching is one env var and a restart — no code change, and the state
// machine (Scored / LowMatch / ScoreFailed) is identical either way.
type Scorer interface {
	// Score returns a validated result, or an error. An error parks the job in
	// ScoreFailed for a later run rather than recording a misleading zero.
	Score(ctx context.Context, p models.Profile, j models.Job) (Result, error)

	// Name identifies the scorer in logs and in the run summary, so a score in
	// the table can be traced back to what produced it.
	Name() string
}

// Mode selects a scorer. It is read from SCORING_MODE.
type Mode string

const (
	// ModeKeyword is the default: deterministic overlap scoring, no API key.
	ModeKeyword Mode = "keyword"
	// ModeLLM asks a model. Requires LLM_API_KEY.
	ModeLLM Mode = "llm"
)
