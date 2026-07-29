package scoring

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// KeywordScorer scores by overlap between what a job asks for and what the
// profile evidences. No network, no API key, no cost, and the same output
// shape as the LLM scorer.
//
// This works better here than it would in most places because the source API
// has already done the hard extraction: ai_key_skills and ai_keywords arrive
// as clean, discrete skill names rather than prose to be mined. The scorer's
// job is therefore matching, not comprehension.
//
// What it cannot do: read the requirements summary as prose, weigh "3 years of
// Flutter" against "2+ years required", or notice that a "Mobile Engineer"
// posting is really a Flutter role. That is the trade for costing nothing.
type KeywordScorer struct{}

// NewKeywordScorer builds the deterministic scorer.
func NewKeywordScorer() *KeywordScorer { return &KeywordScorer{} }

// Name identifies this scorer in logs and run summaries.
func (k *KeywordScorer) Name() string { return "keyword" }

// Weights for the three signals. They sum to 100.
//
// Skills dominate because they are the most reliable signal in the payload:
// the API extracts them explicitly, whereas title and seniority are free text
// that varies wildly between postings.
const (
	weightSkills    = 70
	weightTitle     = 20
	weightSeniority = 10
)

// depthTarget is how many matched skills counts as "deep" coverage. Four is
// roughly the point where a match stops being coincidental — one shared skill
// is noise, four means the role genuinely overlaps the CV.
const depthTarget = 4

// Score computes the match.
func (k *KeywordScorer) Score(_ context.Context, p models.Profile, j models.Job) (Result, error) {
	// The job's requirements are its key skills; keywords are supporting
	// signal, so they inform the match but are not held against the candidate
	// as "missing" — a keyword like "mobile" is a category, not a requirement.
	required := normalizeAll(jsonList(j.AIKeySkills))
	keywords := normalizeAll(jsonList(j.AIKeywords))

	if len(required) == 0 && len(keywords) == 0 {
		// Nothing to match on. Refusing beats inventing a score: with no
		// signal, any number would be fiction, and ScoreFailed correctly
		// flags the row for attention.
		return Result{}, fmt.Errorf("%w: job lists no skills or keywords to match against", ErrUnparseable)
	}

	// What the candidate evidences: explicit preferred skills, plus anything
	// their CV text mentions. The CV is searched rather than parsed, which is
	// crude but robust — it needs no CV format and no section headings.
	evidence := normalizeAll(jsonList(p.PreferredSkills))
	cvText := " " + normalizeText(p.MasterCV) + " "

	var matched, missing []string
	for _, req := range required {
		if evidences(evidence, cvText, req) {
			matched = append(matched, req.original)
		} else {
			missing = append(missing, req.original)
		}
	}

	// Keywords contribute to the score but never to missing_skills.
	keywordHits := 0
	for _, kw := range keywords {
		if evidences(evidence, cvText, kw) {
			keywordHits++
		}
	}

	score := k.combine(len(matched), len(required), keywordHits, len(keywords), p, j)

	sort.Strings(matched)
	sort.Strings(missing)

	return Result{
		Score:         score,
		MatchedSkills: nonNil(matched),
		MissingSkills: nonNil(missing),
		Summary:       buildSummary(score, matched, missing, j),
	}, nil
}

// combine turns the raw counts into a 0-100 score.
func (k *KeywordScorer) combine(matched, required, kwHits, kwTotal int, p models.Profile, j models.Job) int {
	// --- skills (70) ---
	//
	// Two signals, blended, because neither works alone:
	//
	//   coverage = matched/required — what proportion of the ask is met. Alone
	//     it punishes thorough postings: a job listing 16 skills where you have
	//     the 3 that matter scores 0.19, while a lazy posting listing 2 skills
	//     you both have scores 1.0. That ranked a Flutter internship above a
	//     real Flutter engineering role on live data.
	//
	//   depth = how many skills matched outright. Alone it ignores the ask
	//     entirely, so a job wanting 3 things you have looks identical to one
	//     wanting 30 of which you have 3.
	//
	// Together they behave: strong coverage still wins, but matching several
	// core skills carries weight even when the posting lists a long tail.
	var skillFraction float64
	switch {
	case required > 0:
		coverage := float64(matched) / float64(required)
		depth := min(1.0, float64(matched)/float64(depthTarget))
		skillFraction = 0.6*coverage + 0.4*depth
	case kwTotal > 0:
		// No explicit skills listed; fall back to keywords so the job is still
		// scored on something rather than dropping to zero.
		skillFraction = float64(kwHits) / float64(kwTotal)
	}
	// Keywords nudge the skill component up, capped, so a job whose skills all
	// match cannot exceed the component's weight.
	if required > 0 && kwTotal > 0 {
		skillFraction = min(1.0, skillFraction+0.15*float64(kwHits)/float64(kwTotal))
	}
	total := skillFraction * weightSkills

	// --- title (20) ---
	// Does the posting's title look like something the candidate is searching
	// for? This is what separates a Flutter role from a React role that merely
	// lists Dart somewhere.
	total += titleScore(jsonList(p.SearchTitles), j.Title) * weightTitle

	// --- seniority (10) ---
	total += seniorityScore(p.MasterCV, j) * weightSeniority

	return clamp(int(total + 0.5))
}

// titleScore returns 1.0 for a title containing one of the searched-for terms,
// 0.5 for a partial word overlap, 0 otherwise.
func titleScore(searchTitles []string, jobTitle string) float64 {
	title := normalizeText(jobTitle)
	if title == "" || len(searchTitles) == 0 {
		// Without search titles configured there is nothing to compare, so this
		// component is neutral rather than punitive.
		return 0.5
	}

	best := 0.0
	for _, want := range searchTitles {
		w := normalizeText(want)
		if w == "" {
			continue
		}
		if strings.Contains(title, w) {
			return 1.0
		}
		// Partial: any significant word in common ("Dart Developer" vs
		// "Senior Developer, Mobile").
		for _, word := range strings.Fields(w) {
			if len(word) > 3 && strings.Contains(title, word) {
				best = 0.5
			}
		}
	}
	return best
}

// seniorityScore is a coarse check that the role's level is not wildly beyond
// what the CV suggests. It is weighted lowest because posting seniority text
// is inconsistent — "Premier emploi", "Mid-Senior level", "2-5" all appear.
func seniorityScore(cv string, j models.Job) float64 {
	level := normalizeText(j.Seniority + " " + j.ExperienceLevel)
	if level == "" {
		return 0.5 // unknown: neutral
	}

	senior := isExperienced(cv)

	switch {
	case strings.Contains(level, "intern") || strings.Contains(level, "entry") ||
		strings.Contains(level, "graduate") || strings.Contains(level, "premier emploi"):
		// An internship is a poor fit for an experienced candidate, and this is
		// the one case worth actively penalising — it is common in the feed.
		if senior {
			return 0.0
		}
		return 0.7
	case strings.Contains(level, "director") || strings.Contains(level, "vp") ||
		strings.Contains(level, "head of") || strings.Contains(level, "executive"):
		if senior {
			return 0.7
		}
		return 0.2
	default:
		return 1.0
	}
}

// isExperienced decides whether the CV describes someone past entry level.
//
// It reads a stated number of years as well as job-title words, because plenty
// of good CVs say "four years building Flutter apps" and never use the word
// "senior" — and without this an internship outranks a real role, which is
// exactly what happened on the first live run.
func isExperienced(cv string) bool {
	lower := normalizeText(cv)

	for _, w := range []string{"senior", "lead", "principal", "architect", "staff engineer"} {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return yearsOfExperience(lower) >= 3
}

// yearsOfExperience finds the largest "N years" figure in the text, in digits
// or spelled out. Returns 0 when nothing is stated.
func yearsOfExperience(normalized string) int {
	words := map[string]int{
		"one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
		"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
	}

	fields := strings.Fields(normalized)
	best := 0
	for i, f := range fields {
		// Only count a number that is actually followed by "year"/"years",
		// so a postcode or a version number is not read as experience.
		if i+1 >= len(fields) || !strings.HasPrefix(fields[i+1], "year") {
			continue
		}
		n := 0
		if v, ok := words[f]; ok {
			n = v
		} else if v, err := strconv.Atoi(strings.TrimSuffix(f, "+")); err == nil {
			n = v
		}
		// Ignore implausible figures (a year like "2026 years").
		if n > best && n <= 50 {
			best = n
		}
	}
	return best
}

// buildSummary writes the same kind of one-liner the LLM produces, so the app
// renders both identically.
func buildSummary(score int, matched, missing []string, j models.Job) string {
	var b strings.Builder

	switch {
	case score >= 85:
		b.WriteString("Strong match. ")
	case score >= 70:
		b.WriteString("Good match. ")
	case score >= 50:
		b.WriteString("Partial match. ")
	default:
		b.WriteString("Weak match. ")
	}

	if len(matched) > 0 {
		fmt.Fprintf(&b, "Evidences %d of %d listed skills (%s). ",
			len(matched), len(matched)+len(missing), joinCapped(matched, 4))
	} else {
		b.WriteString("None of the listed skills are evidenced in the profile. ")
	}
	if len(missing) > 0 {
		fmt.Fprintf(&b, "Not evidenced: %s.", joinCapped(missing, 4))
	}

	return truncate(strings.TrimSpace(b.String()), maxSummary)
}

// ---------- normalisation ----------

// term is a skill with both its display form and its comparison form.
type term struct {
	original   string
	normalized string
}

// normalizeAll converts a skill list to comparable terms, dropping blanks.
func normalizeAll(in []string) []term {
	out := make([]term, 0, len(in))
	for _, s := range in {
		n := normalizeText(s)
		if n == "" {
			continue
		}
		out = append(out, term{original: strings.TrimSpace(s), normalized: n})
	}
	return out
}

// normalizeText lowercases, strips punctuation, and collapses whitespace, so
// "React Native", "react-native" and "React  native." all compare equal.
func normalizeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastSpace := true

	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '#':
			// + and # survive because they carry meaning: c++, c#, notepad++.
			b.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// evidences reports whether the candidate demonstrates a required term, either
// by listing it explicitly or by mentioning it in the CV.
func evidences(preferred []term, cvText string, required term) bool {
	for _, p := range preferred {
		if p.normalized == required.normalized {
			return true
		}
		// Containment both ways: "flutter" matches "flutter development", and
		// a preferred "google cloud platform" matches a required "google cloud".
		if len(required.normalized) >= 3 &&
			(strings.Contains(p.normalized, required.normalized) ||
				strings.Contains(required.normalized, p.normalized)) {
			return true
		}
	}

	// Word-boundary search in the CV. The padding spaces are what stop "go"
	// matching inside "django" or "algorithm" — a false positive that would
	// otherwise inflate scores badly for short skill names.
	return strings.Contains(cvText, " "+required.normalized+" ")
}

// ---------- small helpers ----------

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func joinCapped(items []string, cap int) string {
	if len(items) <= cap {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(items[:cap], ", "), len(items)-cap)
}
