// Package generate produces a tailored CV and cover letter for a scored job.
//
// It reuses the model client from internal/scoring rather than opening its own,
// so there is one provider, one key, and one place that talks to an LLM. What
// differs is the prompt and the parsing: scoring wants a strict JSON object,
// generation wants prose.
package generate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// Documents is what one generation run produces for a job.
type Documents struct {
	CV          string
	CoverLetter string
	Model       string
}

// ErrEmptyOutput means the model returned nothing usable. The job stays in
// Scored so a later run retries it, rather than advancing to approval with a
// blank document.
var ErrEmptyOutput = errors.New("model returned no usable document")

// Section markers. Two documents come back from one call — halving the request
// count matters on a free tier metered per day — so the reply is split on
// these rather than parsed as JSON, which would mangle the formatting of prose.
const (
	cvMarker     = "===CV==="
	letterMarker = "===COVER_LETTER==="
)

const systemPrompt = `You tailor a candidate's CV and write a cover letter for one specific job.

Output EXACTLY this structure, with both markers on their own lines and nothing
before, after, or between them except the documents themselves:

` + cvMarker + `
<the tailored CV, in plain markdown>
` + letterMarker + `
<the cover letter, in plain markdown>

Rules that matter:
- Use ONLY facts present in the candidate's master CV. Do not invent employers,
  dates, qualifications, or numbers. If the job asks for something the
  candidate cannot evidence, leave it out rather than implying it.
- Reorder and re-emphasise honestly: lead with the experience most relevant to
  this posting, and cut what is irrelevant to it. That is tailoring; adding
  things is fabrication, and it is the candidate who has to defend it in an
  interview.
- The cover letter is at most four short paragraphs, addressed to the hiring
  team, naming the role and the organisation. No "Dear Sir/Madam", no restating
  the whole CV, no salary talk.
- Plain markdown only: headings, bullets, bold. No tables, no HTML, no code
  fences around the whole document.
- Write in the first person, in a plain professional register. Avoid
  "passionate", "synergy", "dynamic", and similar filler.`

// BuildPrompt renders the user turn for one job.
func BuildPrompt(p models.Profile, j models.Job) string {
	var b strings.Builder

	b.WriteString("=== CANDIDATE MASTER CV ===\n")
	b.WriteString(strings.TrimSpace(p.MasterCV))
	b.WriteString("\n\n=== TARGET JOB ===\n")

	writeField(&b, "Title", j.Title)
	writeField(&b, "Organization", j.Organization)
	writeField(&b, "Location", strings.TrimSpace(j.LocationRaw+" "+j.Country))
	writeField(&b, "Work arrangement", j.WorkArrangement)
	writeField(&b, "Employment type", j.EmploymentType)
	writeField(&b, "Seniority", j.Seniority)
	writeField(&b, "Required skills", strings.Join(jsonList(j.AIKeySkills), ", "))
	writeField(&b, "Keywords", strings.Join(jsonList(j.AIKeywords), ", "))
	writeField(&b, "Requirements", j.AIRequirementsSummary)
	writeField(&b, "Responsibilities", j.AICoreResponsibilities)

	// The scoring stage already worked out where the candidate is strong and
	// weak against this posting. Passing that through means the letter can lead
	// with the matched skills instead of the model re-deriving the comparison.
	if matched := jsonList(j.MatchedSkills); len(matched) > 0 {
		writeField(&b, "Candidate already evidences", strings.Join(matched, ", "))
	}
	if missing := jsonList(j.MissingSkills); len(missing) > 0 {
		b.WriteString("Not evidenced by the CV (do NOT claim these): " +
			strings.Join(missing, ", ") + "\n")
	}

	if notes := strings.TrimSpace(p.CoverLetterNotes); notes != "" {
		b.WriteString("\n=== CANDIDATE'S INSTRUCTIONS (follow these) ===\n")
		b.WriteString(notes)
		b.WriteString("\n")
	}

	b.WriteString("\nProduce the tailored CV and the cover letter now, using the exact markers.")
	return b.String()
}

// Generate calls the model and splits the reply into the two documents.
func Generate(ctx context.Context, client scoring.Client, model string, p models.Profile, j models.Job) (Documents, error) {
	reply, err := client.Complete(ctx, model, systemPrompt, BuildPrompt(p, j))
	if err != nil {
		return Documents{}, err
	}

	cv, letter, err := Split(reply)
	if err != nil {
		return Documents{}, err
	}
	return Documents{CV: cv, CoverLetter: letter, Model: model}, nil
}

// Split pulls the two documents out of a reply.
//
// It is lenient about a missing CV marker (some models start straight into the
// document) but strict about there being real content in both — an empty
// cover letter advancing to the approval screen is worse than a retry.
func Split(reply string) (cv, letter string, err error) {
	text := strings.TrimSpace(reply)
	if text == "" {
		return "", "", fmt.Errorf("%w: reply was empty", ErrEmptyOutput)
	}

	li := strings.Index(text, letterMarker)
	if li < 0 {
		return "", "", fmt.Errorf("%w: no %s marker in the reply", ErrEmptyOutput, letterMarker)
	}

	cvPart := text[:li]
	letter = strings.TrimSpace(text[li+len(letterMarker):])

	// The CV marker is optional: if it is there, drop everything before it,
	// which also strips any preamble the model added.
	if ci := strings.Index(cvPart, cvMarker); ci >= 0 {
		cvPart = cvPart[ci+len(cvMarker):]
	}
	cv = strings.TrimSpace(stripOuterFence(cvPart))
	letter = strings.TrimSpace(stripOuterFence(letter))

	if cv == "" {
		return "", "", fmt.Errorf("%w: the CV section was empty", ErrEmptyOutput)
	}
	if letter == "" {
		return "", "", fmt.Errorf("%w: the cover letter section was empty", ErrEmptyOutput)
	}
	return cv, letter, nil
}

// stripOuterFence removes a markdown code fence wrapping a whole document.
// Models add one despite being told not to, and it would otherwise render as
// literal backticks in the app.
func stripOuterFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	return strings.TrimSuffix(strings.TrimSpace(s), "```")
}

func writeField(b *strings.Builder, label, value string) {
	if value = strings.TrimSpace(value); value != "" {
		fmt.Fprintf(b, "%s: %s\n", label, value)
	}
}

// jsonList decodes a JSONB string array, tolerating null and malformed values.
func jsonList(raw []byte) []string {
	return scoring.JSONList(raw)
}
