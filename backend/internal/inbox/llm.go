package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// llmSystemPrompt asks for a classification and nothing else.
//
// The instruction about mixed messages is the important one, and it mirrors the
// keyword rules deliberately: the two classifiers must agree on what matters,
// or escalating from one to the other would change the answer in ways nobody
// can predict.
const llmSystemPrompt = `You classify a single email about a job application.

Return ONLY a JSON object, no prose and no code fence:

{"kind": "<one of: acknowledgement, rejection, interview, offer, other>",
 "confidence": <integer 0-100>,
 "evidence": "<the phrase that decided it, under 120 characters>"}

Definitions:
- acknowledgement: confirms the application was received or is under review.
- rejection: the candidate will not proceed.
- interview: an invitation to interview, a call, an assessment, or any
  advancement to a further stage.
- offer: an offer of employment.
- other: anything else, including unrelated mail, newsletters, and automated
  notifications that say nothing about the outcome.

Critical rule: if the email contains BOTH disappointing wording and an
invitation to continue — for example "unfortunately we cannot match your salary
expectations, but we would like to invite you to interview" — classify it as
interview, NOT rejection. Reading such an email as a rejection would make the
candidate abandon a live opportunity, which is the most costly mistake you can
make here. An offer outranks everything.

Be conservative with confidence. Below 60 means you are genuinely unsure, and a
human will review it — which is the correct outcome when the email is unclear.`

// ClassifyLLM asks a model when the keyword rules were not confident enough.
//
// Only the subject and a bounded slice of the body are sent: enough to judge
// the outcome, without shipping an entire thread of someone else's
// correspondence to a third party on every scan.
func ClassifyLLM(ctx context.Context, client scoring.Client, model string, msg Message) (Classification, error) {
	if client == nil {
		return Classification{}, fmt.Errorf("no model client configured")
	}

	user := "Subject: " + strings.TrimSpace(msg.Subject) + "\n\nBody:\n" + trim(msg.Body, 2000)

	reply, err := client.Complete(ctx, model, llmSystemPrompt, user)
	if err != nil {
		return Classification{}, err
	}
	return parseLLMReply(reply, model)
}

// parseLLMReply validates the model's answer. Split out so it can be tested
// without a network.
func parseLLMReply(reply, model string) (Classification, error) {
	body := extractJSONObject(reply)
	if body == "" {
		return Classification{}, fmt.Errorf("no JSON object in reply %q", trim(reply, 120))
	}

	var wire struct {
		Kind       string `json:"kind"`
		Confidence *int   `json:"confidence"`
		Evidence   string `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(body), &wire); err != nil {
		return Classification{}, fmt.Errorf("decode reply: %w", err)
	}

	kind := Kind(strings.ToLower(strings.TrimSpace(wire.Kind)))
	switch kind {
	case KindAcknowledgement, KindRejection, KindInterview, KindOffer, KindOther:
	default:
		// An unrecognised label must not be coerced into a real category —
		// guessing here could invent a rejection. "other" implies no status
		// change, which is the safe landing place.
		return Classification{
			Kind: KindOther, Confidence: 0, Classifier: "llm:" + model,
			Evidence: fmt.Sprintf("model returned an unknown kind %q", trim(wire.Kind, 40)),
		}, nil
	}

	conf := 0
	if wire.Confidence != nil {
		conf = *wire.Confidence
	}
	if conf < 0 {
		conf = 0
	}
	if conf > 100 {
		conf = 100
	}

	return Classification{
		Kind:       kind,
		Confidence: conf,
		Classifier: "llm:" + model,
		Evidence:   trim(wire.Evidence, 120),
	}, nil
}

// extractJSONObject pulls the outermost {...} out of a reply, brace-counting
// with string awareness so a brace inside the evidence text cannot truncate it.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth, inString, escaped := 0, false, false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		switch {
		case c == '\\' && inString:
			escaped = true
		case c == '"':
			inString = !inString
		case inString:
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// Classify runs the cheap path first and escalates only when it must.
//
// Most recruiting mail is formulaic enough for the rules, so the model is a
// fallback rather than the default — which keeps a per-day free quota available
// for generation, where there is no free alternative.
func Classify(ctx context.Context, client scoring.Client, model string, msg Message) Classification {
	kw := ClassifyKeywords(msg)
	if !NeedsLLM(kw) {
		return kw
	}
	if client == nil {
		// No model available: return the weak keyword verdict as-is. Its low
		// confidence is what routes it to a human.
		return kw
	}

	llm, err := ClassifyLLM(ctx, client, model, msg)
	if err != nil {
		kw.Evidence = strings.TrimSpace(kw.Evidence + " (model escalation failed: " + trim(err.Error(), 80) + ")")
		return kw
	}
	return llm
}
