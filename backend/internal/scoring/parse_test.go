package scoring

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAcceptsACleanReply(t *testing.T) {
	got, err := Parse(`{"score": 82, "matched_skills": ["Flutter","Dart"], "missing_skills": ["Kotlin"], "summary": "Strong mobile match."}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Score != 82 {
		t.Errorf("Score = %d, want 82", got.Score)
	}
	if len(got.MatchedSkills) != 2 || got.MatchedSkills[0] != "Flutter" {
		t.Errorf("MatchedSkills = %v", got.MatchedSkills)
	}
	if got.Summary != "Strong mobile match." {
		t.Errorf("Summary = %q", got.Summary)
	}
}

// Models wrap JSON in fences and preamble often enough that rejecting it would
// manufacture ScoreFailed rows for perfectly good scores.
func TestParseToleratesPackaging(t *testing.T) {
	replies := map[string]string{
		"markdown fence":      "```json\n{\"score\": 70, \"summary\": \"ok\"}\n```",
		"bare fence":          "```\n{\"score\": 70}\n```",
		"leading prose":       "Here is the score you asked for:\n{\"score\": 70}",
		"trailing prose":      "{\"score\": 70}\nLet me know if you want more detail.",
		"surrounded by prose": "Sure!\n{\"score\": 70}\nHope that helps.",
		"leading whitespace":  "\n\n   {\"score\": 70}",
	}
	for name, reply := range replies {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(reply)
			if err != nil {
				t.Fatalf("Parse(%q): %v", reply, err)
			}
			if got.Score != 70 {
				t.Errorf("Score = %d, want 70", got.Score)
			}
		})
	}
}

// A brace inside the summary must not truncate the object — this is why the
// extractor counts braces with string-awareness instead of regexing.
func TestParseHandlesBracesInsideStrings(t *testing.T) {
	got, err := Parse(`{"score": 55, "summary": "Uses {} and {{mustache}} syntax heavily."}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Score != 55 {
		t.Errorf("Score = %d, want 55", got.Score)
	}
	if !strings.Contains(got.Summary, "{{mustache}}") {
		t.Errorf("Summary was truncated: %q", got.Summary)
	}
}

func TestParseHandlesEscapedQuotesInSummary(t *testing.T) {
	got, err := Parse(`{"score": 60, "summary": "Wants a \"senior\" engineer."}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(got.Summary, `"senior"`) {
		t.Errorf("Summary = %q", got.Summary)
	}
}

// Zero is a legitimate score and must not be confused with a missing field —
// which is why the wire struct decodes into *int.
func TestParseDistinguishesZeroFromMissing(t *testing.T) {
	got, err := Parse(`{"score": 0, "summary": "Completely different discipline."}`)
	if err != nil {
		t.Fatalf("Parse of score 0: %v", err)
	}
	if got.Score != 0 {
		t.Errorf("Score = %d, want 0", got.Score)
	}

	if _, err := Parse(`{"summary": "forgot the score"}`); !errors.Is(err, ErrUnparseable) {
		t.Errorf("a reply with no score field must be ErrUnparseable, got %v", err)
	}
}

func TestParseRejectsUnusableReplies(t *testing.T) {
	cases := map[string]string{
		"empty":             "",
		"prose only":        "I think this is a pretty good match, maybe 80 out of 100.",
		"not an object":     "[1,2,3]",
		"malformed json":    `{"score": 70,}`,
		"unbalanced braces": `{"score": 70`,
		"score above range": `{"score": 150}`,
		"score below range": `{"score": -5}`,
		"score as string":   `{"score": "eighty"}`,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(reply); !errors.Is(err, ErrUnparseable) {
				t.Errorf("Parse(%q) should be ErrUnparseable, got %v", reply, err)
			}
		})
	}
}

// An out-of-range score is a contract violation, not something to clamp:
// silently turning 150 into 100 would hide a broken prompt, and the database
// CHECK would reject it anyway.
func TestParseDoesNotClampOutOfRangeScores(t *testing.T) {
	_, err := Parse(`{"score": 150}`)
	if err == nil {
		t.Fatal("expected an error for score 150")
	}
	if !strings.Contains(err.Error(), "150") {
		t.Errorf("error should name the offending value, got %v", err)
	}
}

func TestParseNormalisesListsForJSONB(t *testing.T) {
	got, err := Parse(`{"score": 50, "matched_skills": ["Flutter", "  ", "", "Dart"]}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.MatchedSkills) != 2 {
		t.Errorf("blank entries were not dropped: %v", got.MatchedSkills)
	}
	// Non-nil so the JSONB column gets [] rather than null.
	if got.MissingSkills == nil {
		t.Error("MissingSkills is nil; the JSONB column would be null instead of []")
	}
}

func TestParseTruncatesAnOverlongSummary(t *testing.T) {
	long := strings.Repeat("x", 2000)
	got, err := Parse(`{"score": 50, "summary": "` + long + `"}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.Summary) > maxSummary+4 {
		t.Errorf("summary not truncated: %d chars", len(got.Summary))
	}
}
