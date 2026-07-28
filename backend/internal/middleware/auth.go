// Package middleware holds the cross-cutting HTTP concerns: authentication,
// request IDs, structured access logging, and panic recovery.
package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/yourname/jobhunter/backend/internal/config"
)

// Role identifies which credential a request presented.
type Role string

const (
	// RoleApp is the Flutter app: it may read jobs, patch the human-decision
	// fields, and edit the profile.
	RoleApp Role = "app"
	// RoleN8N is the automation layer: it may additionally ingest jobs and
	// write pipeline fields (score, CV URLs, error log).
	RoleN8N Role = "n8n"
)

type ctxKey int

const roleKey ctxKey = iota

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
func Auth(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := presentedToken(r)
			if token == "" {
				unauthorized(w, "missing credentials")
				return
			}

			// Constant-time compare on both branches: a timing difference here
			// would leak the token a character at a time.
			var role Role
			switch {
			case secureEqual(token, cfg.N8NKey):
				role = RoleN8N
			case secureEqual(token, cfg.AppToken):
				role = RoleApp
			default:
				unauthorized(w, "invalid credentials")
				return
			}

			ctx := context.WithValue(r.Context(), roleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole gates a route group to a specific credential. Applied to
// /internal/*, it is what stops a phone token from reaching the ingest
// endpoint.
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
