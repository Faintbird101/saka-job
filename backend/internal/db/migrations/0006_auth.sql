-- ============================================================
--  0006 — accounts, sessions, and push devices
-- ============================================================
--
-- Replaces the shared static API_AUTH_TOKEN for the app with a real login.
-- That token is a permanent secret that would have to live on a phone forever,
-- cannot be revoked without breaking every other client, and identifies nobody.
--
-- Scope note: this is deliberately SINGLE-user. There is no user_id on jobs,
-- profile, or job_events, because adding one that is always the same value is
-- cost without benefit — it would touch every query while isolating nothing.
-- Going multi-tenant later means adding those columns and scoping the queries;
-- the auth layer built here is the part that would stay.

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    -- bcrypt output. Never a plaintext or reversible form: a stolen database
    -- must not yield a usable password, here or on whatever else the password
    -- was reused for.
    password_hash TEXT NOT NULL,
    display_name  TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

-- Opaque server-side sessions rather than self-contained JWTs.
--
-- A JWT cannot be revoked before it expires without keeping a deny-list, which
-- is a session table wearing a disguise. A phone can be lost; being able to
-- kill its session immediately is worth more here than saving a database
-- lookup per request.
CREATE TABLE IF NOT EXISTS sessions (
    -- SHA-256 of the token, not the token. A leaked backup then yields nothing
    -- usable, exactly as with passwords.
    token_hash  TEXT PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user    ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- ---------- Push devices ----------
--
-- One row per installed app instance. FCM tokens rotate — on reinstall, on
-- restore, and occasionally on their own — so the token is the key and stale
-- ones are pruned when FCM reports them unregistered.
CREATE TABLE IF NOT EXISTS push_devices (
    token       TEXT PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    platform    TEXT,                        -- android | ios | web
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Set when FCM rejects the token; kept briefly rather than deleted so a
    -- delivery failure is diagnosable.
    failed_at   TIMESTAMPTZ,
    fail_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_push_devices_live ON push_devices(user_id)
    WHERE failed_at IS NULL;

-- Which notifications you want. Defaults match what was asked for: approvals,
-- replies, follow-ups, and failures.
ALTER TABLE profile ADD COLUMN IF NOT EXISTS push_on_approval  BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS push_on_reply     BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS push_on_followup  BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE profile ADD COLUMN IF NOT EXISTS push_on_failure   BOOLEAN NOT NULL DEFAULT true;
