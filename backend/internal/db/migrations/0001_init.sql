-- ============================================================
--  Job Application Automation — Postgres schema (v1)
--  Built against the real linkedin-job-search-api payload.
-- ============================================================

-- Requires pgcrypto for gen_random_uuid() (built-in on PG13+ via pgcrypto,
-- or use gen_random_uuid() natively on PG13+).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------- ENUM-style status via CHECK, kept as TEXT for flexibility ----------
-- Status is the state machine that every workflow reads/writes.
--   New -> Scored -> CVGenerated -> AwaitingApproval -> Approved -> Applied -> FollowUpSent -> Closed
-- Side states: LowMatch, Rejected, ScoreFailed, ManualApply

CREATE TABLE jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- ---- Layered dedup guards (any one tripping => skip, already have it) ----
    -- 1) API's own id: strongest, stable across runs.
    source_job_id       BIGINT      UNIQUE NOT NULL,      -- API "id" field
    -- 2) LinkedIn's real job id: catches the SAME job under different country
    --    subdomains (vn./mx./eg. linkedin.com) where the raw URL string differs
    --    but the numeric id (…/view/...-4444856922) is identical.
    --    UNIQUE but NULLABLE: non-LinkedIn sources have none, and Postgres
    --    permits multiple NULLs under a UNIQUE constraint.
    linkedin_id         BIGINT      UNIQUE,               -- API "linkedin_id"
    -- 3) Normalized URL: fallback dedup for non-LinkedIn sources with no id.
    --    lowercased, query-string stripped, www./country-subdomain collapsed.
    normalized_url      TEXT        UNIQUE,

    -- Core job facts
    title               TEXT        NOT NULL,
    organization        TEXT,
    organization_url    TEXT,
    url                 TEXT,                              -- apply/view link
    source              TEXT,                              -- e.g. "linkedin"
    source_domain       TEXT,                              -- e.g. "vn.linkedin.com"
    description_text     TEXT,

    -- Dates (API sends mixed precision; TIMESTAMPTZ absorbs both)
    date_posted         TIMESTAMPTZ,
    date_valid_through  TIMESTAMPTZ,

    -- Location (trust country + raw address, NOT the derived geo fields)
    country             TEXT,                              -- from locations[].address.addressCountry
    location_raw        TEXT,                              -- human-readable locality string
    work_arrangement    TEXT,                              -- ai_work_arrangement: On-site / Remote OK / etc.

    -- Employment
    employment_type     TEXT,                              -- FULL_TIME etc. (first element)
    seniority           TEXT,
    experience_level    TEXT,                              -- ai_experience_level: "2-5" etc.
    direct_apply        BOOLEAN,

    -- AI-enriched fields the API already provides (store as JSONB where list-shaped)
    ai_key_skills       JSONB,
    ai_keywords         JSONB,
    ai_requirements_summary TEXT,
    ai_core_responsibilities TEXT,

    -- Salary (mostly null in sample, but present sometimes — Appchance had it)
    salary_currency     TEXT,
    salary_min          NUMERIC,
    salary_max          NUMERIC,
    salary_unit         TEXT,                              -- MONTH / YEAR

    -- ---- OUR pipeline fields ----
    status              TEXT        NOT NULL DEFAULT 'New'
                        CHECK (status IN (
                            'New','Scored','CVGenerated','AwaitingApproval',
                            'Approved','Applied','FollowUpSent','Closed',
                            'LowMatch','Rejected','ScoreFailed','ManualApply'
                        )),
    score               INT         CHECK (score BETWEEN 0 AND 100),
    matched_skills      JSONB,
    missing_skills      JSONB,
    ai_summary          TEXT,                              -- OUR scoring model's summary
    cv_url              TEXT,
    cover_letter_url    TEXT,
    prompt_used         TEXT,                              -- audit: exact prompt for CV gen

    date_applied        TIMESTAMPTZ,
    email_used          TEXT,

    -- Raw payload kept for reprocessing / debugging without re-fetching
    raw_payload         JSONB,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_status       ON jobs(status);
CREATE INDEX idx_jobs_score        ON jobs(score);
CREATE INDEX idx_jobs_date_posted  ON jobs(date_posted DESC);
CREATE INDEX idx_jobs_country      ON jobs(country);

-- ---------- Single-user config the app edits and n8n reads ----------
CREATE TABLE profile (
    id                  INT PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- singleton row
    master_cv           TEXT,                              -- your base CV text/markdown
    search_titles       JSONB,                             -- ["Flutter","Dart Developer"]
    preferred_skills    JSONB,
    min_score_threshold INT NOT NULL DEFAULT 70,
    max_jobs_per_run    INT NOT NULL DEFAULT 10,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO profile (id) VALUES (1) ON CONFLICT DO NOTHING;

-- ---------- Error / audit log ----------
CREATE TABLE errors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow    TEXT,               -- which n8n workflow / Go handler
    job_id      UUID REFERENCES jobs(id) ON DELETE SET NULL,
    message     TEXT,
    context     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- API consumption log (rate-limit visibility) ----------
-- One row per ingest run so you can SEE quota usage instead of guessing.
CREATE TABLE fetch_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fetched_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    query_title   TEXT,               -- the title filter used
    returned_count INT,               -- how many the API sent back
    inserted_count INT,               -- how many were actually new
    skipped_count  INT,               -- duplicates caught by the guards
    notes         TEXT
);
CREATE INDEX idx_fetch_log_time ON fetch_log(fetched_at DESC);

-- ---------- auto-update updated_at ----------
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_jobs_updated   BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_profile_updated BEFORE UPDATE ON profile
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
