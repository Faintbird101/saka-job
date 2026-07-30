package queries

// CountUsers gates the one-time bootstrap: registration is only open while
// there are zero accounts, so the endpoint cannot be used to add a second one.
const CountUsers = `SELECT count(*) FROM users`

// InsertUser creates an account.
const InsertUser = `
INSERT INTO users (email, password_hash, display_name)
VALUES ($1, $2, $3)
RETURNING id, email, COALESCE(display_name, ''), created_at`

// GetUserByEmail is the login lookup.
const GetUserByEmail = `
SELECT id, email, password_hash, COALESCE(display_name, ''), created_at
FROM users WHERE email = $1`

// TouchLastLogin records a successful login.
const TouchLastLogin = `UPDATE users SET last_login_at = now() WHERE id = $1`

// InsertSession stores only the token HASH; the plaintext is shown once and
// never persisted.
const InsertSession = `
INSERT INTO sessions (token_hash, user_id, user_agent, expires_at)
VALUES ($1, $2, $3, $4)`

// LookupSession resolves a token hash to its user, rejecting expired rows in
// the query rather than in Go — an expired session must never be a race away
// from being accepted.
const LookupSession = `
SELECT s.user_id, u.email, COALESCE(u.display_name, ''), s.expires_at
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now()`

// TouchSession keeps last_seen_at current, for showing active devices.
const TouchSession = `UPDATE sessions SET last_seen_at = now() WHERE token_hash = $1`

// DeleteSession is logout.
const DeleteSession = `DELETE FROM sessions WHERE token_hash = $1`

// DeleteUserSessions is "sign out everywhere" — what you want after losing a
// phone, and the reason sessions are server-side rather than JWTs.
const DeleteUserSessions = `DELETE FROM sessions WHERE user_id = $1`

// PurgeExpiredSessions is housekeeping, run on login.
const PurgeExpiredSessions = `DELETE FROM sessions WHERE expires_at < now()`

// ---------- push devices ----------

// UpsertDevice registers or refreshes a device token. FCM rotates tokens, so a
// re-register of an existing token must refresh it rather than fail.
const UpsertDevice = `
INSERT INTO push_devices (token, user_id, platform)
VALUES ($1, $2, $3)
ON CONFLICT (token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    last_seen_at = now(),
    failed_at = NULL,
    fail_reason = NULL
RETURNING token, COALESCE(platform, ''), created_at, last_seen_at`

// DeleteDevice removes a device on sign-out.
const DeleteDevice = `DELETE FROM push_devices WHERE token = $1`

// LiveDeviceTokens is the send list — everything not known to be dead.
const LiveDeviceTokens = `SELECT token FROM push_devices WHERE failed_at IS NULL`

// MarkDeviceFailed records an FCM rejection. Kept rather than deleted so a
// delivery problem stays diagnosable.
const MarkDeviceFailed = `
UPDATE push_devices SET failed_at = now(), fail_reason = $2 WHERE token = $1`
