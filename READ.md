# JobHunter — automated job application pipeline

Monorepo:
- `backend/`  Go REST API (single source of truth, talks to Postgres)
- `mobile/`   Flutter app (manage jobs, approve/reject, edit profile)
- `n8n/`      Exported n8n workflows (scrape, score, generate, send, follow-up)
- `infra/`    docker-compose + Dockerfiles (Postgres, backend, n8n)
- `docs/`     architecture notes, schema docs, runbooks

## Quick start
1. `cp .env.example .env` and fill in secrets (DB, RapidAPI key, LLM, SMTP).
2. `cd infra && docker compose up -d`  (starts Postgres + n8n + backend)
3. Apply schema: migrations run on backend start (or manually via psql).
4. Import workflows from `n8n/workflows/` into the n8n UI.
5. Run the Flutter app: `cd mobile && flutter run`.

## Architecture
Flutter app  ──HTTP──▶  Go backend  ──▶  Postgres
                            ▲
                         n8n  ──▶  RapidAPI, LLM, SMTP
Postgres is the state machine; `jobs.status` drives every workflow.
