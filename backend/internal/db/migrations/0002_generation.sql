-- ============================================================
--  0002 — CV/cover-letter generation (WF-C) + per-stage models
-- ============================================================

-- ---------- Generated documents ----------
-- Stored as text in the row rather than as files on a volume.
--
-- The row is the single source of truth for a job, so keeping the generated
-- documents beside the score and the status means no orphaned files when a job
-- is rejected, no volume to back up, and the text stays editable right up to
-- the moment you approve. cv_url / cover_letter_url become paths into our own
-- API (/jobs/{id}/cv), which keeps the existing columns meaningful.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cv_text            TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cover_letter_text  TEXT;

-- When the documents were produced, so a stale generation from before a CV
-- change is identifiable.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS generated_at       TIMESTAMPTZ;

-- Which model wrote them. The audit trail for "why does this read oddly" —
-- prompt_used already records the input, this records what processed it.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS generated_by       TEXT;

-- Only generated jobs are ever fetched by the approval screen.
CREATE INDEX IF NOT EXISTS idx_jobs_generated_at ON jobs(generated_at DESC)
    WHERE generated_at IS NOT NULL;

-- ---------- Per-stage model selection ----------
-- Scoring and generation want different models: scoring is a high-volume
-- classification that a small fast model handles well, while generation is
-- prose that benefits from a stronger one.
--
-- They live in the profile rather than the environment for two reasons: they
-- are editable from the app without restarting a container, and — because
-- provider free tiers meter quota PER MODEL — pointing the two stages at
-- different models doubles the daily allowance instead of sharing one.
--
-- NULL means "fall back to the LLM_MODEL environment value".
ALTER TABLE profile ADD COLUMN IF NOT EXISTS scoring_model    TEXT;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS generation_model TEXT;

-- Tone and constraints for generated documents, so the writing style is a
-- setting rather than a code change.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS cover_letter_notes TEXT;
