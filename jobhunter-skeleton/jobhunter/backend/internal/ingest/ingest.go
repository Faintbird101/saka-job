// Package ingest turns a raw RapidAPI response into rows we're willing to
// insert: it normalises each item, computes the three dedup keys, and drops
// duplicates that appear within the same batch.
//
// It does no database work — that's service.Ingest. Keeping the transform
// pure means the dedup logic is testable without a container.
package ingest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/models"
)

// MaxBatchSize caps one ingest call. The profile's max_jobs_per_run is the
// real rate-limit guard; this is a blunt backstop against a misconfigured
// workflow POSTing a 10k-item page and holding a transaction open for minutes.
const MaxBatchSize = 200

// Batch is the ingest request body.
//
// QueryTitle and Notes are metadata for the fetch_log row: knowing that the
// run which returned 40 duplicates was the "Flutter" search and not the "Dart"
// one is the difference between a useful quota log and a pile of numbers.
type Batch struct {
	QueryTitle string            `json:"query_title"`
	Notes      string            `json:"notes"`
	Jobs       []json.RawMessage `json:"jobs"`
}

// UnmarshalJSON accepts either shape:
//
//	{"query_title": "Flutter", "jobs": [ ... ]}
//	[ ... ]
//
// The bare array exists because it's what you get by wiring the RapidAPI node
// straight into the HTTP Request node in n8n. Accepting it removes a Function
// node from the workflow, at the cost of a nameless fetch_log row.
func (b *Batch) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimLeftFunc(string(data), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})

	if strings.HasPrefix(trimmed, "[") {
		var jobs []json.RawMessage
		if err := json.Unmarshal(data, &jobs); err != nil {
			return fmt.Errorf("decode job array: %w", err)
		}
		b.Jobs = jobs
		return nil
	}

	// Alias to avoid recursing into this method.
	type batchAlias Batch
	var a batchAlias
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("decode ingest body: %w", err)
	}
	*b = Batch(a)
	return nil
}

// Result is what Prepare produced.
type Result struct {
	Keep []models.Job
	// SkippedInBatch counts items dropped because an earlier item in the SAME
	// batch had the same key. These never reach Postgres, so they'd otherwise
	// be invisible in the fetch_log numbers.
	SkippedInBatch int
	// Invalid counts items we could not use at all (unparseable JSON, or no
	// source id to dedup on).
	Invalid int
	// Problems describes the invalid items, for the fetch_log notes field.
	Problems []string
}

// Prepare normalises a batch and removes intra-batch duplicates.
//
// The API genuinely returns the same posting twice within one response when a
// job is cross-listed across country subdomains. Postgres would catch that via
// ON CONFLICT, but only after we'd already spent a round trip per item — and
// the "inserted vs skipped" counts would then hide the fact that the source
// data itself was duplicated.
func Prepare(batch Batch) (Result, error) {
	if len(batch.Jobs) > MaxBatchSize {
		return Result{}, fmt.Errorf("batch of %d exceeds maximum of %d jobs", len(batch.Jobs), MaxBatchSize)
	}

	var res Result
	seenSource := make(map[int64]bool, len(batch.Jobs))
	seenLinkedIn := make(map[int64]bool, len(batch.Jobs))
	seenURL := make(map[string]bool, len(batch.Jobs))

	for i, raw := range batch.Jobs {
		var api models.APIJob
		if err := json.Unmarshal(raw, &api); err != nil {
			res.Invalid++
			res.Problems = append(res.Problems, fmt.Sprintf("item %d: unparseable (%v)", i, err))
			continue
		}

		// source_job_id is NOT NULL and the primary dedup guard. Without it we
		// have no stable identity for the posting, so we refuse rather than
		// insert a row that will duplicate on the next run.
		if api.ID == 0 {
			res.Invalid++
			res.Problems = append(res.Problems, fmt.Sprintf("item %d: missing source id", i))
			continue
		}
		if strings.TrimSpace(api.Title) == "" {
			res.Invalid++
			res.Problems = append(res.Problems, fmt.Sprintf("item %d (source %d): missing title", i, api.ID))
			continue
		}

		job := api.Normalize(raw)
		job.NormalizedURL = NormalizeURL(job.URL)

		// Cross-check: if the API omitted linkedin_id but the URL carries the
		// numeric job id, recover it. This is the guard that catches the same
		// posting arriving as vn.linkedin.com on one run and www.linkedin.com
		// on the next.
		if job.LinkedInID == 0 {
			if id := ExtractLinkedInID(job.URL); id != "" {
				if n, err := strconv.ParseInt(id, 10, 64); err == nil {
					job.LinkedInID = n
				}
			}
		}

		switch {
		case seenSource[job.SourceJobID]:
			res.SkippedInBatch++
			continue
		case job.LinkedInID != 0 && seenLinkedIn[job.LinkedInID]:
			res.SkippedInBatch++
			continue
		case job.NormalizedURL != "" && seenURL[job.NormalizedURL]:
			res.SkippedInBatch++
			continue
		}

		seenSource[job.SourceJobID] = true
		if job.LinkedInID != 0 {
			seenLinkedIn[job.LinkedInID] = true
		}
		if job.NormalizedURL != "" {
			seenURL[job.NormalizedURL] = true
		}

		res.Keep = append(res.Keep, job)
	}

	return res, nil
}
