package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/jobhunter/backend/internal/inbox"
)

// ScanInbox handles POST /internal/inbox/scan — WF-F's entry point.
//
// n8n's IMAP node does the fetching; this endpoint does the thinking. Keeping
// the matching and classification here rather than in the workflow means the
// rules are version-controlled and unit-tested, and the mailbox credential
// stays in n8n where it belongs.
func (h *Handler) ScanInbox(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Messages []struct {
			MessageID  string `json:"message_id"`
			From       string `json:"from"`
			Subject    string `json:"subject"`
			Body       string `json:"body"`
			ReceivedAt string `json:"received_at"`
		} `json:"messages"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid scan body: "+err.Error())
		return
	}

	msgs := make([]inbox.Message, 0, len(body.Messages))
	for _, m := range body.Messages {
		// A message with nothing to match on is noise; skipping it here keeps
		// the counts in the result honest.
		if strings.TrimSpace(m.From) == "" && strings.TrimSpace(m.Subject) == "" {
			continue
		}
		msgs = append(msgs, inbox.Message{
			MessageID:  m.MessageID,
			From:       m.From,
			Subject:    m.Subject,
			Body:       m.Body,
			ReceivedAt: m.ReceivedAt,
		})
	}

	res, err := h.svc.ScanInbox(r.Context(), msgs)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, res)
}

// JobEvents handles GET /jobs/{id}/events — the reply timeline for one job.
func (h *Handler) JobEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.badRequest(w, r, "job id is required")
		return
	}

	events, err := h.svc.EventsForJob(r.Context(), id, intParam(r, "limit"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"events": events})
}

// PendingEvents handles GET /events/pending — classifications awaiting your
// decision. This is the queue that exists because a rejection or an interview
// invitation is never applied on a guess.
func (h *Handler) PendingEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.PendingEvents(r.Context(), intParam(r, "limit"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"events": events})
}

// UnmatchedEvents handles GET /events/unmatched — mail the matcher could not
// attribute. Worth reviewing: it is the evidence that the rules need work.
func (h *Handler) UnmatchedEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.UnmatchedEvents(r.Context(), intParam(r, "limit"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"events": events})
}

// ConfirmEvent handles POST /events/{id}/confirm — accept or dismiss a
// suggestion. Body: {"accept": true|false}.
func (h *Handler) ConfirmEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.badRequest(w, r, "event id is required")
		return
	}

	var body struct {
		Accept *bool `json:"accept"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}
	// A pointer, so an absent field is rejected rather than silently read as
	// false — dismissing a suggestion by accident is not recoverable.
	if body.Accept == nil {
		h.badRequest(w, r, `"accept" is required and must be true or false`)
		return
	}

	event, err := h.svc.ConfirmEvent(r.Context(), id, *body.Accept)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, event)
}
