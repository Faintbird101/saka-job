// Package queries holds every SQL string the backend runs.
//
// Keeping the SQL out of the service layer means you can read the whole
// database contract in one place, and a schema change has one obvious set of
// files to grep.
package queries

// JobColumns is the canonical projection for a jobs row, and it defines the
// scan order used by service.scanJob. Anything selecting a job MUST use this
// list (or ListJobColumns, which substitutes the same slots) so a single
// scanner works for every query.
//
// COALESCE everywhere: the schema allows NULL for most text columns, but the
// Go structs use plain strings. Normalising in SQL beats sprinkling
// sql.NullString through the model.
const JobColumns = `
    id,
    source_job_id,
    COALESCE(linkedin_id, 0),
    COALESCE(normalized_url, ''),
    title,
    COALESCE(organization, ''),
    COALESCE(organization_url, ''),
    COALESCE(url, ''),
    COALESCE(source, ''),
    COALESCE(source_domain, ''),
    COALESCE(description_text, ''),
    date_posted,
    date_valid_through,
    COALESCE(country, ''),
    COALESCE(location_raw, ''),
    COALESCE(work_arrangement, ''),
    COALESCE(employment_type, ''),
    COALESCE(seniority, ''),
    COALESCE(experience_level, ''),
    COALESCE(direct_apply, false),
    COALESCE(ai_key_skills, '[]'::jsonb),
    COALESCE(ai_keywords, '[]'::jsonb),
    COALESCE(ai_requirements_summary, ''),
    COALESCE(ai_core_responsibilities, ''),
    COALESCE(salary_currency, ''),
    salary_min,
    salary_max,
    COALESCE(salary_unit, ''),
    COALESCE(org_domain, ''),
    status,
    score,
    COALESCE(matched_skills, '[]'::jsonb),
    COALESCE(missing_skills, '[]'::jsonb),
    COALESCE(ai_summary, ''),
    COALESCE(cv_url, ''),
    COALESCE(cover_letter_url, ''),
    COALESCE(prompt_used, ''),
    date_applied,
    COALESCE(email_used, ''),
    score_axes,
    cv_edits,
    generated_at,
    COALESCE(generated_by, ''),
    created_at,
    updated_at`

// ListJobColumns is JobColumns with the two heavyweight text fields blanked.
// A list of 50 jobs would otherwise ship ~50 full job descriptions to a phone
// on mobile data; the detail screen fetches the real thing by id.
const ListJobColumns = `
    id,
    source_job_id,
    COALESCE(linkedin_id, 0),
    COALESCE(normalized_url, ''),
    title,
    COALESCE(organization, ''),
    COALESCE(organization_url, ''),
    COALESCE(url, ''),
    COALESCE(source, ''),
    COALESCE(source_domain, ''),
    '' AS description_text,
    date_posted,
    date_valid_through,
    COALESCE(country, ''),
    COALESCE(location_raw, ''),
    COALESCE(work_arrangement, ''),
    COALESCE(employment_type, ''),
    COALESCE(seniority, ''),
    COALESCE(experience_level, ''),
    COALESCE(direct_apply, false),
    COALESCE(ai_key_skills, '[]'::jsonb),
    COALESCE(ai_keywords, '[]'::jsonb),
    COALESCE(ai_requirements_summary, ''),
    COALESCE(ai_core_responsibilities, ''),
    COALESCE(salary_currency, ''),
    salary_min,
    salary_max,
    COALESCE(salary_unit, ''),
    COALESCE(org_domain, ''),
    status,
    score,
    COALESCE(matched_skills, '[]'::jsonb),
    COALESCE(missing_skills, '[]'::jsonb),
    COALESCE(ai_summary, ''),
    '' AS cv_url,
    COALESCE(cover_letter_url, ''),
    '' AS prompt_used,
    date_applied,
    COALESCE(email_used, ''),
    score_axes,
    cv_edits,
    generated_at,
    COALESCE(generated_by, ''),
    created_at,
    updated_at`

// GetJob fetches one job by primary key.
const GetJob = `SELECT ` + JobColumns + ` FROM jobs WHERE id = $1`

// ListJobs is the app's list screen and n8n's work queue in one query.
//
// Every filter is optional and expressed as "$n IS NULL OR ...", so one
// prepared statement covers all combinations rather than building SQL by
// string concatenation (which is where injection bugs come from).
//
// $1 status, $2 min score, $3 country, $4 search term, $5 limit, $6 offset.
const ListJobs = `
SELECT ` + ListJobColumns + `
FROM jobs
WHERE ($1::text IS NULL OR status = $1)
  AND ($2::int  IS NULL OR score >= $2)
  AND ($3::text IS NULL OR country ILIKE $3)
  AND ($4::text IS NULL OR title ILIKE '%' || $4 || '%' OR organization ILIKE '%' || $4 || '%')
ORDER BY
    CASE WHEN score IS NULL THEN 1 ELSE 0 END,
    score DESC,
    COALESCE(date_posted, created_at) DESC
LIMIT $5 OFFSET $6`

// CountJobs mirrors ListJobs' WHERE clause so the app can paginate.
const CountJobs = `
SELECT count(*)
FROM jobs
WHERE ($1::text IS NULL OR status = $1)
  AND ($2::int  IS NULL OR score >= $2)
  AND ($3::text IS NULL OR country ILIKE $3)
  AND ($4::text IS NULL OR title ILIKE '%' || $4 || '%' OR organization ILIKE '%' || $4 || '%')`

// InsertJob is the ingestion write.
//
// ON CONFLICT DO NOTHING with no conflict target covers ALL THREE unique
// constraints at once (source_job_id, linkedin_id, normalized_url) — naming a
// single target would only guard one of the three dedup nets.
//
// It returns the new id, so "no rows" is precisely how the caller learns the
// row was a duplicate. That is the entire dedup mechanism: one atomic
// statement, no read-then-write race between concurrent ingest runs.
const InsertJob = `
INSERT INTO jobs (
    source_job_id, linkedin_id, normalized_url,
    title, organization, organization_url, url, source, source_domain,
    description_text, date_posted, date_valid_through,
    org_domain,
    country, location_raw, work_arrangement,
    employment_type, seniority, experience_level, direct_apply,
    ai_key_skills, ai_keywords, ai_requirements_summary, ai_core_responsibilities,
    salary_currency, salary_min, salary_max, salary_unit,
    status, raw_payload
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7, $8, $9,
    $10, $11, $12,
    $30,
    $13, $14, $15,
    $16, $17, $18, $19,
    $20, $21, $22, $23,
    $24, $25, $26, $27,
    $28, $29
)
ON CONFLICT DO NOTHING
RETURNING id`

// GetJobStatus reads just the current status, for the transition check before
// an update.
const GetJobStatus = `SELECT status FROM jobs WHERE id = $1`

// UpdateJob applies a partial patch. Each parameter is "NULL means leave
// alone", which is why the handler layer decodes into pointer fields.
//
// $1 id, $2 status, $3 score, $4 matched_skills, $5 missing_skills,
// $6 ai_summary, $7 cv_url, $8 cover_letter_url, $9 prompt_used,
// $10 email_used, $11 date_applied.
const UpdateJob = `
UPDATE jobs SET
    status           = COALESCE($2,  status),
    score            = COALESCE($3,  score),
    matched_skills   = COALESCE($4,  matched_skills),
    missing_skills   = COALESCE($5,  missing_skills),
    ai_summary       = COALESCE($6,  ai_summary),
    cv_url           = COALESCE($7,  cv_url),
    cover_letter_url = COALESCE($8,  cover_letter_url),
    prompt_used      = COALESCE($9,  prompt_used),
    email_used       = COALESCE($10, email_used),
    date_applied     = COALESCE($11, date_applied),
    score_axes       = COALESCE($12, score_axes),
    cv_edits         = COALESCE($13, cv_edits)
WHERE id = $1
RETURNING ` + JobColumns

// StatusCounts powers the dashboard. It counts every status present, and the
// service layer fills in zeros for the ones that aren't.
const StatusCounts = `SELECT status, count(*) FROM jobs GROUP BY status`

// CountAllJobs is the dashboard total.
const CountAllJobs = `SELECT count(*) FROM jobs`

// ListJobsByStatus pulls a batch of work for a pipeline stage — WF-B asks for
// `New`, WF-C for `Scored`, and so on.
//
// Oldest first: a job that has been sitting in the queue since yesterday
// morning should be scored before one that arrived ten minutes ago, or a busy
// search term could starve the backlog indefinitely.
const ListJobsByStatus = `
SELECT ` + JobColumns + `
FROM jobs
WHERE status = $1
ORDER BY created_at ASC
LIMIT $2`

// StoreDocuments writes the WF-C output.
//
// The *_url columns become paths into our own API rather than external links,
// which keeps them meaningful now that the documents live in the row.
const StoreDocuments = `
UPDATE jobs SET
    cv_text           = $2,
    cover_letter_text = $3,
    generated_by      = $4,
    generated_at      = now(),
    cv_url            = $5,
    cover_letter_url  = $6,
    cv_edits          = $7
WHERE id = $1`

// GetDocuments reads the generated documents back.
const GetDocuments = `
SELECT COALESCE(cv_text, ''), COALESCE(cover_letter_text, ''), generated_at
FROM jobs WHERE id = $1`

// ForceStatus sets a status bypassing the transition graph. Only the rescore
// path uses it; see service.forceStatus for why that exception exists.
const ForceStatus = `
UPDATE jobs SET status = $2, score = NULL, matched_skills = NULL,
    missing_skills = NULL, ai_summary = NULL, score_axes = NULL
WHERE id = $1
RETURNING ` + JobColumns
