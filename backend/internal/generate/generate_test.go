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
	cv, letter, _, err := Split("===CV===\n# Victor\nFlutter dev.\n===COVER_LETTER===\nDear WorkBuddy team,\nI'd like to apply.")
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
			cv, letter, _, err := Split(reply)
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
			if _, _, _, err := Split(reply); !errors.Is(err, ErrEmptyOutput) {
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

func TestSplitParsesTheEditList(t *testing.T) {
	reply := `===CV===
CV body
===COVER_LETTER===
Letter body
===EDITS===
[{"section":"Summary","before":"Backend engineer with 7 years",
  "after":"Backend engineer building payment systems","reason":"matches payments"}]`

	cv, letter, edits, err := Split(reply)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if !strings.Contains(cv, "CV body") || strings.Contains(cv, "Summary") {
		t.Errorf("the edits section leaked into the CV: %q", cv)
	}
	// The letter must stop at the marker, or the JSON renders as prose.
	if strings.Contains(letter, "===EDITS===") || strings.Contains(letter, "section") {
		t.Errorf("the edits section leaked into the letter: %q", letter)
	}
	if len(edits) != 1 {
		t.Fatalf("got %d edits, want 1", len(edits))
	}
	if edits[0].Section != "Summary" || edits[0].Reason != "matches payments" {
		t.Errorf("edit parsed wrong: %+v", edits[0])
	}
}

// The documents matter more than the annotations. A model that omits or
// mangles the trailing array must not cost the CV and letter.
func TestSplitSurvivesAMissingOrBrokenEditList(t *testing.T) {
	cases := map[string]string{
		"no edits section": "===CV===\nCV body\n===COVER_LETTER===\nLetter body",
		"empty array":      "===CV===\nCV body\n===COVER_LETTER===\nLetter body\n===EDITS===\n[]",
		"malformed json":   "===CV===\nCV body\n===COVER_LETTER===\nLetter body\n===EDITS===\n[{broken",
		"prose instead":    "===CV===\nCV body\n===COVER_LETTER===\nLetter body\n===EDITS===\nI made a few changes.",
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			cv, letter, edits, err := Split(reply)
			if err != nil {
				t.Fatalf("documents were lost over the edit list: %v", err)
			}
			if cv == "" || letter == "" {
				t.Error("a document came back empty")
			}
			if edits != nil && len(edits) != 0 {
				t.Errorf("expected no usable edits, got %v", edits)
			}
		})
	}
}

func TestSplitToleratesAFencedEditList(t *testing.T) {
	reply := "===CV===\nCV body\n===COVER_LETTER===\nLetter body\n===EDITS===\n" +
		"```json\n[{\"section\":\"Skills\",\"before\":\"a, b\",\"after\":\"b, a\",\"reason\":\"their order\"}]\n```"

	_, _, edits, err := Split(reply)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(edits) != 1 || edits[0].Section != "Skills" {
		t.Errorf("fenced edit list not parsed: %v", edits)
	}
}

// An entry quoting neither document is worse than no entry, because the
// candidate will check. Empty-on-both is dropped; one side empty is a real
// edit meaning text was cut or added.
func TestParseEditsDropsEmptyEntriesButKeepsCuts(t *testing.T) {
	edits := parseEdits(`[
	  {"section":"A","before":"","after":"","reason":"nothing"},
	  {"section":"Trimmed","before":"an old role","after":"","reason":"one page"}
	]`)
	if len(edits) != 1 {
		t.Fatalf("got %d edits, want 1", len(edits))
	}
	if edits[0].Section != "Trimmed" || edits[0].After != "" {
		t.Errorf("the cut was not preserved: %+v", edits[0])
	}
}

func TestSystemPromptAsksForHonestEdits(t *testing.T) {
	for _, want := range []string{"Only real changes", "Do not invent edits"} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("system prompt is missing %q", want)
		}
	}
}

// Regression: on the first live run the model reported an "after" of
// "Databases: MongoDb, Firestore" that appeared nowhere in the CV it had just
// written. A diff the candidate cannot reconcile with the document teaches
// them the annotations are decorative, which is worse than showing none.
func TestValidateEditsDropsFabricatedChanges(t *testing.T) {
	master := "Database: MongoDb, Firestore\nFour years of Flutter."
	tailored := "# Victor\nDatabase: MongoDb, Firestore\nFour years of Flutter."

	edits := []Edit{
		{Section: "Skills", Before: "Database: MongoDb, Firestore",
			After: "Databases: MongoDb, Firestore", Reason: "pluralised"},
		{Section: "Summary", Before: "", After: "Four years of Flutter.",
			Reason: "leads with the match"},
	}

	kept := ValidateEdits(edits, master, tailored)
	if len(kept) != 1 {
		t.Fatalf("kept %d edits, want 1 — the fabricated one should be dropped", len(kept))
	}
	if kept[0].Section != "Summary" {
		t.Errorf("dropped the wrong edit: %+v", kept[0])
	}
}

// Markdown emphasis and whitespace differ constantly between what a model
// reports and what it wrote. Rejecting on that would discard honest edits.
func TestValidateEditsIgnoresFormatting(t *testing.T) {
	master := "Backend engineer with 7 years across fintech"
	tailored := "## Summary\n\n**Backend engineer**   building   payment systems"

	kept := ValidateEdits([]Edit{{
		Section: "Summary",
		Before:  "Backend engineer with 7 years across fintech",
		After:   "Backend engineer building payment systems",
	}}, master, tailored)

	if len(kept) != 1 {
		t.Error("a formatting difference caused an honest edit to be dropped")
	}
}

func TestValidateEditsKeepsCutsAndAdditions(t *testing.T) {
	master := "An old role nobody needs.\nFlutter work."
	tailored := "Flutter work.\nNew line added."

	kept := ValidateEdits([]Edit{
		{Section: "Trimmed", Before: "An old role nobody needs.", After: ""},
		{Section: "Added", Before: "", After: "New line added."},
	}, master, tailored)

	if len(kept) != 2 {
		t.Errorf("cuts and additions should both survive, kept %d", len(kept))
	}
}

func TestValidateEditsHandlesEmptyInput(t *testing.T) {
	if got := ValidateEdits(nil, "a", "b"); got != nil {
		t.Errorf("nil in should give nil out, got %v", got)
	}
	// Everything fabricated means nothing to show, not an empty list to render.
	if got := ValidateEdits([]Edit{{Before: "nowhere", After: "nowhere either"}}, "a", "b"); got != nil {
		t.Errorf("all-fabricated should give nil, got %v", got)
	}
}
