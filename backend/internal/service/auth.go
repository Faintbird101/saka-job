package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/yourname/jobhunter/backend/internal/auth"
	"github.com/yourname/jobhunter/backend/internal/db/queries"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// SessionTTL is how long a login lasts. Long, because this is a personal tool
// on your own phone and re-authenticating daily would just train you to pick a
// shorter password. Revocation is the real control, not expiry.
const SessionTTL = 30 * 24 * time.Hour

// ErrUnauthorized → 401.
var ErrUnauthorized = errors.New("unauthorized")

// Session is what an authenticated request resolves to.
type Session struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"display_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginResult is returned once, and carries the only copy of the token.
type LoginResult struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      models.User `json:"user"`
}

// NeedsBootstrap reports whether no account exists yet.
//
// Registration is open only in that window. Once an account exists the endpoint
// closes permanently, so a reachable API cannot be used by anyone else to
// create themselves an account.
func (s *Service) NeedsBootstrap(ctx context.Context) (bool, error) {
	var n int
	if err := s.db.Pool.QueryRow(ctx, queries.CountUsers).Scan(&n); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return n == 0, nil
}

// Bootstrap creates the first and only account.
func (s *Service) Bootstrap(ctx context.Context, email, password, name string) (models.User, error) {
	open, err := s.NeedsBootstrap(ctx)
	if err != nil {
		return models.User{}, err
	}
	if !open {
		// Not 404: pretending the endpoint does not exist would be confusing
		// when you are trying to work out why your own signup failed.
		return models.User{}, fmt.Errorf("%w: an account already exists; registration is closed", ErrConflict)
	}

	email = auth.NormalizeEmail(email)
	if !auth.ValidEmail(email) {
		return models.User{}, fmt.Errorf("%w: that does not look like an email address", ErrInvalidInput)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		if errors.Is(err, auth.ErrWeakPassword) {
			return models.User{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
		}
		return models.User{}, err
	}

	var u models.User
	if err := s.db.Pool.QueryRow(ctx, queries.InsertUser, email, hash, nullIfEmpty(name)).
		Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt); err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}

	s.log.Info("account created", "email", u.Email)
	return u, nil
}

// Login verifies credentials and mints a session.
func (s *Service) Login(ctx context.Context, email, password, userAgent string) (LoginResult, error) {
	email = auth.NormalizeEmail(email)

	var (
		u    models.User
		hash string
	)
	err := s.db.Pool.QueryRow(ctx, queries.GetUserByEmail, email).
		Scan(&u.ID, &u.Email, &hash, &u.DisplayName, &u.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		// Deliberately the same error, and deliberately still doing work: an
		// immediate return here would make "no such account" measurably faster
		// than "wrong password", which is a timing oracle for which addresses
		// are registered.
		_ = auth.VerifyPassword("$2a$12$........................................................", password)
		return LoginResult{}, fmt.Errorf("%w: %s", ErrUnauthorized, auth.ErrBadCredentials)
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("look up user: %w", err)
	}

	if err := auth.VerifyPassword(hash, password); err != nil {
		s.log.Warn("failed login attempt", "email", email)
		return LoginResult{}, fmt.Errorf("%w: %s", ErrUnauthorized, auth.ErrBadCredentials)
	}

	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return LoginResult{}, err
	}
	expires := time.Now().Add(SessionTTL)

	if _, err := s.db.Pool.Exec(ctx, queries.InsertSession,
		tokenHash, u.ID, nullIfEmpty(trimTo(userAgent, 300)), expires); err != nil {
		return LoginResult{}, fmt.Errorf("create session: %w", err)
	}

	// Opportunistic housekeeping; a failure here must not fail the login.
	if _, err := s.db.Pool.Exec(ctx, queries.PurgeExpiredSessions); err != nil {
		s.log.Warn("could not purge expired sessions", "error", err)
	}
	if _, err := s.db.Pool.Exec(ctx, queries.TouchLastLogin, u.ID); err != nil {
		s.log.Warn("could not record last login", "error", err)
	}

	s.log.Info("login", "email", u.Email)
	return LoginResult{Token: token, ExpiresAt: expires, User: u}, nil
}

// ResolveSession validates a bearer token.
//
// Called on every authenticated request, so it is one indexed primary-key
// lookup on a SHA-256 hash — which is exactly why session tokens are hashed
// with SHA-256 and not bcrypt.
func (s *Service) ResolveSession(ctx context.Context, token string) (Session, error) {
	if token == "" {
		return Session{}, ErrUnauthorized
	}

	var sess Session
	err := s.db.Pool.QueryRow(ctx, queries.LookupSession, auth.HashToken(token)).
		Scan(&sess.UserID, &sess.Email, &sess.Name, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Covers unknown, revoked, and expired alike — the query filters on
		// expiry so an expired session is never a race away from working.
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, fmt.Errorf("resolve session: %w", err)
	}
	return sess, nil
}

// TouchSession updates last-seen, best effort.
func (s *Service) TouchSession(ctx context.Context, token string) {
	if _, err := s.db.Pool.Exec(ctx, queries.TouchSession, auth.HashToken(token)); err != nil {
		s.log.Debug("could not touch session", "error", err)
	}
}

// Logout revokes one session.
func (s *Service) Logout(ctx context.Context, token string) error {
	if _, err := s.db.Pool.Exec(ctx, queries.DeleteSession, auth.HashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// LogoutEverywhere revokes every session for a user — the thing to do when a
// phone goes missing, and the reason these are server-side sessions.
func (s *Service) LogoutEverywhere(ctx context.Context, userID string) error {
	if _, err := s.db.Pool.Exec(ctx, queries.DeleteUserSessions, userID); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	s.log.Info("all sessions revoked", "user_id", userID)
	return nil
}

// ---------- push devices ----------

// RegisterDevice records an FCM token for this account.
func (s *Service) RegisterDevice(ctx context.Context, userID, token, platform string) (models.PushDevice, error) {
	if token == "" {
		return models.PushDevice{}, fmt.Errorf("%w: device token is required", ErrInvalidInput)
	}

	var d models.PushDevice
	if err := s.db.Pool.QueryRow(ctx, queries.UpsertDevice, token, userID, nullIfEmpty(platform)).
		Scan(&d.Token, &d.Platform, &d.CreatedAt, &d.LastSeenAt); err != nil {
		return models.PushDevice{}, fmt.Errorf("register device: %w", err)
	}
	s.log.Info("push device registered", "platform", d.Platform)
	return d, nil
}

// UnregisterDevice drops a device token, for sign-out.
func (s *Service) UnregisterDevice(ctx context.Context, token string) error {
	if _, err := s.db.Pool.Exec(ctx, queries.DeleteDevice, token); err != nil {
		return fmt.Errorf("unregister device: %w", err)
	}
	return nil
}
