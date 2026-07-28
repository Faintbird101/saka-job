// Package handlers is the HTTP surface: decode, delegate to service, render.
//
// No SQL and no business rules live here. If a handler grows an `if` about
// what a status means, that `if` belongs in internal/service.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/yourname/jobhunter/backend/internal/middleware"
	"github.com/yourname/jobhunter/backend/internal/service"
)

// maxBodyBytes caps request bodies. The ingest payload carries full job
// descriptions for a whole page of results, so it needs real headroom — but
// not unbounded, or one malformed request can exhaust the container's memory.
const maxBodyBytes = 8 << 20 // 8 MiB

// Handler bundles the dependencies every route needs.
type Handler struct {
	svc *service.Service
	log *slog.Logger
}

// New builds the handler set.
func New(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// errorBody is the single error envelope every failure uses, so the Flutter
// app has exactly one shape to parse.
type errorBody struct {
	Error struct {
		Message   string `json:"message"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"error"`
}

func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent, so there's nothing to do but note it.
		h.log.Error("failed to encode response",
			"error", err, "path", r.URL.Path,
			"request_id", middleware.RequestIDFrom(r.Context()))
	}
}

// writeError maps a service error onto a status code.
//
// This mapping is the only place the sentinel errors are interpreted, which is
// why service methods can return plain wrapped errors and stay HTTP-agnostic.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	switch {
	case errors.Is(err, service.ErrNotFound):
		code, msg = http.StatusNotFound, "not found"
	case errors.Is(err, service.ErrInvalidInput):
		code, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, service.ErrConflict):
		code, msg = http.StatusConflict, err.Error()
	default:
		// Unexpected errors are logged in full but reported generically:
		// a raw pgx error can leak column names and the connection string.
		h.log.Error("request failed",
			"error", err, "path", r.URL.Path, "method", r.Method,
			"request_id", middleware.RequestIDFrom(r.Context()))
	}

	var body errorBody
	body.Error.Message = msg
	body.Error.RequestID = middleware.RequestIDFrom(r.Context())
	h.writeJSON(w, r, code, body)
}

// badRequest is the shortcut for input the handler itself rejected.
func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	var body errorBody
	body.Error.Message = msg
	body.Error.RequestID = middleware.RequestIDFrom(r.Context())
	h.writeJSON(w, r, http.StatusBadRequest, body)
}

// decodeJSON reads a size-limited body and rejects unknown fields.
//
// DisallowUnknownFields is deliberate: a typo'd key in an n8n HTTP node would
// otherwise be silently dropped, and you'd spend an evening wondering why
// `{"statuz": "Approved"}` did nothing.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Exactly one JSON value per request; trailing garbage is a client bug.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("body must contain a single JSON value")
	}
	return nil
}
