package scoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeminiClient calls Google's Gemini API.
//
// It is an alternative to AnthropicClient behind the same Client interface, so
// switching providers is config rather than code. Gemini's free tier covers
// this workload comfortably — a few dozen scoring calls a day sits well inside
// it — which is why it is a reasonable default for a personal pipeline.
type GeminiClient struct {
	apiKey string
	model  string
	http   *http.Client

	// mu + lastCall implement the pacing. Scoring is sequential today, but a
	// mutex means adding concurrency later cannot accidentally defeat it.
	mu       sync.Mutex
	lastCall time.Time
}

// wait blocks until MinInterval has elapsed since the previous call.
func (g *GeminiClient) wait(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if gap := time.Since(g.lastCall); gap < MinInterval && !g.lastCall.IsZero() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(MinInterval - gap):
		}
	}
	g.lastCall = time.Now()
	return nil
}

// MinInterval paces requests. The Gemini free tier allows roughly 15 requests
// per minute on flash models; scoring 12 jobs back to back overruns that and
// every call after the limit fails. 4.5s between calls keeps a batch inside the
// allowance without needing a retry loop.
const MinInterval = 4500 * time.Millisecond

// DefaultGeminiModel is the flash tier: fast and cheap, which is what a
// high-volume classification over short pre-extracted inputs wants.
//
// The "-latest" alias tracks Google's current flash release rather than
// pinning. Pin an explicit id (e.g. gemini-3.6-flash) via LLM_MODEL if you
// need scores to stay reproducible across model updates.
const DefaultGeminiModel = "gemini-flash-latest"

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

// NewGeminiClient builds the client, rejecting an unfilled key at boot rather
// than at 07:00 on the first scheduled run.
func NewGeminiClient(apiKey, model string) (*GeminiClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is not set")
	}
	if placeholders[strings.ToLower(apiKey)] {
		return nil, fmt.Errorf("LLM_API_KEY is still the placeholder %q", apiKey)
	}

	model = strings.TrimSpace(model)
	// "gemini-models" is not a model id — it is what you get from skim-reading
	// the console's model list page. Catching it here saves a confusing 404.
	if model == "" || model == "gemini-models" || placeholders[strings.ToLower(model)] {
		model = DefaultGeminiModel
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		// Generous: scoring is sequential and a flash model on a cold start can
		// take a few seconds. The batch-level timeout is the real bound.
		http: &http.Client{Timeout: 90 * time.Second},
	}, nil
}

// Model reports the model id in use, for the audit trail.
func (g *GeminiClient) Model() string { return g.model }

// responseSchema constrains the reply to exactly the shape Parse expects.
//
// This is the main practical advantage of Gemini here: the JSON is enforced
// server-side, so the markdown-fence and prose-preamble failures that Parse
// has to tolerate from a free-form model largely stop happening — and
// ScoreFailed becomes genuinely rare rather than routine.
var responseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"score":          map[string]any{"type": "INTEGER"},
		"matched_skills": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
		"missing_skills": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
		"summary":        map[string]any{"type": "STRING"},
	},
	"required": []string{"score", "matched_skills", "missing_skills", "summary"},
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  map[string]any  `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Complete sends one scoring request and returns the reply text.
func (g *GeminiClient) Complete(ctx context.Context, system, user string) (string, error) {
	if err := g.wait(ctx); err != nil {
		return "", err
	}

	body, err := json.Marshal(geminiRequest{
		SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: system}}},
		Contents:          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: user}}}},
		GenerationConfig: map[string]any{
			"responseMimeType": "application/json",
			"responseSchema":   responseSchema,
			"temperature":      0,
		},
	})
	if err != nil {
		return "", fmt.Errorf("build gemini request: %w", err)
	}

	url := fmt.Sprintf(geminiEndpoint, g.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build gemini request: %w", err)
	}
	// Header rather than ?key= so the secret never lands in a proxy log.
	req.Header.Set("x-goog-api-key", g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read gemini response: %w", err)
	}

	var out geminiResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode gemini response (http %d): %w", resp.StatusCode, err)
	}

	if out.Error != nil {
		// 429 is the free tier's daily/per-minute quota. Surfacing it plainly
		// matters: it is a wait-and-retry condition, not a broken prompt, and
		// the job should be retried rather than written off.
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", fmt.Errorf("%w: gemini says %s", ErrRateLimited, out.Error.Message)
		}
		return "", fmt.Errorf("gemini error %d: %s", out.Error.Code, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini returned http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	// A safety block arrives as a 200 with no candidates, so it has to be
	// checked before indexing into them.
	if out.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("%w: gemini blocked the prompt (%s)", ErrUnparseable, out.PromptFeedback.BlockReason)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("%w: gemini returned no content", ErrUnparseable)
	}

	var text strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("%w: gemini returned empty text (finish reason %q)",
			ErrUnparseable, out.Candidates[0].FinishReason)
	}
	return text.String(), nil
}
