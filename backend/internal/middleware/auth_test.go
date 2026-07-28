package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourname/jobhunter/backend/internal/config"
)

var testCfg = config.Config{
	AppToken: "app-token-value",
	N8NKey:   "n8n-key-value",
}

// okHandler records the role it saw, so tests can assert on what Auth decided.
func okHandler(seen *Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role, ok := RoleFrom(r.Context()); ok && seen != nil {
			*seen = role
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthAcceptsBothCredentials(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		value    string
		wantRole Role
	}{
		{"app token as bearer", "Authorization", "Bearer app-token-value", RoleApp},
		{"app token as raw authorization", "Authorization", "app-token-value", RoleApp},
		{"app token as api key header", "X-API-Key", "app-token-value", RoleApp},
		{"n8n key as bearer", "Authorization", "Bearer n8n-key-value", RoleN8N},
		{"n8n key as api key header", "X-API-Key", "n8n-key-value", RoleN8N},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var seen Role
			req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
			req.Header.Set(tc.header, tc.value)
			rec := httptest.NewRecorder()

			Auth(testCfg)(okHandler(&seen)).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if seen != tc.wantRole {
				t.Errorf("role = %q, want %q", seen, tc.wantRole)
			}
		})
	}
}

func TestAuthRejectsBadCredentials(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"no credentials at all", "", ""},
		{"wrong token", "Authorization", "Bearer nope"},
		{"empty bearer", "Authorization", "Bearer "},
		{"empty api key", "X-API-Key", ""},
		// A blank configured secret must never match a blank header.
		{"whitespace only", "X-API-Key", "   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			rec := httptest.NewRecorder()

			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
			Auth(testCfg)(next).ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if called {
				t.Error("the wrapped handler ran despite failed authentication")
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// The whole reason for two separate secrets: a leaked app token must not be
// able to reach the ingest endpoint.
func TestRequireRoleBlocksTheAppTokenFromInternalRoutes(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/ingest", nil)
	req.Header.Set("Authorization", "Bearer app-token-value")
	rec := httptest.NewRecorder()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	Auth(testCfg)(RequireRole(RoleN8N)(next)).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if called {
		t.Error("an app-token request reached an n8n-only handler")
	}
}

func TestRequireRoleAllowsTheN8NKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/ingest", nil)
	req.Header.Set("X-API-Key", "n8n-key-value")
	rec := httptest.NewRecorder()

	var seen Role
	Auth(testCfg)(RequireRole(RoleN8N)(okHandler(&seen))).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if seen != RoleN8N {
		t.Errorf("role = %q, want n8n", seen)
	}
}

// RequireRole must fail closed if it is ever mounted without Auth in front.
func TestRequireRoleFailsClosedWithoutAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/internal/jobs/ingest", nil)
	rec := httptest.NewRecorder()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	RequireRole(RoleN8N)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	if called {
		t.Error("handler ran with no authenticated role on the context")
	}
}

func TestRequestIDIsGeneratedAndEchoed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	var got string
	RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFrom(r.Context())
	})).ServeHTTP(rec, req)

	if got == "" {
		t.Fatal("no request id on the context")
	}
	if rec.Header().Get("X-Request-ID") != got {
		t.Errorf("header %q does not match context id %q", rec.Header().Get("X-Request-ID"), got)
	}
}

func TestRequestIDPreservesAnIncomingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "trace-from-caddy")
	rec := httptest.NewRecorder()

	var got string
	RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFrom(r.Context())
	})).ServeHTTP(rec, req)

	if got != "trace-from-caddy" {
		t.Errorf("request id = %q, want the incoming value preserved for end-to-end tracing", got)
	}
}
