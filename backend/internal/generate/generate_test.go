package generate

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// stubClient returns a canned reply, and records what it was asked.
type stubClient struct {
	reply     string
	err       error
	gotModel  string
	gotSystem string
	gotUser   string
}

func (s *stubClient) Complete(_ context.Context, model, system, user string) (string, error) {
	s.gotModel, s.gotSystem, s.gotUser = model, system, user
	return s.reply, s.err
}

func testProfile() models.Profile {
	return models.Profile{
		MasterCV:        "Victor Kinyua. Four years of Flutter and Dart. Backend in Go.",
		PreferredSkills: json.RawMessage(`["Flutter","Go"]`),
	}
}

func testJob() models.Job {
	return models.Job{
		ID:            "job-1",
		Title:         "Flutter Product Engineer",
		Organization:  "WorkBuddy",
		AIKeySkills:   json.RawMessage(`["Flutter","Dart","Kotlin"]`),
		MatchedSkills: json.RawMessage(`["Flutter","Dart"]`),
		MissingSkills: json.RawMessage(`["Kotlin"]`),
	}
}

func TestSplitExtractsBothDocuments(t *testing.T) {
	cv, letter, err := Split("===CV===\n# Victor\nFlutter dev.\n===COVER_LETTER===\nDear WorkBuddy team,\nI'd like to apply.")
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if !strings.Contains(cv, "# Victor") || strings.Contains(cv, "Dear WorkBuddy") {
		t.Errorf("CV section wrong: %q", cv)
	}
	if !strings.Contains(letter, "Dear WorkBuddy") || strings.Contains(letter, "# Victor") {
		t.Errorf("letter section wrong: %q", letter)
	}
}

// Models add preamble and fences despite instructions. Rejecting those would
// throw away perfectly good documents.
func TestSplitTolerantOfPackaging(t *testing.T) {
	cases := map[string]string{
		"preamble before the marker": "Sure! Here you go:\n===CV===\nCV body\n===COVER_LETTER===\nLetter body",
		"missing CV marker":          "CV body\n===COVER_LETTER===\nLetter body",
		"fenced sections":            "===CV===\n```markdown\nCV body\n```\n===COVER_LETTER===\n```\nLetter body\n```",
		"extra blank lines":          "===CV===\n\n\nCV body\n\n\n===COVER_LETTER===\n\nLetter body\n\n",
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			cv, letter, err := Split(reply)
			if err != nil {
				t.Fatalf("Split: %v", err)
			}
			if !strings.Contains(cv, "CV body") {
				t.Errorf("cv = %q", cv)
			}
			if !strings.Contains(letter, "Letter body") {
				t.Errorf("letter = %q", letter)
			}
			if strings.Contains(cv, "```") || strings.Contains(letter, "```") {
				t.Error("code fence survived; it would render as literal backticks")
			}
		})
	}
}

// A blank document reaching the approval screen is worse than a retry — the
// job stays in Scored and a later run tries again.
func TestSplitRejectsIncompleteOutput(t *testing.T) {
	cases := map[string]string{
		"empty reply":      "",
		"whitespace only":  "   \n\n  ",
		"no letter marker": "===CV===\nJust a CV, no letter.",
		"empty letter":     "===CV===\nCV body\n===COVER_LETTER===\n   ",
		"empty cv":         "===CV===\n\n===COVER_LETTER===\nLetter body",
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Split(reply); !errors.Is(err, ErrEmptyOutput) {
				t.Errorf("want ErrEmptyOutput, got %v", err)
			}
		})
	}
}

func TestBuildPromptIncludesJobAndCV(t *testing.T) {
	got := BuildPrompt(testProfile(), testJob())
	for _, want := range []string{"Victor Kinyua", "Flutter Product Engineer", "WorkBuddy", "Flutter, Dart, Kotlin"} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

// The scoring stage already worked out the gaps; passing them through is what
// stops the letter claiming a skill the CV cannot support.
func TestBuildPromptWarnsAgainstClaimingMissingSkills(t *testing.T) {
	got := BuildPrompt(testProfile(), testJob())
	if !strings.Contains(got, "do NOT claim") || !strings.Contains(got, "Kotlin") {
		t.Errorf("prompt does not carry the missing-skills warning:\n%s", got)
	}
	if !strings.Contains(got, "Candidate already evidences") {
		t.Error("prompt does not carry the matched skills")
	}
}

func TestBuildPromptIncludesUserNotes(t *testing.T) {
	p := testProfile()
	p.CoverLetterNotes = "Mention that I am available immediately."
	if got := BuildPrompt(p, testJob()); !strings.Contains(got, "available immediately") {
		t.Error("cover_letter_notes did not reach the prompt")
	}
}

// The honesty rule is the whole reason a human still approves each application.
func TestSystemPromptForbidsFabrication(t *testing.T) {
	for _, want := range []string{"Do not invent", "ONLY facts", "fabrication"} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("system prompt is missing the anti-fabrication rule %q", want)
		}
	}
}

func TestGenerateUsesTheRequestedModel(t *testing.T) {
	c := &stubClient{reply: "===CV===\nCV body\n===COVER_LETTER===\nLetter body"}

	docs, err := Generate(context.Background(), c, "gemini-3.6-flash", testProfile(), testJob())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if c.gotModel != "gemini-3.6-flash" {
		t.Errorf("model = %q, want the per-stage override to be passed through", c.gotModel)
	}
	if docs.Model != "gemini-3.6-flash" {
		t.Errorf("Documents.Model = %q — it is the audit trail for what wrote this", docs.Model)
	}
	if docs.CV == "" || docs.CoverLetter == "" {
		t.Error("documents came back empty")
	}
}

func TestGeneratePropagatesClientErrors(t *testing.T) {
	sentinel := errors.New("provider exploded")
	c := &stubClient{err: sentinel}

	if _, err := Generate(context.Background(), c, "m", testProfile(), testJob()); !errors.Is(err, sentinel) {
		t.Errorf("client error was not propagated: %v", err)
	}
}
