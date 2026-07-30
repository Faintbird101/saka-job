-- ============================================================
--  0004 — inbox scanning: match employer replies to jobs
-- ============================================================
--
-- Once we can read the inbox, two things become possible that were not before:
-- knowing an application was acknowledged, and knowing there was NO reply —
-- which is what makes a follow-up (WF-E) meaningful rather than a blind nag.

-- ---------- New statuses ----------
--
-- Note the naming care here. The existing `Rejected` means "YOU declined this
-- job" (set by you in the app). An employer turning you down is a different
-- event with a different meaning, so it gets its own value rather than
-- overloading one that already means something else. Conflating them would
-- make the dashboard unreadable — you could not tell what you passed on from
-- what passed on you.
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check CHECK (status = ANY (ARRAY[
    'New', 'Scored', 'CVGenerated', 'AwaitingApproval', 'Approved',
    'Applied', 'FollowUpSent', 'Closed',
    'LowMatch', 'Rejected', 'ScoreFailed', 'ManualApply',
    -- added in 0004, all driven by inbound email:
    'Acknowledged',      -- employer confirmed receipt
    'Interviewing',      -- invited to interview / next stage
    'OfferReceived',     -- an offer arrived
    'EmployerRejected'   -- they turned you down (NOT the same as 'Rejected')
]));

-- ---------- Event log ----------
--
-- One row per matched email. Separate from the jobs table because a job can
-- accumulate several replies (acknowledgement, then interview, then offer) and
-- flattening them onto the row would lose all but the last.
--
-- Deliberately stores an EXCERPT, not the full body. This is someone else's
-- correspondence in a database that exists to track job applications; keeping
-- entire messages is more personal data than the feature needs. The excerpt is
-- enough to see why something was classified the way it was.
CREATE TABLE IF NOT EXISTS job_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id       UUID REFERENCES jobs(id) ON DELETE CASCADE,

    source       TEXT NOT NULL DEFAULT 'email',
    kind         TEXT NOT NULL,        -- acknowledgement|rejection|interview|offer|other
    -- How sure the classifier was, 0-100. Low confidence is not a failure: it
    -- is the signal that a human should look.
    confidence   INT  NOT NULL DEFAULT 0,
    classifier   TEXT,                 -- "keyword" or "llm:<model>", for audit

    sender       TEXT,
    sender_domain TEXT,
    subject      TEXT,
    excerpt      TEXT,
    received_at  TIMESTAMPTZ,

    -- Match quality, so an attribution can be second-guessed later.
    match_score  INT NOT NULL DEFAULT 0,
    match_reason TEXT,

    -- The status this event suggests. NULL when it implies no change.
    suggested_status TEXT,
    -- NULL = awaiting your decision, true = you confirmed, false = you rejected
    -- the suggestion. Consequential moves are never applied automatically.
    confirmed    BOOLEAN,
    applied_at   TIMESTAMPTZ,          -- when the status change actually happened

    -- IMAP message id, so a rescan cannot record the same email twice.
    message_id   TEXT UNIQUE,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_job_events_job     ON job_events(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_events_pending ON job_events(created_at DESC)
    WHERE confirmed IS NULL AND suggested_status IS NOT NULL;

-- Unmatched mail is kept with job_id NULL: it is the record of what the matcher
-- could not attribute, which is how you find out the matching rules need work.
CREATE INDEX IF NOT EXISTS idx_job_events_unmatched ON job_events(created_at DESC)
    WHERE job_id IS NULL;

-- ---------- Inbox settings ----------

-- The real company domain, lifted out of raw_payload at ingest time so the
-- matcher can index it instead of re-parsing JSON on every email.
--
-- organization_url is NOT usable for this: it is a linkedin.com/company/...
-- page, so every job would appear to share one domain.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS org_domain TEXT;
CREATE INDEX IF NOT EXISTS idx_jobs_org_domain ON jobs(org_domain)
    WHERE org_domain IS NOT NULL;

-- Backfill from what we already ingested.
UPDATE jobs
SET org_domain = lower(regexp_replace(
        regexp_replace(raw_payload->>'org_linkedin_website', '^https?://', ''),
        '^www\.|/.*$', '', 'g'))
WHERE org_domain IS NULL
  AND coalesce(raw_payload->>'org_linkedin_website', '') <> '';

-- Marks the last message we processed, so a scan is incremental.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS inbox_last_scan_at TIMESTAMPTZ;
-- Below this the classification is never auto-applied, only suggested.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS inbox_auto_confidence INT NOT NULL DEFAULT 85;
