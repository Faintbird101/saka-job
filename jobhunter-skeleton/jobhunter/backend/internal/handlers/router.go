package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/yourname/jobhunter/backend/internal/config"
	"github.com/yourname/jobhunter/backend/internal/db"
	"github.com/yourname/jobhunter/backend/internal/middleware"
	"github.com/yourname/jobhunter/backend/internal/service"
)

// Router wires every route.
//
// Path note: Caddy proxies with `handle_path /api/*`, which STRIPS the /api
// prefix before forwarding. So the app calls https://host/api/jobs and this
// router sees /jobs. Do not add an /api prefix here or you'll get /api/api.
func Router(cfg config.Config, database *db.DB, svc *service.Service, log *slog.Logger) http.Handler {
	h := New(svc, log)
	health := NewHealth(database)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recover(log))
	r.Use(middleware.Logger(log))
	r.Use(chimw.Timeout(cfg.Timeout))

	// Unauthenticated. The health check has to answer before credentials are
	// configured, and it deliberately reveals nothing beyond up/down.
	r.Get("/health", health.Health)

	// ---- app + n8n: either credential is accepted ----
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg))

		r.Get("/jobs", h.ListJobs)
		r.Get("/jobs/{id}", h.GetJob)
		// PATCH is shared on purpose: the app writes Approved/Rejected here and
		// n8n writes score and CV URLs through the same validated path, so the
		// state machine is enforced once rather than twice.
		r.Patch("/jobs/{id}", h.PatchJob)

		r.Get("/profile", h.GetProfile)
		r.Patch("/profile", h.PatchProfile)

		r.Get("/stats", h.Stats)
		r.Get("/statuses", h.Statuses)
		r.Get("/fetch-logs", h.ListFetchLogs)
		r.Get("/errors", h.ListErrors)
	})

	// ---- n8n only ----
	// Ingestion and error reporting are automation-side concerns. Gating them
	// behind the n8n key means a leaked app token can't inject rows into the
	// jobs table.
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg))
		r.Use(middleware.RequireRole(middleware.RoleN8N))

		r.Post("/internal/jobs/ingest", h.Ingest)
		r.Post("/internal/errors", h.RecordError)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.writeJSON(w, r, http.StatusNotFound, map[string]any{
			"error": map[string]string{"message": "no such endpoint"},
		})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		h.writeJSON(w, r, http.StatusMethodNotAllowed, map[string]any{
			"error": map[string]string{"message": "method not allowed"},
		})
	})

	return r
}
