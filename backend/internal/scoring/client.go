package scoring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client is the model call, behind an interface.
//
// Everything else in this package is pure, so this is the single seam where
// the network lives — which is what lets the prompt and parsing logic be
// tested without an API key.
type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// DefaultModel is used when LLM_MODEL is unset.
//
// Sonnet tier rather than Opus: scoring is a high-volume classification over
// short, pre-extracted inputs, run on every ingested job twice a day. Opus
// would cost several times more for a judgement this well-bounded.
const DefaultModel = "claude-sonnet-5"

// maxTokens is generous for what is a small JSON object, because on models
// where adaptive thinking is on by default the reply and the thinking share
// this budget — too tight a cap truncates the JSON and manufactures
// ScoreFailed rows.
const maxTokens = 4096

// AnthropicClient calls the Claude API.
type AnthropicClient struct {
	client anthropic.Client
	model  string
}

// placeholders are the values .env.example ships with. Treating them as
// "unset" is what stops the backend reporting "scoring enabled" for a key that
// is going to 401 on the first scheduled run at 07:00 — the failure belongs at
// boot, where you are looking.
var placeholders = map[string]bool{
	"your_llm_key":      true,
	"your_new_key_here": true,
	"change_me":         true,
	"changeme":          true,
	"todo":              true,
}

// NewAnthropicClient builds the client. It fails fast on a missing or
// obviously-unfilled key rather than deferring to a 401 on the first run.
func NewAnthropicClient(apiKey, model string) (*AnthropicClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("LLM_API_KEY is not set")
	}
	if placeholders[strings.ToLower(apiKey)] {
		return nil, fmt.Errorf("LLM_API_KEY is still the placeholder %q from .env.example", apiKey)
	}
	if model = strings.TrimSpace(model); model == "" {
		model = DefaultModel
	}
	return &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}, nil
}

// Model reports the model id in use, for the audit trail.
func (c *AnthropicClient) Model() string { return c.model }

// Complete sends one scoring request and returns the reply text.
func (c *AnthropicClient) Complete(ctx context.Context, system, user string) (string, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{{
			Text: system,
			// The system prompt is identical on every scoring call, so caching
			// it means a batch of 20 jobs pays for the rubric once.
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("model request failed: %w", err)
	}

	// A safety refusal is not a transport error — it arrives as a normal 200
	// with no usable content, so it has to be checked before reading blocks.
	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("%w: the model declined to score this posting", ErrUnparseable)
	}

	var out strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			out.WriteString(text.Text)
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("%w: model returned no text (stop_reason %q)", ErrUnparseable, resp.StopReason)
	}
	return out.String(), nil
}
