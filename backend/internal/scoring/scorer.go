package scoring

import (
	"context"
	"fmt"

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

// Provider selects which LLM vendor backs ModeLLM. Read from LLM_PROVIDER.
type Provider string

const (
	// ProviderGemini is Google's Gemini API. Its free tier covers this
	// workload, which makes it the practical default for a personal pipeline.
	ProviderGemini Provider = "gemini"
	// ProviderAnthropic is the Claude API. Note a Claude Pro subscription does
	// NOT include API access — that is billed separately, in prepaid credits.
	ProviderAnthropic Provider = "anthropic"
)

// NewLLMClient builds the client for a provider, with that provider's default
// model when none is configured.
func NewLLMClient(provider Provider, apiKey, model string) (Client, string, error) {
	switch provider {
	case ProviderAnthropic:
		if model == "" {
			model = DefaultModel
		}
		c, err := NewAnthropicClient(apiKey, model)
		if err != nil {
			return nil, "", err
		}
		return c, c.Model(), nil
	case ProviderGemini, "":
		c, err := NewGeminiClient(apiKey, model)
		if err != nil {
			return nil, "", err
		}
		return c, c.Model(), nil
	default:
		return nil, "", fmt.Errorf("unknown LLM_PROVIDER %q (want \"gemini\" or \"anthropic\")", provider)
	}
}

// Mode selects a scorer. It is read from SCORING_MODE.
type Mode string

const (
	// ModeKeyword is the default: deterministic overlap scoring, no API key.
	ModeKeyword Mode = "keyword"
	// ModeLLM asks a model. Requires LLM_API_KEY.
	ModeLLM Mode = "llm"
)
