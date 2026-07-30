// Command api is the JobHunter backend: the single process allowed to talk to
// Postgres, serving both the Flutter app and the n8n workflows.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/jobhunter/backend/internal/config"
	"github.com/yourname/jobhunter/backend/internal/db"
	"github.com/yourname/jobhunter/backend/internal/handlers"
	"github.com/yourname/jobhunter/backend/internal/scoring"
	"github.com/yourname/jobhunter/backend/internal/service"
	"github.com/yourname/jobhunter/backend/pkg/logger"
	"log/slog"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

// buildScorer resolves SCORING_MODE into a Scorer, or nil if the configured
// mode cannot be satisfied.
func buildScorer(cfg config.Config, log *slog.Logger) scoring.Scorer {
	if scoring.Mode(cfg.ScoringMode) == scoring.ModeLLM {
		client, model, err := scoring.NewLLMClient(
			scoring.Provider(cfg.LLMProvider), cfg.LLMAPIKey, cfg.LLMModel)
		if err != nil {
			log.Error("SCORING_MODE=llm but the client could not be built; scoring is disabled",
				"reason", err, "hint", "set SCORING_MODE=keyword for the free scorer")
			return nil
		}
		log.Info("scoring enabled", "mode", "llm", "provider", cfg.LLMProvider, "model", model)
		return scoring.NewLLMScorer(client, model)
	}

	log.Info("scoring enabled", "mode", "keyword", "cost", "free")
	return scoring.NewKeywordScorer()
}

// buildLLM constructs the model client if a usable key is configured. A nil
// return is a supported state: the API still serves everything else, and only
// generation reports itself unavailable.
func buildLLM(cfg config.Config, log *slog.Logger) (scoring.Client, string) {
	client, model, err := scoring.NewLLMClient(
		scoring.Provider(cfg.LLMProvider), cfg.LLMAPIKey, cfg.LLMModel)
	if err != nil {
		log.Warn("LLM features disabled (generation will report unavailable)", "reason", err)
		return nil, ""
	}
	log.Info("LLM client ready", "provider", cfg.LLMProvider, "model", model)
	return client, model
}

// run does the real work and returns an error, so every failure path gets the
// same treatment and deferred cleanup actually runs (os.Exit in main would
// skip it).
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel, cfg.LogFormat)
	log.Info("starting jobhunter backend", "env", cfg.Env, "port", cfg.Port)

	// Boot has its own deadline: if Postgres never comes up, we want a clear
	// failure rather than a container that hangs forever in "starting".
	bootCtx, cancelBoot := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelBoot()

	database, err := db.Connect(bootCtx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer database.Close()
	log.Info("connected to postgres")

	if err := db.Migrate(bootCtx, database, log); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Info("schema up to date")

	// Pick the scorer. Keyword mode is the default and always available; LLM
	// mode needs a key, and if it is missing we degrade to no scorer rather
	// than refusing to boot — the rest of the API must keep serving.
	scorer := buildScorer(cfg, log)

	// The model client is built regardless of SCORING_MODE, because generation
	// (WF-C) needs one even when scoring deliberately does not use one.
	llm, llmModel := buildLLM(cfg, log)

	svc := service.New(database, log, scorer, llm, llmModel, cfg.PublicBaseURL)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handlers.Router(cfg, database, svc, log),
		// These must not undercut the longest handler timeout, or the server
		// drops the connection before chi can return a clean 504 — and the
		// client sees an opaque "unexpected EOF" instead of a timeout.
		// Scoring is the longest route, so it sets the floor.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.ScoringTimeout + 30*time.Second,
		WriteTimeout:      cfg.ScoringTimeout + 30*time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Serve in the background so main can wait on signals.
	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info("shutdown signal received, draining connections")
	}

	// Graceful shutdown: let in-flight requests finish. An ingest run cut off
	// mid-batch would leave the fetch_log row unwritten and the counts wrong.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	log.Info("shutdown complete")
	return nil
}
