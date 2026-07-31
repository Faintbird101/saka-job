-- ============================================================
--  0007 — five scoring axes, and a per-edit CV diff
-- ============================================================
--
-- Both exist for the same reason: the app has to show its work. A single
-- number and an opaque rewritten CV are exactly the black box you would not
-- hand your job search to.

-- ---------- Score axes ----------
--
-- {"skills":96,"seniority":88,"domain":94,"location":90,"pay":null}
--
-- A NULL axis means "the posting did not say", NOT zero. That distinction
-- matters more than it looks: salary is absent from most postings in the live
-- feed, and scoring silence as zero would drag every score down and make the
-- pay bar permanently red for reasons that have nothing to do with the job.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS score_axes JSONB;

-- ---------- CV edit trail ----------
--
-- [{"section":"Summary","before":"...","after":"...","reason":"matches
--   'payments'"}]
--
-- Captured at generation time rather than diffed later, because the reason is
-- the part that builds trust and only the model that made the edit knows it.
-- A client-side diff could show what changed but never why.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cv_edits JSONB;

-- ---------- Profile inputs the new axes need ----------
--
-- Location and pay cannot be scored against nothing. These are what the
-- location and pay axes compare a posting to.

-- Somewhere the candidate would actually work: ["Nairobi","Kenya","Remote"].
ALTER TABLE profile ADD COLUMN IF NOT EXISTS preferred_locations JSONB DEFAULT '[]'::jsonb;

-- any | remote_only | hybrid_ok | onsite_ok
ALTER TABLE profile ADD COLUMN IF NOT EXISTS remote_preference TEXT NOT NULL DEFAULT 'any';

-- The floor, in the currency below. 0 means "not stated", which makes the pay
-- axis unknown rather than failing every job.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS salary_floor INT NOT NULL DEFAULT 0;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS salary_currency TEXT NOT NULL DEFAULT 'KES';

-- Weights, so the dial in the app can retune scoring without a deploy. They do
-- not have to sum to 100 — the scorer normalises whatever is here, which means
-- setting one to 0 cleanly removes that axis from consideration.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS weight_skills    INT NOT NULL DEFAULT 45;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS weight_seniority INT NOT NULL DEFAULT 15;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS weight_domain    INT NOT NULL DEFAULT 20;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS weight_location  INT NOT NULL DEFAULT 12;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS weight_pay       INT NOT NULL DEFAULT 8;

-- Finding the weakest axis across everything scored is how the app answers
-- "what is holding my matches back".
CREATE INDEX IF NOT EXISTS idx_jobs_score_axes ON jobs USING gin (score_axes)
    WHERE score_axes IS NOT NULL;
