// Package logger builds the single structured logger the whole backend shares.
//
// Everything is JSON in production so the container logs stay greppable;
// text in dev so they stay readable while you're iterating.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger. level is one of debug/info/warn/error (default
// info); format is "json" or "text" (default json).
func New(level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var h slog.Handler
	if strings.EqualFold(format, "text") {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
