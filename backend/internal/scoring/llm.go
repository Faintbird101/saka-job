package scoring

import (
	"context"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// LLMScorer asks a model. It is the higher-quality, non-free path: unlike the
// keyword scorer it reads the requirements summary as prose, so it can weigh
// "3 years of Flutter" against "2+ years required" and recognise that a
// "Mobile Engineer" posting is really a Flutter role.
type LLMScorer struct {
	client Client
	model  string
}

// NewLLMScorer wraps a model client in the Scorer interface.
func NewLLMScorer(c Client, model string) *LLMScorer {
	return &LLMScorer{client: c, model: model}
}

// Name identifies the scorer, including which model produced the score — so a
// row in the table can be traced to the exact thing that judged it.
func (l *LLMScorer) Name() string { return "llm:" + l.model }

// Score builds the prompt, calls the model, and validates the reply.
func (l *LLMScorer) Score(ctx context.Context, p models.Profile, j models.Job) (Result, error) {
	reply, err := l.client.Complete(ctx, SystemPrompt(), BuildUserPrompt(p, j))
	if err != nil {
		return Result{}, err
	}
	return Parse(reply)
}

// Prompt exposes the rendered user turn for the prompt_used audit column.
// The keyword scorer has no equivalent, which is why the service layer records
// this separately rather than assuming every scorer has a prompt.
func (l *LLMScorer) Prompt(p models.Profile, j models.Job) string {
	return BuildUserPrompt(p, j)
}
