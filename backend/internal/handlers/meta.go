package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/jobhunter/backend/internal/models"
)

// GetProfile handles GET /profile.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.GetProfile(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, p)
}

// PatchProfile handles PATCH /profile.
func (h *Handler) PatchProfile(w http.ResponseWriter, r *http.Request) {
	var patch models.ProfileUpdate
	if err := decodeJSON(w, r, &patch); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	p, err := h.svc.UpdateProfile(r.Context(), patch)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, p)
}

// Stats handles GET /stats — the dashboard.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Stats(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, s)
}

// ListFetchLogs handles GET /fetch-logs — API quota consumption history.
func (h *Handler) ListFetchLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.svc.ListFetchLogs(r.Context(), intParam(r, "limit"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"fetch_logs": logs})
}

// ListErrors handles GET /errors — the application error feed.
func (h *Handler) ListErrors(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListErrors(r.Context(), intParam(r, "limit"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"errors": list})
}

// RecordError handles POST /internal/errors — this is the endpoint n8n's
// error-trigger workflow posts to, which is what makes workflow failures
// visible in the app instead of only in n8n's own execution list.
func (h *Handler) RecordError(w http.ResponseWriter, r *http.Request) {
	var rec models.ErrorRecord
	if err := decodeJSON(w, r, &rec); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	saved, err := h.svc.RecordError(r.Context(), rec)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusCreated, saved)
}

// Statuses handles GET /statuses — the state machine, served to the app so the
// approval screen doesn't hardcode a second copy of the transition graph.
func (h *Handler) Statuses(w http.ResponseWriter, r *http.Request) {
	next := make(map[string][]string, len(models.AllStatuses))
	for _, s := range models.AllStatuses {
		n := models.NextStatuses(s)
		if n == nil {
			n = []string{}
		}
		next[s] = n
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{
		"statuses":    models.AllStatuses,
		"transitions": next,
	})
}

// intParam reads an optional integer query parameter, returning 0 (which every
// caller treats as "use the default") when absent or unparseable.
func intParam(r *http.Request, name string) int {
	n, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		return 0
	}
	return n
}

// ClearErrors handles DELETE /errors — prune the application error log.
//
// Optional ?older_than_hours=N keeps recent entries. Without it, everything
// goes.
func (h *Handler) ClearErrors(w http.ResponseWriter, r *http.Request) {
	var cutoff *time.Time
	if n := intParam(r, "older_than_hours"); n > 0 {
		t := time.Now().Add(-time.Duration(n) * time.Hour)
		cutoff = &t
	}

	deleted, err := h.svc.ClearErrors(r.Context(), cutoff)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"deleted": deleted})
}

// RescoreJob handles POST /jobs/{id}/rescore — re-run scoring for one job.
func (h *Handler) RescoreJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.badRequest(w, r, "job id is required")
		return
	}

	job, err := h.svc.RescoreJob(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, job)
}
