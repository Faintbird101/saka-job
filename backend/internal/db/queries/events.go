package queries

// EventColumns is the canonical projection for a job_events row, and defines
// the scan order used by service.scanEvent.
const EventColumns = `
    id,
    job_id,
    COALESCE(source, 'email'),
    kind,
    confidence,
    COALESCE(classifier, ''),
    COALESCE(sender, ''),
    COALESCE(sender_domain, ''),
    COALESCE(subject, ''),
    COALESCE(excerpt, ''),
    received_at,
    match_score,
    COALESCE(match_reason, ''),
    COALESCE(suggested_status, ''),
    confirmed,
    applied_at,
    COALESCE(message_id, ''),
    created_at`

// InsertEvent records one matched email.
//
// ON CONFLICT (message_id) DO NOTHING makes a rescan idempotent: IMAP will hand
// us the same message again after a restart or an overlapping window, and
// without this the same rejection would be recorded — and suggested — twice.
const InsertEvent = `
INSERT INTO job_events (
    job_id, source, kind, confidence, classifier,
    sender, sender_domain, subject, excerpt, received_at,
    match_score, match_reason, suggested_status, confirmed, applied_at, message_id
) VALUES (
    $1, 'email', $2, $3, $4,
    $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15
)
ON CONFLICT (message_id) DO NOTHING
RETURNING ` + EventColumns

// ListEventsForJob is the timeline on the job detail screen.
const ListEventsForJob = `
SELECT ` + EventColumns + `
FROM job_events WHERE job_id = $1 ORDER BY created_at DESC LIMIT $2`

// ListPendingEvents is every suggestion still awaiting your decision — the
// queue the app surfaces so a consequential classification is never applied
// without a human seeing it.
const ListPendingEvents = `
SELECT ` + EventColumns + `
FROM job_events
WHERE confirmed IS NULL AND suggested_status <> '' AND suggested_status IS NOT NULL
ORDER BY created_at DESC LIMIT $1`

// ListUnmatchedEvents is mail the matcher could not attribute. Reviewing this
// is how you find out the matching rules need work.
const ListUnmatchedEvents = `
SELECT ` + EventColumns + `
FROM job_events WHERE job_id IS NULL ORDER BY created_at DESC LIMIT $1`

// GetEvent reads one event by id.
const GetEvent = `SELECT ` + EventColumns + ` FROM job_events WHERE id = $1`

// SetEventConfirmed records your decision on a suggestion.
const SetEventConfirmed = `
UPDATE job_events
SET confirmed = $2, applied_at = CASE WHEN $2 THEN now() ELSE applied_at END
WHERE id = $1
RETURNING ` + EventColumns

// CountPendingEvents powers the badge on the dashboard.
const CountPendingEvents = `
SELECT count(*) FROM job_events
WHERE confirmed IS NULL AND suggested_status <> '' AND suggested_status IS NOT NULL`

// JobsAwaitingReply is the candidate set for inbox matching: everything that
// has actually been sent and could therefore receive a reply.
//
// Narrowed by status rather than scanning the whole table, so an email cannot
// be attributed to a job that was never applied for.
const JobsAwaitingReply = `
SELECT ` + JobColumns + `
FROM jobs
WHERE status IN ('Applied', 'FollowUpSent', 'Acknowledged', 'Interviewing', 'ManualApply')
ORDER BY coalesce(date_applied, updated_at) DESC
LIMIT 500`

// EarliestApplication is the lower bound for how far back a scan should read.
// Nothing older than your first application can be a reply to one.
const EarliestApplication = `
SELECT min(coalesce(date_applied, manual_apply_at)) FROM jobs
WHERE status IN ('Applied', 'FollowUpSent', 'Acknowledged', 'Interviewing', 'ManualApply', 'EmployerRejected', 'OfferReceived')`

// HasInboundEmail reports whether a job ever received a matched reply — the
// distinction that makes a follow-up meaningful rather than a blind nag.
const HasInboundEmail = `SELECT EXISTS(SELECT 1 FROM job_events WHERE job_id = $1)`
