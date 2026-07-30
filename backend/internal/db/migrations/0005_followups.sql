-- ============================================================
--  0005 — WF-E follow-ups
-- ============================================================
--
-- Follow-ups only became worth building once the inbox scanner existed. Before
-- it, "no reply" could not be known, so a follow-up was a nudge on a timer that
-- would happily chase an employer who had already answered. WF-E now checks
-- job_events and skips anything that got a real reply.

-- When the follow-up nudge went out. A separate clock from date_applied so the
-- close-out sweep measures the right interval.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS followup_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_jobs_followup_at ON jobs(followup_at)
    WHERE followup_at IS NOT NULL;

-- How long after applying to chase. The README's original range was 7-14 days.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS followup_after_days INT NOT NULL DEFAULT 7;

-- How long after chasing to give up and close. 0 disables the sweep, for anyone
-- who would rather tidy up by hand.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS followup_close_days INT NOT NULL DEFAULT 14;
