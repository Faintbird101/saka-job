package queries

// ---------- profile (singleton, id = 1) ----------

// GetProfile reads the one settings row. The row is seeded by 0001_init.sql,
// so this never returns zero rows on a correctly migrated database.
const GetProfile = `
SELECT
    COALESCE(master_cv, ''),
    COALESCE(search_titles, '[]'::jsonb),
    COALESCE(preferred_skills, '[]'::jsonb),
    min_score_threshold,
    max_jobs_per_run,
    COALESCE(scoring_model, ''),
    COALESCE(generation_model, ''),
    COALESCE(cover_letter_notes, ''),
    manual_apply_grace_days,
    COALESCE(notify_email, ''),
    updated_at
FROM profile
WHERE id = 1`

// UpdateProfile patches the singleton. Same COALESCE-means-leave-alone rule as
// UpdateJob.
const UpdateProfile = `
UPDATE profile SET
    master_cv           = COALESCE($1, master_cv),
    search_titles       = COALESCE($2, search_titles),
    preferred_skills    = COALESCE($3, preferred_skills),
    min_score_threshold = COALESCE($4, min_score_threshold),
    max_jobs_per_run    = COALESCE($5, max_jobs_per_run),
    scoring_model       = COALESCE($6, scoring_model),
    generation_model    = COALESCE($7, generation_model),
    cover_letter_notes  = COALESCE($8, cover_letter_notes),
    manual_apply_grace_days = COALESCE($9, manual_apply_grace_days),
    notify_email        = COALESCE($10, notify_email)
WHERE id = 1
RETURNING
    COALESCE(master_cv, ''),
    COALESCE(search_titles, '[]'::jsonb),
    COALESCE(preferred_skills, '[]'::jsonb),
    min_score_threshold,
    max_jobs_per_run,
    COALESCE(scoring_model, ''),
    COALESCE(generation_model, ''),
    COALESCE(cover_letter_notes, ''),
    manual_apply_grace_days,
    COALESCE(notify_email, ''),
    updated_at`

// ---------- fetch_log (API quota visibility) ----------

// InsertFetchLog records one ingest run.
const InsertFetchLog = `
INSERT INTO fetch_log (query_title, returned_count, inserted_count, skipped_count, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, fetched_at`

// LatestFetchLog is the "when did we last hit the API, and what did we get"
// line on the dashboard.
const LatestFetchLog = `
SELECT
    id, fetched_at,
    COALESCE(query_title, ''),
    COALESCE(returned_count, 0),
    COALESCE(inserted_count, 0),
    COALESCE(skipped_count, 0),
    COALESCE(notes, '')
FROM fetch_log
ORDER BY fetched_at DESC
LIMIT 1`

// ListFetchLogs is the consumption history.
const ListFetchLogs = `
SELECT
    id, fetched_at,
    COALESCE(query_title, ''),
    COALESCE(returned_count, 0),
    COALESCE(inserted_count, 0),
    COALESCE(skipped_count, 0),
    COALESCE(notes, '')
FROM fetch_log
ORDER BY fetched_at DESC
LIMIT $1`

// ---------- errors (application error log) ----------

// InsertError writes one failure. job_id is a nullable FK, so a workflow-level
// error that isn't tied to a specific job still gets recorded.
const InsertError = `
INSERT INTO errors (workflow, job_id, message, context)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at`

// ListErrors is the newest-first error feed.
const ListErrors = `
SELECT
    id,
    COALESCE(workflow, ''),
    job_id,
    COALESCE(message, ''),
    COALESCE(context, '{}'::jsonb),
    created_at
FROM errors
ORDER BY created_at DESC
LIMIT $1`

// CountRecentErrors is the 24-hour error badge on the dashboard.
const CountRecentErrors = `SELECT count(*) FROM errors WHERE created_at > now() - interval '24 hours'`

// ClearErrors empties the application error log. The dashboard's 24-hour count
// is only useful if stale noise can be cleared out after it has been dealt with.
const ClearErrors = `DELETE FROM errors`

// ClearErrorsBefore removes entries older than a cutoff, for routine pruning.
const ClearErrorsBefore = `DELETE FROM errors WHERE created_at < $1`

// MarkManualApply moves an approved job to ManualApply and stamps the clock the
// grace period runs against. A separate column from updated_at, which any later
// edit would touch and silently restart the countdown.
const MarkManualApply = `UPDATE jobs SET manual_apply_at = now() WHERE id = $1`

// ExpireManualApply closes jobs that have sat in ManualApply past the grace
// period. Returns what it closed so the run can report it rather than silently
// tidying up behind you.
const ExpireManualApply = `
UPDATE jobs
SET status = 'Closed'
WHERE status = 'ManualApply'
  AND manual_apply_at IS NOT NULL
  -- make_interval takes an integer directly. Building the interval by string
  -- concatenation ("$1 || ' days'") makes Postgres infer $1 as TEXT, and the
  -- driver then cannot encode an int into it: "unable to encode 7 into text
  -- format for text (OID 25)".
  AND manual_apply_at < now() - make_interval(days => $1)
RETURNING id, title, organization`
