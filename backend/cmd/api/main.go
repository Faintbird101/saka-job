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
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
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

	// Scoring is optional: without a key the API still boots and serves
	// everything else, and only /internal/scoring/run reports itself
	// unavailable. Refusing to start would take the whole pipeline down over a
	// stage that may not be configured yet.
	var scorer scoring.Client
	scorerModel := cfg.LLMModel
	if scorerModel == "" {
		scorerModel = scoring.DefaultModel
	}
	if c, err := scoring.NewAnthropicClient(cfg.LLMAPIKey, scorerModel); err != nil {
		log.Warn("scoring disabled", "reason", err)
	} else {
		scorer = c
		log.Info("scoring enabled", "model", c.Model())
	}

	svc := service.New(database, log, scorer, scorerModel)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handlers.Router(cfg, database, svc, log),
		// Generous write timeout relative to the handler timeout so a slow
		// response is cut off by chi's timeout middleware (which produces a
		// clean 504) rather than by the server dropping the connection.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      cfg.Timeout + 15*time.Second,
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
