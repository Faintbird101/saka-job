// Package scoring turns a job row plus your profile into a 0-100 match score.
//
// The package is split so the expensive, non-deterministic part (the model
// call) is isolated behind an interface, and everything either side of it —
// building the prompt, validating the reply — is pure and unit-testable
// without a network or an API key.
package scoring

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// systemPrompt fixes the model's role and output contract.
//
// It is a constant, deliberately: it is the stable prefix of every scoring
// request, which is what makes it cacheable, and a scoring rubric that drifts
// per-request would make scores incomparable between jobs.
const systemPrompt = `You score how well a job posting matches a candidate's CV.

Return ONLY a JSON object, with no prose, no explanation, and no markdown code
fence, in exactly this shape:

{"score": <integer 0-100>,
 "matched_skills": [<strings from the job's required skills that the CV evidences>],
 "missing_skills": [<strings from the job's required skills that the CV does not evidence>],
 "summary": "<one or two sentences, under 300 characters, on why this score>"}

Scoring guidance:
- 85-100  strong match: the candidate meets essentially all stated requirements.
- 70-84   good match: meets the core requirements, gaps are secondary.
- 50-69   partial match: some core requirements are unmet.
- 0-49    weak match: most core requirements are unmet, or the role is a
          different discipline than the CV suggests.

Judge only against what the posting actually states. Note that these postings
carry NO full description — you are given the source API's extracted skills,
keywords and requirements summary instead. Do not invent requirements that are
not listed, and do not penalise the candidate for requirements the posting
never states. Every entry in matched_skills and missing_skills must be a skill
the posting itself lists.`

// BuildUserPrompt renders one job and the candidate profile into the user turn.
//
// The job's description_text is deliberately not used: the source API's plan
// returns it as null, so the only substantive signal available is the AI-
// extracted fields it does provide. That is also the cheaper path the README
// argues for — the API has already done the extraction, so we do not pay to
// redo it.
func BuildUserPrompt(p models.Profile, j models.Job) string {
	var b strings.Builder

	b.WriteString("=== CANDIDATE CV ===\n")
	b.WriteString(strings.TrimSpace(p.MasterCV))
	b.WriteString("\n")

	if skills := jsonList(p.PreferredSkills); len(skills) > 0 {
		b.WriteString("\nSkills the candidate especially wants to use: ")
		b.WriteString(strings.Join(skills, ", "))
		b.WriteString("\n")
	}

	b.WriteString("\n=== JOB POSTING ===\n")
	writeField(&b, "Title", j.Title)
	writeField(&b, "Organization", j.Organization)
	writeField(&b, "Location", strings.TrimSpace(j.LocationRaw+" "+j.Country))
	writeField(&b, "Work arrangement", j.WorkArrangement)
	writeField(&b, "Employment type", j.EmploymentType)
	writeField(&b, "Seniority", j.Seniority)
	writeField(&b, "Experience level", j.ExperienceLevel)

	if skills := jsonList(j.AIKeySkills); len(skills) > 0 {
		writeField(&b, "Required skills", strings.Join(skills, ", "))
	}
	if kw := jsonList(j.AIKeywords); len(kw) > 0 {
		writeField(&b, "Keywords", strings.Join(kw, ", "))
	}
	writeField(&b, "Requirements", j.AIRequirementsSummary)
	writeField(&b, "Responsibilities", j.AICoreResponsibilities)

	b.WriteString("\nScore this match and reply with the JSON object only.")
	return b.String()
}

// SystemPrompt exposes the constant so the service layer can record exactly
// what was sent in the job's prompt_used audit column.
func SystemPrompt() string { return systemPrompt }

func writeField(b *strings.Builder, label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		// Omit rather than emit "Seniority: " — a blank label invites the model
		// to speculate about what belongs there.
		return
	}
	fmt.Fprintf(b, "%s: %s\n", label, value)
}

// JSONList decodes a JSONB string array, tolerating null and malformed values
// by returning nothing rather than failing the run. Exported so the generation
// stage can reuse it rather than keeping a second copy in step with this one.
func JSONList(raw json.RawMessage) []string {
	return jsonList(raw)
}

// jsonList decodes a JSONB string array, tolerating null and malformed values
// by returning nothing rather than failing the whole scoring run.
func jsonList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	cleaned := make([]string, 0, len(out))
	for _, s := range out {
		if s = strings.TrimSpace(s); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	return cleaned
}
