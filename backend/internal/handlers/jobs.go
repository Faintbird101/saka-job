package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/jobhunter/backend/internal/ingest"
	"github.com/yourname/jobhunter/backend/internal/models"
	"github.com/yourname/jobhunter/backend/internal/service"
)

// ListJobs handles GET /jobs.
//
// Query parameters: status, min_score, country, q, limit, offset.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := service.JobFilter{
		Status:  strings.TrimSpace(q.Get("status")),
		Country: strings.TrimSpace(q.Get("country")),
		Search:  strings.TrimSpace(q.Get("q")),
	}

	if v := q.Get("min_score"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			h.badRequest(w, r, "min_score must be an integer")
			return
		}
		f.MinScore = &n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			h.badRequest(w, r, "limit must be an integer")
			return
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			h.badRequest(w, r, "offset must be an integer")
			return
		}
		f.Offset = n
	}

	page, err := h.svc.ListJobs(r.Context(), f)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, page)
}

// GetJob handles GET /jobs/{id}.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.badRequest(w, r, "job id is required")
		return
	}

	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, job)
}

// PatchJob handles PATCH /jobs/{id} — the approval action from the app, and
// the pipeline writes from n8n. The service layer enforces which transitions
// are legal; this only decodes.
func (h *Handler) PatchJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.badRequest(w, r, "job id is required")
		return
	}

	var patch models.JobUpdate
	if err := decodeJSON(w, r, &patch); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	job, err := h.svc.UpdateJob(r.Context(), id, patch)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, job)
}

// Ingest handles POST /internal/jobs/ingest — the n8n-only write that starts
// the whole pipeline.
//
// It always returns 200 with counts rather than failing on partial success:
// a batch where 8 of 10 jobs were duplicates is a completely normal run, and
// n8n should log the numbers, not retry.
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	var batch ingest.Batch
	if err := decodeJSON(w, r, &batch); err != nil {
		h.badRequest(w, r, "invalid ingest body: "+err.Error())
		return
	}

	// An empty batch is NOT short-circuited here. A search that matched nothing
	// is a legitimate outcome, and it still consumed an API call — so it still
	// has to reach the service and produce a fetch_log row. Returning early
	// would make "the 07:00 run found zero jobs" indistinguishable from "the
	// 07:00 run never happened", which is the exact question fetch_log exists
	// to answer.
	res, err := h.svc.Ingest(r.Context(), batch)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, res)
}

// RunScoring handles POST /internal/scoring/run — WF-B's entry point.
//
// n8n triggers it on a schedule; the backend owns the prompt, the model call,
// the strict JSON validation, and the threshold decision. Keeping all of that
// here rather than in n8n means the LLM key never leaves the backend, and the
// scoring rubric is versioned in git rather than inside a workflow node.
func (h *Handler) RunScoring(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Limit int `json:"limit"`
	}
	// An empty body is valid and means "use the default batch size", so a
	// decode failure on an absent body must not be an error.
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &body); err != nil {
			h.badRequest(w, r, "invalid request body: "+err.Error())
			return
		}
	}

	run, err := h.svc.ScoreNewJobs(r.Context(), body.Limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, run)
}
