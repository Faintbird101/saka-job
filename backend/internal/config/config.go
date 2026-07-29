// Package config loads and validates every environment variable the backend
// reads, once, at startup.
//
// The rule: nothing outside this package calls os.Getenv. If a required
// secret is missing, Load fails loudly at boot rather than producing a 500
// somewhere deep in a handler three hours later.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config is the typed view of the environment.
type Config struct {
	// Runtime
	Env       string        // "dev" | "prod"
	Port      string        // API_PORT, the port we listen on inside the container
	LogLevel  string        // debug | info | warn | error
	LogFormat string        // json | text
	Timeout   time.Duration // per-request handler timeout
	// ScoringTimeout bounds /internal/scoring/run separately. Scoring is one
	// model call per job, sequentially, so a batch legitimately takes minutes —
	// far longer than any other endpoint should be allowed to take.
	ScoringTimeout time.Duration

	// Postgres
	DatabaseURL string

	// Auth: two separate credentials, see internal/middleware.
	AppToken string // API_AUTH_TOKEN  — the Flutter app
	N8NKey   string // N8N_API_KEY     — the automation workflows

	// Downstream services. Not used by the ingest path, but read here so the
	// later workflows have a single typed place to pick them up from.
	RapidAPIKey  string
	RapidAPIHost string
	LLMAPIKey    string
	LLMModel     string
	// LLMProvider is "gemini" (default) or "anthropic". Only consulted when
	// ScoringMode is "llm".
	LLMProvider string
	// ScoringMode selects the scorer: "keyword" (default, free, no key) or
	// "llm" (needs LLMAPIKey). See internal/scoring.
	ScoringMode string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
}

// IsDev reports whether we're running in development mode.
func (c Config) IsDev() bool { return strings.EqualFold(c.Env, "dev") }

// Load reads the environment (plus a local .env if present) into a Config.
//
// The .env file is a convenience for `go run` on the host; inside Docker the
// values arrive via env_file and godotenv finds nothing, which is fine.
func Load() (Config, error) {
	// Best-effort: walk up a couple of levels so `go run ./cmd/api` from
	// backend/ still finds the repo-root .env.
	for _, p := range []string{".env", "../.env", "../../.env", "../../../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	c := Config{
		Env:       getDefault("APP_ENV", "dev"),
		Port:      getDefault("API_PORT", "8080"),
		LogLevel:  getDefault("LOG_LEVEL", "info"),
		LogFormat: getDefault("LOG_FORMAT", "json"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		AppToken: os.Getenv("API_AUTH_TOKEN"),
		N8NKey:   os.Getenv("N8N_API_KEY"),

		RapidAPIKey:  os.Getenv("RAPIDAPI_KEY"),
		RapidAPIHost: os.Getenv("RAPIDAPI_HOST"),
		ScoringMode:  strings.ToLower(getDefault("SCORING_MODE", "keyword")),
		LLMProvider:  strings.ToLower(getDefault("LLM_PROVIDER", "gemini")),
		LLMAPIKey:    os.Getenv("LLM_API_KEY"),
		LLMModel:     os.Getenv("LLM_MODEL"),
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPass:     os.Getenv("SMTP_PASS"),
	}

	secs, err := strconv.Atoi(getDefault("REQUEST_TIMEOUT_SECONDS", "30"))
	if err != nil || secs <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT_SECONDS must be a positive integer, got %q", os.Getenv("REQUEST_TIMEOUT_SECONDS"))
	}
	c.Timeout = time.Duration(secs) * time.Second

	ssecs, err := strconv.Atoi(getDefault("SCORING_TIMEOUT_SECONDS", "900"))
	if err != nil || ssecs <= 0 {
		return Config{}, fmt.Errorf("SCORING_TIMEOUT_SECONDS must be a positive integer, got %q", os.Getenv("SCORING_TIMEOUT_SECONDS"))
	}
	c.ScoringTimeout = time.Duration(ssecs) * time.Second

	if p := strings.TrimSpace(os.Getenv("SMTP_PORT")); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return Config{}, fmt.Errorf("SMTP_PORT must be an integer, got %q", p)
		}
		c.SMTPPort = n
	}

	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// validate enforces only what the backend itself cannot run without. Keys for
// later pipeline stages (LLM, SMTP, RapidAPI) are deliberately *not* required
// here — the API must still boot and serve /jobs on a machine where scoring
// and sending haven't been configured yet.
func (c Config) validate() error {
	var missing []string
	for _, f := range []struct{ name, val string }{
		{"DATABASE_URL", c.DatabaseURL},
		{"API_AUTH_TOKEN", c.AppToken},
		{"N8N_API_KEY", c.N8NKey},
	} {
		if strings.TrimSpace(f.val) == "" {
			missing = append(missing, f.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// A shared token between the app and n8n would silently collapse the two
	// privilege levels into one.
	if c.AppToken == c.N8NKey {
		return fmt.Errorf("API_AUTH_TOKEN and N8N_API_KEY must be different values")
	}

	switch c.ScoringMode {
	case "keyword", "llm":
	default:
		return fmt.Errorf("SCORING_MODE must be \"keyword\" or \"llm\", got %q", c.ScoringMode)
	}
	return nil
}

func getDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
