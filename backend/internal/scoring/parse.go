package scoring

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Result is a validated score for one job.
type Result struct {
	Score         int      `json:"score"`
	MatchedSkills []string `json:"matched_skills"`
	MissingSkills []string `json:"missing_skills"`
	Summary       string   `json:"summary"`
}

// ErrUnparseable means the model's reply was not a usable score. The service
// layer maps it onto the ScoreFailed status, which exists precisely so a bad
// reply parks the job for a retry instead of silently scoring it 0 — a 0 is
// indistinguishable from a genuine "terrible match" once it is in the table.
var ErrUnparseable = errors.New("model reply could not be parsed as a score")

// maxSummary caps the stored summary. The prompt asks for under 300
// characters; this enforces it rather than trusting it.
const maxSummary = 500

// Parse validates a raw model reply.
//
// It is strict on the things that carry meaning — the score must be present
// and in range — and forgiving about packaging, because models wrap JSON in
// prose or a code fence often enough that failing on it would generate
// ScoreFailed rows for replies that are otherwise perfectly good.
func Parse(raw string) (Result, error) {
	body := extractJSONObject(raw)
	if body == "" {
		return Result{}, fmt.Errorf("%w: no JSON object found in reply %q", ErrUnparseable, truncate(raw, 200))
	}

	// A score of 0 and an absent score must stay distinguishable, so decode
	// into a pointer rather than letting the zero value stand in for "missing".
	var wire struct {
		Score         *int     `json:"score"`
		MatchedSkills []string `json:"matched_skills"`
		MissingSkills []string `json:"missing_skills"`
		Summary       string   `json:"summary"`
	}
	if err := json.Unmarshal([]byte(body), &wire); err != nil {
		return Result{}, fmt.Errorf("%w: %v (reply %q)", ErrUnparseable, err, truncate(body, 200))
	}

	if wire.Score == nil {
		return Result{}, fmt.Errorf("%w: reply has no \"score\" field", ErrUnparseable)
	}
	if *wire.Score < 0 || *wire.Score > 100 {
		// Out of range means the model ignored the contract. Clamping would
		// hide that; the DB CHECK would reject it anyway.
		return Result{}, fmt.Errorf("%w: score %d is outside 0-100", ErrUnparseable, *wire.Score)
	}

	return Result{
		Score:         *wire.Score,
		MatchedSkills: cleanList(wire.MatchedSkills),
		MissingSkills: cleanList(wire.MissingSkills),
		Summary:       truncate(strings.TrimSpace(wire.Summary), maxSummary),
	}, nil
}

// extractJSONObject pulls the outermost {...} out of a reply, tolerating a
// markdown fence or a sentence of preamble around it.
//
// It brace-counts rather than regexing, and skips over string literals, so a
// brace inside the summary text ("we use {} syntax") cannot truncate the
// object early.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

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
			// Braces inside a string literal are data, not structure.
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return "" // unbalanced — treat as unparseable
}

// cleanList trims and drops empties, and guarantees a non-nil slice so the
// JSONB column gets [] rather than null.
func cleanList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
