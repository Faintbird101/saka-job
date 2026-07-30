package handlers

import (
	"net/http"

	"github.com/yourname/jobhunter/backend/internal/middleware"
)

// AuthStatus handles GET /auth/status — unauthenticated, so the app can tell
// whether to show a signup or a login screen on first launch.
//
// It reveals only whether an account exists, which a login attempt would reveal
// anyway.
func (h *Handler) AuthStatus(w http.ResponseWriter, r *http.Request) {
	needs, err := h.svc.NeedsBootstrap(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"needs_setup": needs})
}

// Bootstrap handles POST /auth/bootstrap — creates the first and only account.
//
// Unauthenticated by necessity: there is nothing to authenticate against yet.
// It closes permanently once an account exists, so an internet-reachable API
// cannot be used by anyone else to create themselves an account.
func (h *Handler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	user, err := h.svc.Bootstrap(r.Context(), body.Email, body.Password, body.DisplayName)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusCreated, user)
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	res, err := h.svc.Login(r.Context(), body.Email, body.Password, r.UserAgent())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// The token is returned exactly once and stored only as a hash. If the app
	// loses it, log in again.
	h.writeJSON(w, r, http.StatusOK, res)
}

// Logout handles POST /auth/logout — revokes the session that made the call.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.TokenFrom(r.Context())
	if token == "" {
		// A service token has no session to revoke; say so rather than
		// silently succeeding.
		h.badRequest(w, r, "not a session — nothing to log out")
		return
	}
	if err := h.svc.Logout(r.Context(), token); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"logged_out": true})
}

// LogoutEverywhere handles POST /auth/logout-all — revokes every session.
// This is what to hit after losing a phone, and the reason sessions are
// server-side rather than self-contained tokens.
func (h *Handler) LogoutEverywhere(w http.ResponseWriter, r *http.Request) {
	sess, ok := middleware.SessionFrom(r.Context())
	if !ok {
		h.badRequest(w, r, "not a session")
		return
	}
	if err := h.svc.LogoutEverywhere(r.Context(), sess.UserID); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"revoked": true})
}

// Me handles GET /auth/me — who the caller is.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	sess, ok := middleware.SessionFrom(r.Context())
	if !ok {
		// A valid service token, but not a person.
		h.writeJSON(w, r, http.StatusOK, map[string]any{"service_token": true})
		return
	}
	h.writeJSON(w, r, http.StatusOK, sess)
}

// RegisterDevice handles POST /devices — register an FCM token for push.
func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}

	userID := ""
	if sess, ok := middleware.SessionFrom(r.Context()); ok {
		userID = sess.UserID
	}

	device, err := h.svc.RegisterDevice(r.Context(), userID, body.Token, body.Platform)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, device)
}

// UnregisterDevice handles DELETE /devices — stop sending push to this device.
func (h *Handler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.badRequest(w, r, "invalid request body: "+err.Error())
		return
	}
	if err := h.svc.UnregisterDevice(r.Context(), body.Token); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, map[string]any{"unregistered": true})
}
