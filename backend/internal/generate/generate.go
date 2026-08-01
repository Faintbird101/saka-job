// Package generate produces a tailored CV and cover letter for a scored job.
//
// It reuses the model client from internal/scoring rather than opening its own,
// so there is one provider, one key, and one place that talks to an LLM. What
// differs is the prompt and the parsing: scoring wants a strict JSON object,
// generation wants prose.
package generate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/scoring"
)

// Edit is one change made to the master CV, with its reason.
//
// The deck's argument: no CV edit is silent. Struck-through before,
// highlighted after, and why it changed — so the candidate can check the work
// instead of trusting it.
type Edit struct {
	Section string `json:"section"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Reason  string `json:"reason"`
}

// Documents is what one generation run produces for a job.
type Documents struct {
	CV          string
	CoverLetter string
	Edits       []Edit
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
	editsMarker  = "===EDITS==="
)

const systemPrompt = `You tailor a candidate's CV and write a cover letter for one specific job.

Output EXACTLY this structure, with both markers on their own lines and nothing
before, after, or between them except the documents themselves:

` + cvMarker + `
<the tailored CV, in plain markdown>
` + letterMarker + `
<the cover letter, in plain markdown>
` + editsMarker + `
<a JSON array describing what you changed in the CV, and why>

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
  "passionate", "synergy", "dynamic", and similar filler.

The EDITS array documents every substantive change you made to the CV, so the
candidate can check your work rather than trust it. Each entry:

  {"section": "<where, e.g. Summary or Experience - Acme>",
   "before":  "<the original wording, verbatim from the master CV>",
   "after":   "<your replacement, verbatim from the CV above>",
   "reason":  "<under 40 characters, e.g. matches \"payments\">"}

Rules for it:
- Only real changes. If you reordered a skills list, the entry is that list
  before and after. If you cut a section entirely, "after" is an empty string.
- "before" must be text that genuinely appears in the master CV, and "after"
  text that genuinely appears in the CV you just wrote. An entry that quotes
  something neither document contains is worse than no entry, because the
  candidate will check.
- Return [] if you changed nothing of substance. Do not invent edits to look
  thorough.`

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

	cv, letter, edits, err := Split(reply)
	if err != nil {
		return Documents{}, err
	}
	// Verified against the documents actually produced, so the app never
	// renders a change that cannot be found in the CV it is annotating.
	return Documents{
		CV:          cv,
		CoverLetter: letter,
		Edits:       ValidateEdits(edits, p.MasterCV, cv),
		Model:       model,
	}, nil
}

// Split pulls the two documents out of a reply.
//
// It is lenient about a missing CV marker (some models start straight into the
// document) but strict about there being real content in both — an empty
// cover letter advancing to the approval screen is worse than a retry.
func Split(reply string) (cv, letter string, edits []Edit, err error) {
	text := strings.TrimSpace(reply)
	if text == "" {
		return "", "", nil, fmt.Errorf("%w: reply was empty", ErrEmptyOutput)
	}

	li := strings.Index(text, letterMarker)
	if li < 0 {
		return "", "", nil, fmt.Errorf("%w: no %s marker in the reply", ErrEmptyOutput, letterMarker)
	}

	cvPart := text[:li]
	rest := text[li+len(letterMarker):]

	// The edits section is optional on purpose. It is the least important of
	// the three outputs, and losing both documents because a model omitted a
	// trailing JSON array would be a bad trade.
	letter = rest
	if ei := strings.Index(rest, editsMarker); ei >= 0 {
		letter = rest[:ei]
		edits = parseEdits(rest[ei+len(editsMarker):])
	}
	letter = strings.TrimSpace(letter)

	// The CV marker is optional: if it is there, drop everything before it,
	// which also strips any preamble the model added.
	if ci := strings.Index(cvPart, cvMarker); ci >= 0 {
		cvPart = cvPart[ci+len(cvMarker):]
	}
	cv = strings.TrimSpace(stripOuterFence(cvPart))
	letter = strings.TrimSpace(stripOuterFence(letter))

	if cv == "" {
		return "", "", nil, fmt.Errorf("%w: the CV section was empty", ErrEmptyOutput)
	}
	if letter == "" {
		return "", "", nil, fmt.Errorf("%w: the cover letter section was empty", ErrEmptyOutput)
	}
	return cv, letter, edits, nil
}

// parseEdits reads the trailing JSON array, tolerating a code fence and prose.
//
// Never returns an error: a malformed edit list costs the diff view, not the
// documents. The job still advances to approval — the candidate simply reviews
// the CV without the annotations.
func parseEdits(raw string) []Edit {
	body := stripOuterFence(strings.TrimSpace(raw))

	start := strings.IndexByte(body, '[')
	end := strings.LastIndexByte(body, ']')
	if start < 0 || end <= start {
		return nil
	}

	var out []Edit
	if err := json.Unmarshal([]byte(body[start:end+1]), &out); err != nil {
		return nil
	}

	cleaned := make([]Edit, 0, len(out))
	for _, e := range out {
		e.Section = strings.TrimSpace(e.Section)
		e.Before = strings.TrimSpace(e.Before)
		e.After = strings.TrimSpace(e.After)
		e.Reason = trimTo(strings.TrimSpace(e.Reason), 60)
		// An entry with neither side is not an edit. One side empty is
		// legitimate: it means text was cut, or added.
		if e.Before == "" && e.After == "" {
			continue
		}
		cleaned = append(cleaned, e)
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func trimTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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

// ValidateEdits drops entries that quote text neither document contains.
//
// Live models get this wrong. On the first real run one edit claimed an "after"
// of "Databases: MongoDb, Firestore" that appears nowhere in the CV it just
// wrote — a plausible-sounding change it did not actually make. Rendering that
// would produce a diff the candidate cannot reconcile with the document in
// front of them, which is worse than showing no diff at all: it teaches them
// the annotations are decorative.
//
// Comparison is normalised, not literal. Markdown emphasis and whitespace
// differ constantly between what a model reports and what it wrote
// ("**Databases:** MongoDb" vs "Databases: MongoDb"), and rejecting on that
// would throw away honest edits.
func ValidateEdits(edits []Edit, masterCV, tailoredCV string) []Edit {
	if len(edits) == 0 {
		return nil
	}

	master := normalizeForMatch(masterCV)
	tailored := normalizeForMatch(tailoredCV)

	kept := make([]Edit, 0, len(edits))
	for _, e := range edits {
		before := normalizeForMatch(e.Before)
		after := normalizeForMatch(e.After)

		// An empty side is meaningful: no "before" is an addition, no "after"
		// is a cut. Only the non-empty side has to be verifiable.
		if before != "" && !containsFragment(master, before) {
			continue
		}
		if after != "" && !containsFragment(tailored, after) {
			continue
		}
		kept = append(kept, e)
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

var markdownNoise = strings.NewReplacer(
	"**", "", "__", "", "*", "", "_", "", "`", "", "#", "", ">", "", "-", " ",
)

// normalizeForMatch lowercases, strips markdown decoration, and collapses
// whitespace, so a comparison is about words rather than formatting.
func normalizeForMatch(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(markdownNoise.Replace(s))), " ")
}

// containsFragment checks the opening of the quoted text rather than all of it.
//
// A model routinely paraphrases the tail of a long quotation while getting the
// start right; requiring an exact full match would reject edits that are
// substantially honest. Short quotes must match outright, since there is no
// prefix to be confident about.
func containsFragment(doc, fragment string) bool {
	const prefix = 45
	if len(fragment) <= prefix {
		return strings.Contains(doc, fragment)
	}
	return strings.Contains(doc, fragment[:prefix])
}
