package handlers

import (
	_ "embed"
	"net/http"
)

// profilePageHTML is the CV editor, compiled into the binary so the distroless
// image has no static directory to mount.
//
//go:embed profile_page.html
var profilePageHTML string

// ProfilePage serves the CV editor at GET /profile/edit.
//
// It is deliberately UNAUTHENTICATED, and that is safe: the page is a static
// shell containing no data and no secrets. A browser cannot attach a custom
// auth header to a plain navigation, so gating the HTML itself would make it
// unreachable without a proxy or a cookie scheme. Everything that actually
// touches data goes through fetch() with the token the page asks you for, and
// the API enforces auth there — the same enforcement as every other client.
func (h *Handler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page talks only to its own origin, loads no third-party assets, and
	// runs no inline event handlers beyond its own script block.
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(profilePageHTML))
}
