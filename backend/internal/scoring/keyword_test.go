package scoring

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/yourname/jobhunter/backend/internal/models"
)

func kwProfile() models.Profile {
	return models.Profile{
		MasterCV: `Victor Kinyua — Mobile & Backend Engineer.
Four years building production Flutter apps in Dart.
Backend services in Go with PostgreSQL. REST API design.
Comfortable with Docker and CI pipelines.`,
		PreferredSkills:   json.RawMessage(`["Flutter","Dart","Go"]`),
		SearchTitles:      json.RawMessage(`["Flutter","Dart Developer"]`),
		MinScoreThreshold: 75,
	}
}

func kwJob(title string, skills, keywords []string) models.Job {
	s, _ := json.Marshal(skills)
	k, _ := json.Marshal(keywords)
	return models.Job{
		Title:       title,
		AIKeySkills: s,
		AIKeywords:  k,
		Seniority:   "Mid-Senior level",
	}
}

func score(t *testing.T, p models.Profile, j models.Job) Result {
	t.Helper()
	r, err := NewKeywordScorer().Score(context.Background(), p, j)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	return r
}

func TestKeywordScorerRanksAGoodMatchHighly(t *testing.T) {
	r := score(t, kwProfile(), kwJob("Flutter Product Engineer",
		[]string{"Flutter", "Dart", "REST"}, []string{"mobile"}))

	if r.Score < 75 {
		t.Errorf("score = %d, want >= 75 for a job matching every skill and the title", r.Score)
	}
	if len(r.MissingSkills) != 0 {
		t.Errorf("MissingSkills = %v, want none", r.MissingSkills)
	}
	if len(r.MatchedSkills) != 3 {
		t.Errorf("MatchedSkills = %v, want all three", r.MatchedSkills)
	}
}

// The whole point of scoring: keep the irrelevant postings away from you. The
// README notes the API's title filter is loose enough to return React roles
// for a Flutter search.
func TestKeywordScorerRejectsAnUnrelatedRole(t *testing.T) {
	r := score(t, kwProfile(), kwJob("Senior Angular Developer",
		[]string{"Angular", "TypeScript", "RxJS"}, []string{"frontend"}))

	if r.Score >= 75 {
		t.Errorf("score = %d, want < 75 — no skill overlap and an unrelated title", r.Score)
	}
	if len(r.MissingSkills) != 3 {
		t.Errorf("MissingSkills = %v, want all three listed as missing", r.MissingSkills)
	}
}

func TestKeywordScorerSeparatesGoodFromBad(t *testing.T) {
	p := kwProfile()
	good := score(t, p, kwJob("Flutter Engineer", []string{"Flutter", "Dart"}, nil))
	bad := score(t, p, kwJob("Angular Engineer", []string{"Angular", "RxJS"}, nil))

	// The absolute numbers can shift as weights are tuned; the ordering is the
	// property that must hold.
	if good.Score <= bad.Score {
		t.Errorf("a matching job scored %d and an unrelated one %d — ordering is broken", good.Score, bad.Score)
	}
}

// "Flutter" and "flutter development" must match; casing and punctuation must
// not decide whether you see a job.
func TestKeywordScorerNormalisesSkillNames(t *testing.T) {
	p := kwProfile()
	for _, variant := range []string{"flutter", "FLUTTER", "Flutter.", "Flutter Development"} {
		r := score(t, p, kwJob("Engineer", []string{variant}, nil))
		if len(r.MatchedSkills) != 1 {
			t.Errorf("variant %q did not match: matched=%v missing=%v", variant, r.MatchedSkills, r.MissingSkills)
		}
	}
}

// The CV is searched with word boundaries. Without them "Go" would match
// inside "Django" and "algorithm", inflating scores for every job listing Go.
func TestKeywordScorerDoesNotMatchSubstringsInsideWords(t *testing.T) {
	p := models.Profile{
		MasterCV:        "Experienced with Django, algorithms and Rust.",
		PreferredSkills: json.RawMessage(`[]`),
		SearchTitles:    json.RawMessage(`[]`),
	}
	r := score(t, p, kwJob("Backend Engineer", []string{"Go"}, nil))

	if len(r.MatchedSkills) != 0 {
		t.Errorf("%q matched inside another word — Django/algorithm should not evidence Go", r.MatchedSkills)
	}
}

func TestKeywordScorerFindsSkillsInTheCVNotJustPreferredList(t *testing.T) {
	p := kwProfile() // preferred list has no PostgreSQL, but the CV mentions it
	r := score(t, p, kwJob("Backend Engineer", []string{"PostgreSQL", "Docker"}, nil))

	if len(r.MatchedSkills) != 2 {
		t.Errorf("matched=%v missing=%v — both appear in the CV text", r.MatchedSkills, r.MissingSkills)
	}
}

// An internship for someone with four years of experience is noise. This is
// common in the live feed.
func TestKeywordScorerPenalisesAnInternshipForASeniorCandidate(t *testing.T) {
	p := kwProfile()
	p.MasterCV = "Senior Engineer. " + p.MasterCV

	junior := kwJob("Flutter Intern", []string{"Flutter", "Dart"}, nil)
	junior.Seniority = "Internship"
	normal := kwJob("Flutter Engineer", []string{"Flutter", "Dart"}, nil)

	jr := score(t, p, junior)
	nr := score(t, p, normal)
	if jr.Score >= nr.Score {
		t.Errorf("internship scored %d vs %d for an equivalent normal role — seniority is not penalised", jr.Score, nr.Score)
	}
}

// Scoring on nothing would be fiction. ScoreFailed correctly flags the row.
func TestKeywordScorerRefusesAJobWithNoSkillsOrKeywords(t *testing.T) {
	_, err := NewKeywordScorer().Score(context.Background(), kwProfile(),
		kwJob("Mystery Role", nil, nil))

	if !errors.Is(err, ErrUnparseable) {
		t.Errorf("want ErrUnparseable for a job with nothing to match on, got %v", err)
	}
}

func TestKeywordScorerAlwaysProducesValidStorableOutput(t *testing.T) {
	p := kwProfile()
	jobs := []models.Job{
		kwJob("Flutter Engineer", []string{"Flutter"}, []string{"mobile"}),
		kwJob("Angular Dev", []string{"Angular"}, nil),
		kwJob("Generalist", nil, []string{"software"}),
	}
	for _, j := range jobs {
		r := score(t, p, j)

		if r.Score < 0 || r.Score > 100 {
			t.Errorf("score %d is outside the DB CHECK range", r.Score)
		}
		// Non-nil so the JSONB columns get [] rather than null.
		if r.MatchedSkills == nil || r.MissingSkills == nil {
			t.Error("nil skill slice would be stored as JSON null")
		}
		if strings.TrimSpace(r.Summary) == "" {
			t.Error("empty summary — the app has nothing to show")
		}
		if len(r.Summary) > maxSummary+4 {
			t.Errorf("summary too long: %d chars", len(r.Summary))
		}
	}
}

// Determinism is the reason this scorer needs no audit trail: the stored job
// row plus the profile fully explain the score.
func TestKeywordScorerIsDeterministic(t *testing.T) {
	p, j := kwProfile(), kwJob("Flutter Engineer", []string{"Flutter", "Kotlin"}, []string{"mobile"})
	first := score(t, p, j)
	for i := 0; i < 5; i++ {
		if got := score(t, p, j); got.Score != first.Score {
			t.Fatalf("run %d scored %d, first run scored %d", i, got.Score, first.Score)
		}
	}
}

func TestKeywordScorerName(t *testing.T) {
	if got := NewKeywordScorer().Name(); got != "keyword" {
		t.Errorf("Name() = %q", got)
	}
}

// Both scorers must satisfy the same interface — that is what makes
// SCORING_MODE a one-variable switch rather than a code change.
func TestBothScorersImplementScorer(t *testing.T) {
	var _ Scorer = NewKeywordScorer()
	var _ Scorer = NewLLMScorer(nil, "test-model")

	if got := NewLLMScorer(nil, "claude-sonnet-5").Name(); got != "llm:claude-sonnet-5" {
		t.Errorf("LLM scorer Name() = %q, want the model recorded for traceability", got)
	}
}

// Regression: on the first live run a "Flutter Intern" listing 2 skills
// outranked a real Flutter engineering role listing 16, because pure coverage
// rewards short skill lists. Matching several core skills has to count even
// when the posting has a long tail.
func TestKeywordScorerDoesNotRewardShortSkillLists(t *testing.T) {
	p := kwProfile()

	thorough := kwJob("Mobile Software Developer (Flutter)",
		[]string{"Flutter", "Dart", "PostgreSQL", "Docker", "Kotlin", "Swift",
			"GraphQL", "Redux", "Firebase", "Jenkins"}, []string{"mobile"})
	lazy := kwJob("Flutter Intern", []string{"Flutter", "Dart"}, nil)
	lazy.Seniority = "Internship"

	th := score(t, p, thorough)
	lz := score(t, p, lazy)

	if lz.Score >= th.Score {
		t.Errorf("internship listing 2 skills scored %d; thorough role matching 4 core skills scored %d — short lists are still being rewarded",
			lz.Score, th.Score)
	}
}

// A CV that says "four years building Flutter apps" describes an experienced
// engineer even though it never uses the word "senior". Without this, an
// internship outranks a real role.
func TestIsExperiencedReadsStatedYears(t *testing.T) {
	cases := map[string]bool{
		"Four years building production Flutter apps.": true,
		"4 years of Go and Postgres.":                  true,
		"5+ years in mobile development":               true,
		"Senior Engineer":                              true,
		"Lead developer on two projects":               true,
		"Recent graduate, one year of internships":     false,
		"Two years of study":                           false,
		"":                                             false,
		// A stray number that is not an experience claim must not count.
		"Nairobi 2026 office, 1 year here": false,
	}
	for cv, want := range cases {
		if got := isExperienced(cv); got != want {
			t.Errorf("isExperienced(%q) = %v, want %v", cv, got, want)
		}
	}
}

func TestYearsOfExperienceIgnoresBareNumbers(t *testing.T) {
	if n := yearsOfExperience(normalizeText("built 12 apps and 3 services")); n != 0 {
		t.Errorf("counted %d years from a sentence with no year claim", n)
	}
	if n := yearsOfExperience(normalizeText("four years of flutter")); n != 4 {
		t.Errorf("spelled-out years = %d, want 4", n)
	}
	if n := yearsOfExperience(normalizeText("in 2026, 3 years experience")); n != 3 {
		t.Errorf("got %d, want 3 — a bare year like 2026 must not be read as experience", n)
	}
}
