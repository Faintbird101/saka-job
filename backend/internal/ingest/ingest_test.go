package ingest

import (
	"encoding/json"
	"strings"
	"testing"
)

// item builds a minimal but realistic API payload.
func item(id int64, linkedinID int64, title, url string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"id":                id,
		"linkedin_id":       linkedinID,
		"title":             title,
		"organization":      "Acme",
		"url":               url,
		"date_posted":       "2026-07-26T16:26:11.63",
		"employment_type":   []string{"FULL_TIME"},
		"ai_key_skills":     []string{"Flutter", "Dart"},
		"countries_derived": []string{"Kenya"},
	})
	return b
}

func TestPrepareKeepsDistinctJobs(t *testing.T) {
	batch := Batch{
		QueryTitle: "Flutter",
		Jobs: []json.RawMessage{
			item(1, 111111111, "Flutter Engineer", "https://www.linkedin.com/jobs/view/a-111111111"),
			item(2, 222222222, "Dart Developer", "https://www.linkedin.com/jobs/view/b-222222222"),
		},
	}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 2 {
		t.Fatalf("kept %d jobs, want 2", len(res.Keep))
	}
	if res.SkippedInBatch != 0 || res.Invalid != 0 {
		t.Errorf("unexpected skips: skipped=%d invalid=%d", res.SkippedInBatch, res.Invalid)
	}
	if res.Keep[0].Status != "New" {
		t.Errorf("status = %q, want New — ingested jobs must enter the state machine at New", res.Keep[0].Status)
	}
	if res.Keep[0].NormalizedURL == "" {
		t.Error("normalized_url was not computed; the third dedup guard would be dead")
	}
}

func TestPrepareDropsDuplicateSourceID(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		item(1, 111111111, "Flutter Engineer", "https://www.linkedin.com/jobs/view/a-111111111"),
		item(1, 999999999, "Flutter Engineer (repost)", "https://www.linkedin.com/jobs/view/z-999999999"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 1 {
		t.Fatalf("kept %d, want 1 — source_job_id is the primary dedup guard", len(res.Keep))
	}
	if res.SkippedInBatch != 1 {
		t.Errorf("SkippedInBatch = %d, want 1", res.SkippedInBatch)
	}
}

// This is the case the README calls out: the same posting cross-listed under
// different country subdomains, with different source ids but an identical
// LinkedIn job id.
func TestPrepareDropsCrossListedSubdomainDuplicate(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		item(1, 4444856922, "Flutter Engineer", "https://vn.linkedin.com/jobs/view/x-4444856922"),
		item(2, 4444856922, "Flutter Engineer", "https://mx.linkedin.com/jobs/view/x-4444856922"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 1 {
		t.Fatalf("kept %d, want 1 — same linkedin_id under two subdomains is one job", len(res.Keep))
	}
	if res.SkippedInBatch != 1 {
		t.Errorf("SkippedInBatch = %d, want 1", res.SkippedInBatch)
	}
}

// Non-LinkedIn sources have no numeric id, so normalized_url is the only guard
// left. It has to work on its own.
func TestPrepareDropsURLDuplicateWithoutLinkedInID(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		item(1, 0, "Backend Engineer", "https://www.greenhouse.io/acme/jobs/998877"),
		item(2, 0, "Backend Engineer", "https://greenhouse.io/acme/jobs/998877/"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 1 {
		t.Fatalf("kept %d, want 1 — normalized_url must catch non-LinkedIn duplicates", len(res.Keep))
	}
}

// Two different postings that both lack a LinkedIn id must NOT collide. This
// is why linkedin_id is stored as NULL rather than 0 when absent.
func TestPrepareKeepsDistinctJobsWithoutLinkedInID(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		item(1, 0, "Backend Engineer", "https://greenhouse.io/acme/jobs/1"),
		item(2, 0, "Frontend Engineer", "https://greenhouse.io/acme/jobs/2"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 2 {
		t.Fatalf("kept %d, want 2 — a missing linkedin_id must not make jobs collide", len(res.Keep))
	}
	for _, j := range res.Keep {
		if j.LinkedInID != 0 {
			t.Errorf("expected LinkedInID 0 for a greenhouse URL with a short numeric tail, got %d", j.LinkedInID)
		}
	}
}

// When the API omits linkedin_id but the URL carries it, we recover it —
// otherwise a subsequent run that *does* include the field would insert a
// second copy of the same posting.
func TestPrepareRecoversLinkedInIDFromURL(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		item(1, 0, "Flutter Engineer", "https://vn.linkedin.com/jobs/view/x-4444856922"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 1 {
		t.Fatalf("kept %d, want 1", len(res.Keep))
	}
	if res.Keep[0].LinkedInID != 4444856922 {
		t.Errorf("LinkedInID = %d, want 4444856922 recovered from the URL", res.Keep[0].LinkedInID)
	}
}

func TestPrepareRejectsUnusableItems(t *testing.T) {
	batch := Batch{Jobs: []json.RawMessage{
		json.RawMessage(`{"id": 0, "title": "No source id"}`),
		json.RawMessage(`{"id": 5, "title": "   "}`),
		json.RawMessage(`{not json`),
		item(9, 999999999, "Good One", "https://linkedin.com/jobs/view/g-999999999"),
	}}

	res, err := Prepare(batch)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(res.Keep) != 1 {
		t.Fatalf("kept %d, want 1", len(res.Keep))
	}
	if res.Invalid != 3 {
		t.Errorf("Invalid = %d, want 3", res.Invalid)
	}
	if len(res.Problems) != 3 {
		t.Errorf("got %d problem descriptions, want 3 — skipped items must be explainable in fetch_log", len(res.Problems))
	}
}

func TestPrepareRejectsOversizedBatch(t *testing.T) {
	jobs := make([]json.RawMessage, MaxBatchSize+1)
	for i := range jobs {
		jobs[i] = item(int64(i+1), 0, "Job", "https://example.com/j")
	}

	if _, err := Prepare(Batch{Jobs: jobs}); err == nil {
		t.Fatal("expected an error for a batch over MaxBatchSize")
	}
}

// n8n's HTTP Request node forwards the API response verbatim, which is a bare
// array. Accepting it saves a Function node in the workflow.
func TestBatchUnmarshalAcceptsBareArray(t *testing.T) {
	body := `[` + string(item(1, 111111111, "Flutter Engineer", "https://linkedin.com/jobs/view/a-111111111")) + `]`

	var b Batch
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatalf("unmarshal bare array: %v", err)
	}
	if len(b.Jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(b.Jobs))
	}
	if b.QueryTitle != "" {
		t.Errorf("QueryTitle = %q, want empty for the bare-array form", b.QueryTitle)
	}
}

func TestBatchUnmarshalAcceptsEnvelope(t *testing.T) {
	body := `{"query_title":"Flutter","notes":"morning run","jobs":[` +
		string(item(1, 111111111, "Flutter Engineer", "https://linkedin.com/jobs/view/a-111111111")) + `]}`

	var b Batch
	if err := json.Unmarshal([]byte(body), &b); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if b.QueryTitle != "Flutter" || b.Notes != "morning run" || len(b.Jobs) != 1 {
		t.Errorf("decoded = %+v, want the envelope fields populated", b)
	}
}

func TestBatchUnmarshalRejectsGarbage(t *testing.T) {
	var b Batch
	err := json.Unmarshal([]byte(`"just a string"`), &b)
	if err == nil {
		t.Fatal("expected an error decoding a non-object, non-array body")
	}
	if !strings.Contains(err.Error(), "decode ingest body") {
		t.Errorf("error = %v, want it to mention the ingest body", err)
	}
}
