# Backend (Go)

The single source of truth. Both the Flutter app and the n8n workflows go
through this API; nothing else opens a connection to Postgres.

## Layout

- `cmd/api/`             entrypoint — config → db → migrate → router → serve, with graceful shutdown
- `internal/config/`     typed env config; the only package that calls `os.Getenv`
- `internal/db/`         pgx pool, health check, embedded migration runner
- `internal/db/queries/` every SQL string, in one place
- `internal/models/`     domain structs, API payload mapping, and the state machine
- `internal/middleware/` auth (app token + n8n key), request id, access log, panic recovery
- `internal/service/`    business rules: dedup, state transitions, validation
- `internal/ingest/`     normalize a RapidAPI batch → dedup keys (pure, no DB)
- `internal/handlers/`   HTTP: decode, delegate, render
- `internal/scoring/`    (WF-B, not built yet) call LLM, parse + validate JSON score
- `pkg/logger/`          slog setup

## Running

```bash
go test ./...          # unit tests, no database required
go run ./cmd/api       # needs DATABASE_URL reachable from the host
```

In practice the backend runs in Docker (`cd infra && docker compose up -d
--build backend`), because `DATABASE_URL` points at the compose-internal
hostname `postgres`, which doesn't resolve from the host.

## Auth

Two credentials, two privilege levels. Send either as `Authorization: Bearer
<token>` or `X-API-Key: <token>`.

| Credential       | Env var          | Can reach            |
|------------------|------------------|----------------------|
| App token        | `API_AUTH_TOKEN` | everything except `/internal/*` |
| n8n key          | `N8N_API_KEY`    | everything           |

They must be different values — `config.Load` refuses to start otherwise.
`/internal/*` is n8n-only so a leaked phone token cannot inject rows into the
jobs table.

## Endpoints

Caddy proxies with `handle_path /api/*`, which **strips** the prefix. The app
calls `/api/jobs`; this router sees `/jobs`.

| Method | Path                        | Auth  | Purpose |
|--------|-----------------------------|-------|---------|
| GET    | `/health`                   | none  | liveness + DB reachability (503 when Postgres is down) |
| GET    | `/profile/edit`             | none  | CV editor page (static shell, no data — see profile_page.go) |
| GET    | `/jobs`                     | any   | list; filters `status`, `min_score`, `country`, `q`, `limit`, `offset` |
| GET    | `/jobs/{id}`                | any   | one job, full description text |
| PATCH  | `/jobs/{id}`                | any   | partial update; enforces the state machine |
| POST   | `/jobs/{id}/rescore`        | any   | re-run scoring for one job after tuning |
| GET    | `/jobs/{id}/cv`             | any   | generated CV (`text/markdown`) |
| GET    | `/jobs/{id}/cover-letter`   | any   | generated cover letter (`text/markdown`) |
| GET    | `/profile`                  | any   | the singleton settings row |
| PATCH  | `/profile`                  | any   | update settings, incl. per-stage models |
| POST   | `/profile/cv`               | any   | upload PDF/DOCX/TXT → extracted text (does **not** save) |
| GET    | `/stats`                    | any   | count per pipeline stage, last fetch, 24h error count |
| GET    | `/statuses`                 | any   | the state machine + legal transitions |
| GET    | `/fetch-logs`               | any   | API quota consumption history |
| GET    | `/errors`                   | any   | application error feed |
| DELETE | `/errors`                   | any   | prune the log; `?older_than_hours=N` |
| POST   | `/internal/jobs/ingest`     | n8n   | **WF-A**: normalize a batch, dedup, insert, log the run |
| POST   | `/internal/scoring/run`     | n8n   | **WF-B**: score `New` jobs → `Scored`/`LowMatch` |
| POST   | `/internal/generation/run`  | n8n   | **WF-C**: `Scored` → CV + letter → `AwaitingApproval` |
| POST   | `/internal/errors`          | n8n   | workflow error reporting |

Twenty routes. `any` means either credential; `n8n` means the n8n key only.

### Ingest

Accepts either shape — the bare array is what you get by wiring n8n's RapidAPI
response straight into an HTTP Request node:

```jsonc
{"query_title": "Flutter", "notes": "morning run", "jobs": [ /* raw API items */ ]}
[ /* raw API items */ ]
```

Always returns 200 with counts; a batch that was entirely duplicates is a
normal run, not an error:

```json
{"returned": 3, "inserted": 2, "skipped": 1, "job_ids": ["..."], "fetch_log_id": "..."}
```

## The two rules this layer exists to enforce

**Dedup is Postgres's job.** Every insert is one
`INSERT ... ON CONFLICT DO NOTHING RETURNING id` with *no* conflict target, so
all three unique constraints (`source_job_id`, `linkedin_id`, `normalized_url`)
are checked atomically. No returned row means "duplicate". A read-then-write
check in Go would have a race window between the SELECT and the INSERT; this
has none. `internal/ingest` additionally removes duplicates *within* one batch,
because the API really does return the same posting twice when it's cross-listed
across country subdomains.

**Status transitions are validated in code.** The database `CHECK` constraint
rejects an invalid status *value*; `models.CanTransition` rejects an invalid
*move*. Without it, a client bug could flip a `New` job straight to `Applied`
and skip scoring, generation, and the human approval gate the whole system is
built around. Illegal moves return 409 with the list of legal ones.

## Error envelope

Every failure uses one shape, so the app has one thing to parse:

```json
{"error": {"message": "...", "request_id": "8044db83163e56ef"}}
```

`request_id` is also returned in the `X-Request-ID` header and appears on every
log line for that request. Unexpected errors are logged in full but reported
generically — a raw pgx error leaks column names and the connection string.
