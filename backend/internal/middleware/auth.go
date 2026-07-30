// Package middleware holds the cross-cutting HTTP concerns: authentication,
// request IDs, structured access logging, and panic recovery.
package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/yourname/jobhunter/backend/internal/config"
)

// Role identifies which credential a request presented.
type Role string

const (
	// RoleApp is the Flutter app: it may read jobs, patch the human-decision
	// fields, and edit the profile.
	RoleApp Role = "app"
	// RoleUser is a logged-in account holding a session token. Same privileges
	// as RoleApp, but attributable to a person and revocable individually,
	// which a shared static token can never be.
	RoleUser Role = "user"
	// RoleN8N is the automation layer: it may additionally ingest jobs and
	// write pipeline fields (score, CV URLs, error log).
	RoleN8N Role = "n8n"
)

type ctxKey int

const (
	roleKey ctxKey = iota
	sessionKey
	tokenKey
)

// SessionResolver is the subset of the service layer that Auth needs. An
// interface rather than the concrete type so middleware does not depend on
// service, which depends on db.
type SessionResolver interface {
	ResolveSession(ctx context.Context, token string) (Session, error)
}

// Session is the authenticated identity attached to a request.
type Session struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"display_name,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionFrom returns the logged-in account on a request context, if any.
func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey).(Session)
	return s, ok
}

// TokenFrom returns the raw bearer token, for logout.
func TokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(tokenKey).(string)
	return t
}

// RoleFrom returns the authenticated role on a request context.
func RoleFrom(ctx context.Context) (Role, bool) {
	r, ok := ctx.Value(roleKey).(Role)
	return r, ok
}

// Auth accepts either of the two credentials and records which one was used.
//
// Two separate secrets, not one: n8n runs unattended and its key lives in a
// container's environment, while the app token lives on a phone. Sharing one
// value would mean a leaked phone token could also ingest jobs and rewrite
// scores. Which endpoints each role may reach is enforced by RequireRole.
//
// Accepted forms:
//
//	Authorization: Bearer <token>
//	X-API-Key: <token>
func Auth(cfg config.Config, sessions SessionResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := presentedToken(r)
			if token == "" {
				unauthorized(w, "missing credentials")
				return
			}

			ctx := r.Context()

			// Static service credentials first: they are a constant-time
			// compare with no database round trip, and n8n presents one on
			// every automated call.
			//
			// Constant-time on both branches — a timing difference would leak
			// the token a character at a time.
			var role Role
			switch {
			case secureEqual(token, cfg.N8NKey):
				role = RoleN8N
			case secureEqual(token, cfg.AppToken):
				role = RoleApp
			default:
				// Not a service token: try it as a session. Unknown, revoked,
				// and expired all land here and are indistinguishable to the
				// caller, which is the point.
				if sessions == nil {
					unauthorized(w, "invalid credentials")
					return
				}
				sess, err := sessions.ResolveSession(ctx, token)
				if err != nil {
					unauthorized(w, "invalid or expired credentials")
					return
				}
				role = RoleUser
				ctx = context.WithValue(ctx, sessionKey, sess)
				ctx = context.WithValue(ctx, tokenKey, token)
			}

			ctx = context.WithValue(ctx, roleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole gates a route group to a specific credential. Applied to
// /internal/*, it is what stops a phone token from reaching the ingest
// endpoint.
//
// Note RoleUser is deliberately NOT accepted anywhere RoleN8N is required: a
// logged-in phone must not be able to trigger ingestion or scoring runs.
func RequireRole(want Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := RoleFrom(r.Context())
			if !ok || got != want {
				forbidden(w, "this endpoint requires the "+string(want)+" credential")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func presentedToken(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get("Authorization")); h != "" {
		if after, found := strings.CutPrefix(h, "Bearer "); found {
			return strings.TrimSpace(after)
		}
		return h
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

func secureEqual(a, b string) bool {
	// A zero-length configured secret would otherwise match a zero-length
	// header; config.validate already rejects that, this is belt and braces.
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func unauthorized(w http.ResponseWriter, msg string) {
	writeAuthError(w, http.StatusUnauthorized, msg)
}

func forbidden(w http.ResponseWriter, msg string) {
	writeAuthError(w, http.StatusForbidden, msg)
}

// writeAuthError emits the same JSON envelope the handlers package uses.
// It's duplicated here rather than imported to keep middleware free of a
// dependency on handlers (which depends on service, which depends on db).
func writeAuthError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	// Small enough to hand-write; avoids an encoder allocation on the hot
	// path of an unauthenticated flood.
	_, _ = w.Write([]byte(`{"error":{"message":` + quote(msg) + `}}`))
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
