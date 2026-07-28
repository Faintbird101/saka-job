package models

import (
	"encoding/json"
	"testing"
)

// A trimmed but faithful copy of a real API item, including the quirks the
// normaliser exists to absorb: fractional-second timestamps with no zone, and
// derived geo fields that disagree with the raw address.
const sampleItem = `{
  "id": 4444856922,
  "linkedin_id": 4444856922,
  "title": "  Flutter Product Engineer  ",
  "organization": " WorkBuddy ",
  "organization_url": "https://linkedin.com/company/workbuddy",
  "url": "https://mx.linkedin.com/jobs/view/flutter-product-engineer-at-workbuddy-4444856922",
  "source": "linkedin",
  "source_domain": "mx.linkedin.com",
  "description_text": "Build things.",
  "date_posted": "2026-07-26T16:26:11.63",
  "date_valid_through": "2026-08-26",
  "employment_type": ["FULL_TIME", "CONTRACTOR"],
  "seniority": "Mid-Senior level",
  "direct_apply": true,
  "countries_derived": ["Mexico"],
  "locations_derived": ["Chiapas, Mexico"],
  "ai_key_skills": ["Flutter", "Dart"],
  "ai_keywords": ["mobile", "cross-platform"],
  "ai_experience_level": "2-5",
  "ai_work_arrangement": "Remote OK",
  "ai_requirements_summary": "2+ years of Flutter.",
  "ai_salary_currency": "MXN",
  "ai_salary_min_value": 40000,
  "ai_salary_unit_text": "MONTH",
  "locations": [{"address": {"addressCountry": "Mexico", "addressLocality": "Monterrey"}}]
}`

func TestNormalizeMapsRealPayload(t *testing.T) {
	var api APIJob
	if err := json.Unmarshal([]byte(sampleItem), &api); err != nil {
		t.Fatalf("decode sample: %v", err)
	}

	job := api.Normalize(json.RawMessage(sampleItem))

	if job.SourceJobID != 4444856922 {
		t.Errorf("SourceJobID = %d, want 4444856922", job.SourceJobID)
	}
	if job.Title != "Flutter Product Engineer" {
		t.Errorf("Title = %q, want the trimmed value", job.Title)
	}
	if job.Organization != "WorkBuddy" {
		t.Errorf("Organization = %q, want the trimmed value", job.Organization)
	}
	// employment_type is an array in the API and a single column here.
	if job.EmploymentType != "FULL_TIME" {
		t.Errorf("EmploymentType = %q, want the first element", job.EmploymentType)
	}
	if job.Status != StatusNew {
		t.Errorf("Status = %q, want New", job.Status)
	}
	if job.RawPayload == nil {
		t.Error("RawPayload is nil; reprocessing without re-fetching would be impossible")
	}
	if job.SalaryMin == nil || *job.SalaryMin != 40000 {
		t.Errorf("SalaryMin = %v, want 40000", job.SalaryMin)
	}
	if job.SalaryMax != nil {
		t.Errorf("SalaryMax = %v, want nil when the API omits it", *job.SalaryMax)
	}
}

// The README's data-quality note: locations_derived put a Monterrey job in
// Chiapas, so the raw address wins.
func TestNormalizePrefersRawAddressOverDerivedGeo(t *testing.T) {
	var api APIJob
	if err := json.Unmarshal([]byte(sampleItem), &api); err != nil {
		t.Fatalf("decode sample: %v", err)
	}

	job := api.Normalize(json.RawMessage(sampleItem))

	if job.Country != "Mexico" {
		t.Errorf("Country = %q, want Mexico", job.Country)
	}
	if job.LocationRaw != "Monterrey" {
		t.Errorf("LocationRaw = %q, want Monterrey from the raw address, not the derived %q",
			job.LocationRaw, "Chiapas, Mexico")
	}
}

func TestNormalizeFallsBackToDerivedGeoWhenAddressIsAbsent(t *testing.T) {
	body := `{"id":1,"title":"X","countries_derived":["Kenya"],"locations_derived":["Nairobi, Kenya"]}`
	var api APIJob
	if err := json.Unmarshal([]byte(body), &api); err != nil {
		t.Fatalf("decode: %v", err)
	}

	job := api.Normalize(json.RawMessage(body))
	if job.Country != "Kenya" {
		t.Errorf("Country = %q, want the derived fallback Kenya", job.Country)
	}
	if job.LocationRaw != "Nairobi, Kenya" {
		t.Errorf("LocationRaw = %q, want the derived fallback", job.LocationRaw)
	}
}

// The API mixes timestamp formats within a single response.
func TestParseTimeHandlesEveryObservedFormat(t *testing.T) {
	cases := []struct {
		in      string
		wantNil bool
	}{
		{"2026-07-26T16:26:11.63", false}, // fractional seconds, no zone
		{"2026-07-27T04:19:19", false},    // no fraction, no zone
		{"2026-07-27T04:19:19Z", false},   // RFC3339
		{"2026-08-26", false},             // date only
		{"", true},
		{"   ", true},
		{"not a date", true},
	}

	for _, tc := range cases {
		got := parseTime(tc.in)
		if tc.wantNil && got != nil {
			t.Errorf("parseTime(%q) = %v, want nil", tc.in, got)
		}
		if !tc.wantNil && got == nil {
			t.Errorf("parseTime(%q) = nil, want a parsed time", tc.in)
		}
	}
}

// A NULL jsonb would force every client to handle both null and []. Normalise
// at the boundary instead.
func TestNormalizeEmitsEmptyArraysNotNull(t *testing.T) {
	body := `{"id":1,"title":"X"}`
	var api APIJob
	if err := json.Unmarshal([]byte(body), &api); err != nil {
		t.Fatalf("decode: %v", err)
	}

	job := api.Normalize(json.RawMessage(body))
	if string(job.AIKeySkills) != "[]" {
		t.Errorf("AIKeySkills = %s, want []", job.AIKeySkills)
	}
	if string(job.AIKeywords) != "[]" {
		t.Errorf("AIKeywords = %s, want []", job.AIKeywords)
	}
}
