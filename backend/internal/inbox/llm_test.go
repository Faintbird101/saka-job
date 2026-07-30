package inbox

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stub struct {
	reply    string
	err      error
	gotModel string
	calls    int
}

func (s *stub) Complete(_ context.Context, model, _, _ string) (string, error) {
	s.calls++
	s.gotModel = model
	return s.reply, s.err
}

func TestParseLLMReply(t *testing.T) {
	c, err := parseLLMReply(`{"kind":"rejection","confidence":88,"evidence":"we regret to inform"}`, "m1")
	if err != nil {
		t.Fatalf("parseLLMReply: %v", err)
	}
	if c.Kind != KindRejection || c.Confidence != 88 {
		t.Errorf("got %+v", c)
	}
	if c.Classifier != "llm:m1" {
		t.Errorf("classifier = %q — the model must be recorded for audit", c.Classifier)
	}
}

func TestParseLLMReplyTolerantOfPackaging(t *testing.T) {
	for _, reply := range []string{
		"```json\n{\"kind\":\"interview\",\"confidence\":90}\n```",
		"Here you go:\n{\"kind\":\"interview\",\"confidence\":90}",
		"{\"kind\":\"interview\",\"confidence\":90}\nHope that helps.",
	} {
		c, err := parseLLMReply(reply, "m")
		if err != nil {
			t.Fatalf("%q: %v", reply, err)
		}
		if c.Kind != KindInterview {
			t.Errorf("%q -> %s", reply, c.Kind)
		}
	}
}

// An unrecognised label must land on "other", never be coerced into a real
// category — guessing here could invent a rejection out of nothing.
func TestParseLLMReplyRejectsUnknownKind(t *testing.T) {
	c, err := parseLLMReply(`{"kind":"maybe_rejection","confidence":99}`, "m")
	if err != nil {
		t.Fatalf("should degrade, not error: %v", err)
	}
	if c.Kind != KindOther {
		t.Errorf("kind = %s, want other", c.Kind)
	}
	if c.Confidence != 0 {
		t.Errorf("confidence = %d, want 0 for an unusable label", c.Confidence)
	}
}

func TestParseLLMReplyClampsConfidence(t *testing.T) {
	for reply, want := range map[string]int{
		`{"kind":"offer","confidence":250}`: 100,
		`{"kind":"offer","confidence":-5}`:  0,
		`{"kind":"offer"}`:                  0,
	} {
		c, err := parseLLMReply(reply, "m")
		if err != nil {
			t.Fatalf("%q: %v", reply, err)
		}
		if c.Confidence != want {
			t.Errorf("%q -> confidence %d, want %d", reply, c.Confidence, want)
		}
	}
}

func TestParseLLMReplyRejectsGarbage(t *testing.T) {
	for _, reply := range []string{"", "no json here", "{broken"} {
		if _, err := parseLLMReply(reply, "m"); err == nil {
			t.Errorf("%q should have errored", reply)
		}
	}
}

// The cheap path must handle the formulaic majority without touching the model
// — that is what keeps the daily free quota available for generation.
func TestClassifyDoesNotCallTheModelWhenKeywordsAreConfident(t *testing.T) {
	c := &stub{reply: `{"kind":"other","confidence":10}`}
	got := Classify(context.Background(), c, "m", Message{
		Subject: "Update", Body: "We regret to inform you that we will not be moving forward.",
	})
	if c.calls != 0 {
		t.Errorf("model was called %d times for an unambiguous rejection", c.calls)
	}
	if got.Kind != KindRejection || got.Classifier != "keyword" {
		t.Errorf("got %+v", got)
	}
}

func TestClassifyEscalatesWhenKeywordsAreWeak(t *testing.T) {
	c := &stub{reply: `{"kind":"rejection","confidence":80,"evidence":"polite brush-off"}`}
	got := Classify(context.Background(), c, "m-escalate", Message{
		Subject: "Thanks", Body: "We will keep your CV on file for future roles.",
	})
	if c.calls != 1 {
		t.Errorf("expected one escalation, got %d calls", c.calls)
	}
	if c.gotModel != "m-escalate" {
		t.Errorf("model = %q", c.gotModel)
	}
	if got.Classifier != "llm:m-escalate" {
		t.Errorf("classifier = %q", got.Classifier)
	}
}

// No model configured is a supported state: the weak keyword verdict stands,
// and its low confidence is what routes it to a human.
func TestClassifyWithoutAModelFallsBackToKeywords(t *testing.T) {
	got := Classify(context.Background(), nil, "", Message{Subject: "Hi", Body: "Unrelated."})
	if got.Classifier != "keyword" {
		t.Errorf("classifier = %q", got.Classifier)
	}
	if !NeedsLLM(got) {
		t.Error("a verdict this weak must still read as needing review")
	}
}

// A provider failure must not lose the message — the keyword verdict survives
// with the failure noted.
func TestClassifySurvivesModelFailure(t *testing.T) {
	c := &stub{err: errors.New("quota exceeded")}
	got := Classify(context.Background(), c, "m", Message{Subject: "?", Body: "Ambiguous text."})
	if got.Classifier != "keyword" {
		t.Errorf("classifier = %q, want the keyword fallback", got.Classifier)
	}
	if !strings.Contains(got.Evidence, "escalation failed") {
		t.Errorf("the failure should be recorded in the evidence, got %q", got.Evidence)
	}
}

func TestLLMPromptCarriesTheMixedMessageRule(t *testing.T) {
	// The two classifiers must agree on the case that matters, or escalating
	// between them would change the answer unpredictably.
	for _, want := range []string{"interview, NOT rejection", "abandon a live opportunity"} {
		if !strings.Contains(llmSystemPrompt, want) {
			t.Errorf("llm prompt is missing %q", want)
		}
	}
}
