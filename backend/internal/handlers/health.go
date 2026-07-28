package handlers

import (
	"net/http"
	"time"

	"github.com/yourname/jobhunter/backend/internal/db"
)

// HealthHandler answers the liveness/readiness probe. It's separate from
// Handler because it must work without authentication and without the service
// layer — a health check that needs a working database *and* a valid token to
// tell you the database is down isn't much of a health check.
type HealthHandler struct {
	db      *db.DB
	started time.Time
}

// NewHealth builds the health endpoint.
func NewHealth(database *db.DB) *HealthHandler {
	return &HealthHandler{db: database, started: time.Now()}
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	UptimeS  int64  `json:"uptime_seconds"`
}

// Health handles GET /health.
//
// It reports 503 when Postgres is unreachable, so Docker (and Beszel) see a
// container that is up but not *ready* rather than a healthy-looking one that
// 500s on every real request.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:   "ok",
		Database: "ok",
		UptimeS:  int64(time.Since(h.started).Seconds()),
	}
	code := http.StatusOK

	if err := h.db.Health(r.Context()); err != nil {
		resp.Status = "degraded"
		resp.Database = "unreachable"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(healthJSON(resp))
}

func healthJSON(r healthResponse) []byte {
	return []byte(`{"status":"` + r.Status + `","database":"` + r.Database +
		`","uptime_seconds":` + itoa(r.UptimeS) + `}`)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
