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

	// NOTE: the request timeout is applied per route group below, NOT here.
	// A root-level Timeout wraps every route, and a longer timeout nested
	// inside cannot override a shorter one outside it — the outer deadline
	// still fires first. Scoring needs minutes, so it gets its own group.

	// Unauthenticated. The health check has to answer before credentials are
	// configured, and it deliberately reveals nothing beyond up/down.
	r.Get("/health", health.Health)

	// The CV editor is a static shell with no data in it — a browser cannot
	// attach an auth header to a plain navigation, so gating the HTML would
	// make it unreachable. The page asks for the token and every call it makes
	// is authenticated like any other client. See profile_page.go.
	r.Get("/profile/edit", h.ProfilePage)

	// ---- app + n8n: either credential is accepted ----
	r.Group(func(r chi.Router) {
		r.Use(chimw.Timeout(cfg.Timeout))
		r.Use(middleware.Auth(cfg))

		r.Get("/jobs", h.ListJobs)
		r.Get("/jobs/{id}", h.GetJob)
		// Generated documents, served as markdown rather than wrapped in JSON —
		// they are prose meant to be read and edited.
		r.Get("/jobs/{id}/cv", h.GetCV)
		r.Get("/jobs/{id}/cover-letter", h.GetCoverLetter)
		// Inbound-reply timeline, and the queue of classifications awaiting a
		// human decision.
		r.Get("/jobs/{id}/events", h.JobEvents)
		r.Get("/events/pending", h.PendingEvents)
		r.Get("/events/unmatched", h.UnmatchedEvents)
		r.Post("/events/{id}/confirm", h.ConfirmEvent)
		// PATCH is shared on purpose: the app writes Approved/Rejected here and
		// n8n writes score and CV URLs through the same validated path, so the
		// state machine is enforced once rather than twice.
		r.Patch("/jobs/{id}", h.PatchJob)

		r.Get("/profile", h.GetProfile)
		r.Patch("/profile", h.PatchProfile)
		// Extracts text from an uploaded PDF/DOCX and returns it for review.
		// It does not save — the editor shows you the text first.
		r.Post("/profile/cv", h.UploadCV)

		r.Get("/stats", h.Stats)
		r.Get("/statuses", h.Statuses)
		r.Get("/fetch-logs", h.ListFetchLogs)
		r.Get("/errors", h.ListErrors)
		// Pruning matters: a permanently red error badge is one nobody reads.
		r.Delete("/errors", h.ClearErrors)
		// Re-run scoring for one job after tuning the CV, threshold, or weights,
		// without resetting the whole table by hand.
		r.Post("/jobs/{id}/rescore", h.RescoreJob)
	})

	// ---- n8n only ----
	// Ingestion and error reporting are automation-side concerns. Gating them
	// behind the n8n key means a leaked app token can't inject rows into the
	// jobs table.
	r.Group(func(r chi.Router) {
		r.Use(chimw.Timeout(cfg.Timeout))
		r.Use(middleware.Auth(cfg))
		r.Use(middleware.RequireRole(middleware.RoleN8N))

		r.Post("/internal/jobs/ingest", h.Ingest)
		r.Post("/internal/errors", h.RecordError)
		// WF-F: n8n fetches the mail, the backend matches and classifies it.
		r.Post("/internal/inbox/scan", h.ScanInbox)
		// WF-D: move Approved -> ManualApply and hand back a digest to notify
		// the candidate. Never contacts an employer.
		r.Post("/internal/apply-packs/run", h.RunApplyPacks)
	})

	// ---- n8n only, long-running ----
	// Scoring is a sibling group rather than nested inside the one above, so it
	// gets ScoringTimeout INSTEAD of the standard one. It makes a model call per
	// job, sequentially, and a batch of 10 legitimately takes minutes.
	//
	// This bit us: with a 30s deadline the request context died mid-batch, and
	// the remaining jobs were written off as ScoreFailed — a timeout presenting
	// as "the model returned garbage".
	r.Group(func(r chi.Router) {
		r.Use(chimw.Timeout(cfg.ScoringTimeout))
		r.Use(middleware.Auth(cfg))
		r.Use(middleware.RequireRole(middleware.RoleN8N))

		r.Post("/internal/scoring/run", h.RunScoring)
		// Generation is in the same long-running group: it writes two documents
		// of prose per job, so it is slower per job than scoring.
		r.Post("/internal/generation/run", h.RunGeneration)
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
