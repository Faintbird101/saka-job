// Package db owns the Postgres connection pool and the migration runner.
//
// The backend is the only process that talks to Postgres, so this is the one
// place a connection is opened and the one place the schema is advanced.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx pool. It exists so the rest of the codebase depends on our
// type rather than on pgxpool directly.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect opens the pool and verifies it with a ping.
//
// It retries, because on `docker compose up` the backend container reliably
// wins the race against Postgres finishing its first-boot initialisation.
// Failing fast here would just mean a crash-loop until the DB is ready.
func Connect(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	const attempts = 10
	var lastErr error
	for i := 0; i < attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = pool.Ping(pingCtx)
		cancel()
		if lastErr == nil {
			return &DB{Pool: pool}, nil
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}

	pool.Close()
	return nil, fmt.Errorf("postgres unreachable after %d attempts: %w", attempts, lastErr)
}

// Close releases every pooled connection.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}

// Health is the check behind GET /health. A pool that exists but can't round
// trip a query is not healthy, so this does a real query rather than just
// checking that the struct is non-nil.
func (d *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var one int
	if err := d.Pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}
