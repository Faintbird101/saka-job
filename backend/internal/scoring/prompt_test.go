package scoring

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourname/jobhunter/backend/internal/models"
)

func testProfile() models.Profile {
	return models.Profile{
		MasterCV:          "Victor — 4 years building Flutter apps in Dart. Backend in Go.",
		PreferredSkills:   json.RawMessage(`["Flutter","Go"]`),
		MinScoreThreshold: 75,
	}
}

func testJob() models.Job {
	return models.Job{
		Title:                 "Flutter Product Engineer",
		Organization:          "WorkBuddy",
		Country:               "Kenya",
		LocationRaw:           "Nairobi",
		WorkArrangement:       "Remote OK",
		EmploymentType:        "FULL_TIME",
		Seniority:             "Mid-Senior level",
		ExperienceLevel:       "2-5",
		AIKeySkills:           json.RawMessage(`["Flutter","Dart","REST"]`),
		AIKeywords:            json.RawMessage(`["mobile","cross-platform"]`),
		AIRequirementsSummary: "2+ years of Flutter in production.",
	}
}

func TestBuildUserPromptIncludesBothSides(t *testing.T) {
	got := BuildUserPrompt(testProfile(), testJob())

	for _, want := range []string{
		"Victor — 4 years building Flutter", // the CV
		"Flutter Product Engineer",          // the title
		"WorkBuddy",
		"Nairobi",
		"Flutter, Dart, REST",               // key skills, flattened
		"mobile, cross-platform",            // keywords
		"2+ years of Flutter in production", // requirements summary
		"Flutter, Go",                       // preferred skills
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt is missing %q\n---\n%s", want, got)
		}
	}
}

// The source API returns no job description on this plan, so the prompt must
// stand on the extracted fields alone. If description_text ever starts
// arriving, this test is the reminder to decide deliberately whether to
// include it (and what it does to token cost).
func TestBuildUserPromptDoesNotDependOnDescriptionText(t *testing.T) {
	job := testJob()
	job.DescriptionText = "SHOULD NOT APPEAR"

	if got := BuildUserPrompt(testProfile(), job); strings.Contains(got, "SHOULD NOT APPEAR") {
		t.Error("description_text leaked into the prompt; it is null on this API plan and is not part of the contract")
	}
}

// A blank label invites the model to speculate about what belongs there.
func TestBuildUserPromptOmitsEmptyFields(t *testing.T) {
	job := testJob()
	job.Seniority = ""
	job.AICoreResponsibilities = ""
	job.WorkArrangement = "   "

	got := BuildUserPrompt(testProfile(), job)
	for _, absent := range []string{"Seniority:", "Responsibilities:", "Work arrangement:"} {
		if strings.Contains(got, absent) {
			t.Errorf("empty field %q was rendered with no value:\n%s", absent, got)
		}
	}
}

func TestBuildUserPromptSurvivesMalformedJSONB(t *testing.T) {
	job := testJob()
	job.AIKeySkills = json.RawMessage(`{"not":"an array"}`)
	job.AIKeywords = nil

	// Must not panic and must still produce a usable prompt — a bad column
	// should cost the model some context, not fail the whole scoring run.
	got := BuildUserPrompt(testProfile(), job)
	if !strings.Contains(got, "Flutter Product Engineer") {
		t.Error("prompt fell apart on malformed JSONB")
	}
	if strings.Contains(got, "Required skills:") {
		t.Error("unparseable skills should be omitted, not rendered raw")
	}
}

// The rubric is the stable prefix of every request — that is what makes it
// cacheable, and it is why scores are comparable between jobs.
func TestSystemPromptIsStableAndStatesTheContract(t *testing.T) {
	if SystemPrompt() != SystemPrompt() {
		t.Fatal("system prompt is not deterministic")
	}
	for _, want := range []string{"score", "matched_skills", "missing_skills", "summary", "JSON"} {
		if !strings.Contains(SystemPrompt(), want) {
			t.Errorf("system prompt does not mention %q", want)
		}
	}
}
