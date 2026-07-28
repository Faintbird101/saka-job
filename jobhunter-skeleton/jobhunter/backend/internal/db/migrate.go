package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

// The .sql files are compiled into the binary, so the distroless image has no
// migrations directory to mount and `docker compose up` is the only step
// needed to get a correct schema.
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// advisoryLockID is an arbitrary but fixed key. Postgres serialises the
// session-level advisory lock, so two backend replicas starting at once can't
// both try to apply 0002.
const advisoryLockID int64 = 8_15_2026

// Migrate applies every embedded migration that hasn't run yet, in filename
// order, each inside its own transaction.
func Migrate(ctx context.Context, d *DB, log *slog.Logger) error {
	names, err := migrationNames()
	if err != nil {
		return err
	}

	conn, err := d.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		// Best effort: the lock is session-scoped, so releasing the connection
		// would drop it anyway. Explicit is better.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", advisoryLockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		    version     TEXT PRIMARY KEY,
		    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, d)
	if err != nil {
		return err
	}

	// Baseline: this project's schema was originally applied by hand with
	// psql, before there was a runner. If the tables are already there but
	// schema_migrations is empty, record the existing migrations as applied
	// instead of re-running them into a "relation already exists" error.
	if len(applied) == 0 {
		var exists bool
		if err := conn.QueryRow(ctx, "SELECT to_regclass('public.jobs') IS NOT NULL").Scan(&exists); err != nil {
			return fmt.Errorf("probe for existing schema: %w", err)
		}
		if exists {
			log.Warn("existing schema found with no migration history; baselining", "versions", names)
			for _, n := range names {
				if _, err := conn.Exec(ctx,
					"INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", n); err != nil {
					return fmt.Errorf("baseline %s: %w", n, err)
				}
				applied[n] = true
			}
		}
	}

	for _, name := range names {
		if applied[name] {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
		log.Info("migration applied", "version", name)
	}

	return nil
}

func migrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no embedded migrations found")
	}
	return names, nil
}

func appliedVersions(ctx context.Context, d *DB) (map[string]bool, error) {
	rows, err := d.Pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}
