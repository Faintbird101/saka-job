-- ============================================================
--  0003 — WF-D as an apply-pack digest, not an email sender
-- ============================================================
--
-- The original design had WF-D email the application to the employer. The live
-- data does not support that: across every job ingested so far, zero carried a
-- hiring-manager email address. Every posting is an apply URL.
--
-- So WF-D moves Approved jobs to ManualApply and sends YOU a digest with the
-- apply link and the generated documents. SMTP is used to notify the candidate,
-- never to contact an employer — a much smaller blast radius, and it works with
-- the data that actually exists.

-- How long a job may sit in ManualApply before it is closed automatically.
--
-- Without this the list grows forever: jobs you decided against but never
-- explicitly rejected stay "pending" indefinitely and the dashboard stops
-- meaning anything. 0 disables expiry.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS manual_apply_grace_days INT NOT NULL DEFAULT 7;

-- Where the digest goes. Kept in the profile rather than the environment so it
-- is editable from the app, and so it is clearly OUR address rather than an
-- employer's.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS notify_email TEXT;

-- When the job entered ManualApply — the clock the grace period runs against.
--
-- updated_at cannot serve here: any later edit (a note, a re-read) would touch
-- it and silently restart the countdown.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS manual_apply_at TIMESTAMPTZ;

-- The expiry sweep queries exactly this.
CREATE INDEX IF NOT EXISTS idx_jobs_manual_apply_at ON jobs(manual_apply_at)
    WHERE manual_apply_at IS NOT NULL;
