# JobHunter

An automated, human-supervised job application pipeline. It fetches job
postings from an API on a schedule, scores each one against your profile,
generates a tailored CV and cover letter for the good matches, waits for you
to approve, then sends the application and tracks follow-ups — all managed
from a Flutter app instead of scattered across spreadsheets and dashboards.

The guiding principle throughout: **build a thin slice that works end to end,
then widen it.** Nothing here is automated past the point where a human should
stay in the loop — you approve every application before it goes out.

---

## Table of contents

1. [What this does](#what-this-does)
2. [Architecture at a glance](#architecture-at-a-glance)
3. [Why these technology choices](#why-these-technology-choices)
4. [The state machine — the heart of the system](#the-state-machine)
5. [Repository layout](#repository-layout)
6. [The data model](#the-data-model)
7. [Deduplication strategy](#deduplication-strategy)
8. [The automation workflows (n8n)](#the-automation-workflows)
9. [Infrastructure & hardening](#infrastructure--hardening)
10. [Getting started](#getting-started)
11. [Build order / roadmap](#build-order--roadmap)
12. [Security notes](#security-notes)
13. [Glossary](#glossary)

---

## What this does

The pipeline moves a job posting through a series of stages, with a human
gate in the middle:

```
fetch jobs  ->  de-duplicate  ->  score vs your profile  ->  store
     ->  (if score high enough)  generate tailored CV + cover letter
     ->  YOU approve / reject / request edits
     ->  (if approved)  send application email
     ->  track, and follow up after 7-14 days if no reply
```

Every stage reads its work from a database and writes its result back, so any
stage can be re-run in isolation and nothing is lost if a step fails.

---

## Architecture at a glance

Three moving parts, one source of truth:

```
+--------------+     HTTPS      +--------------+              +------------+
| Flutter app  | -------------> |  Go backend  | -----------> |  Postgres  |
| (manage/     | <------------- |  (REST API)  | <----------- | (the state |
|  approve)    |                |              |              |  machine)  |
+--------------+                +--------------+              +------------+
                                       ^                              ^
                                       | writes jobs / reads config   |
                                +--------------+                      |
                                |     n8n      | ---------------------+
                                | (automation) |
                                +--------------+
                                       |
                    +------------------+------------------+
                    v                  v                  v
               RapidAPI            LLM API             SMTP
             (job source)        (score + CV)      (send email)
```

**The Go backend is the only thing that touches the database.** Both the app
and n8n go through it (or, for a couple of high-volume writes, n8n may hit
Postgres directly — decided per case). This keeps validation, deduplication,
and business rules in exactly one place instead of scattered across three
systems.

**Postgres is not just storage — it is the state machine.** Every job row has
a `status`, and each workflow only picks up rows in the status it cares about.
This one decision is what makes the whole system robust: idempotent,
re-runnable, and debuggable. If a workflow crashes halfway, the row keeps its
previous status and gets retried on the next run. No lost data, no stuck
executions, no fragile multi-day "wait" states.

---

## Why these technology choices

Each choice was made for a reason, not by default. Understanding the *why*
makes the system easier to extend and to reason about when something breaks.

### Go for the backend
Compiles to a single static binary that drops straight into Docker, has a
tiny memory footprint, and its standard library covers most of what a REST
API needs. Fast to run, simple to deploy, and a good language to learn
properly. Stack: `net/http` + `chi` (a minimal router) + `pgx` (the best
Postgres driver for Go).

### Flutter for the app
One codebase for Android, iOS, and web. Approving a job from your phone is
genuinely useful, and Flutter compiles to web too if you want a desktop
dashboard later. Laid out **feature-first** so each screen owns its own state
and UI, rather than a giant shared `screens/` folder that rots over time.

### Postgres for the database
A real relational database. The pipeline asks relational questions constantly
— "jobs applied 7+ days ago with no reply", "count of jobs at each stage",
"high-score jobs not yet generated". SQL answers these cleanly; a NoSQL store
would make them clumsy. Postgres's `JSONB` columns also let us store the
API's rich nested data without flattening everything.

### n8n for automation
A self-hosted, visual workflow engine. It handles the schedule, the HTTP
calls, the branching, and the retries without you writing a scheduler. Because
it's self-hosted (Docker), you can run custom code and community nodes freely.
Workflows are exported as JSON and version-controlled in `n8n/workflows/`.

### Caddy as the reverse proxy
Automatic HTTPS with effectively zero config, and a clean single point to
enforce security headers and rate limits. It's the only service exposed to the
outside world; everything else lives on an internal Docker network behind it.

### Beszel for monitoring
A lightweight resource-monitoring dashboard (CPU, memory, disk, container
stats) — far simpler than a Prometheus/Grafana stack, which is overkill for a
single VPS. It watches the *infrastructure*; application errors are handled
separately (see below).

### A note on what monitors what
- **Beszel** = system health. "Is the box healthy, is a container eating RAM?"
- **`errors` table + n8n error-trigger** = application health. "Did scoring
  fail on job X, did an email bounce?"

These two layers don't overlap, and you want both.

---

## The state machine

Everything revolves around the `jobs.status` column. Here is the full life
cycle. The **happy path** runs top to bottom; the **side states** are where a
job leaves the main flow.

```
 New  ->  Scored  ->  CVGenerated  ->  AwaitingApproval  ->  Approved  ->  Applied  ->  FollowUpSent  ->  Closed
  |         |                                |                  |
  |         v                                v                  v
  |      LowMatch                         Rejected           ManualApply
  |      (below threshold)                (you said no)      (portal-only, by hand)
  v
 ScoreFailed
  (LLM returned unparseable output)
```

| Status             | Meaning                                             | Set by            |
|--------------------|-----------------------------------------------------|-------------------|
| `New`              | Just ingested, not yet scored                       | Ingestion (WF-A)  |
| `Scored`           | Scored at or above threshold, awaiting CV           | Scoring (WF-B)    |
| `LowMatch`         | Scored below threshold, parked                      | Scoring (WF-B)    |
| `ScoreFailed`      | LLM output couldn't be parsed; needs a retry        | Scoring (WF-B)    |
| `CVGenerated`      | Tailored CV + cover letter produced                 | Generation (WF-C) |
| `AwaitingApproval` | Sent to you, waiting for your decision              | Generation (WF-C) |
| `Approved`         | You approved it; ready to send                      | You (via the app) |
| `Rejected`         | You declined it                                     | You (via the app) |
| `Applied`          | Application email sent                               | Sending (WF-D)    |
| `ManualApply`      | Portal-only job flagged for you to apply by hand    | Sending (WF-D)    |
| `FollowUpSent`     | Follow-up email sent after no reply                 | Follow-up (WF-E)  |
| `Closed`           | Terminal — done, whatever the outcome               | Follow-up (WF-E)  |

The status values are enforced at the database level by a `CHECK` constraint,
so a typo like `"Aproved"` is rejected by Postgres rather than silently
stranding a row in a status no workflow will ever pick up.

---

## Repository layout

A monorepo: backend, app, automation, and infra in one place so they stay in
sync and one `docker compose up` runs everything.

```
jobhunter/
├── README.md                 <- you are here
├── .env.example              <- copy to .env and fill in secrets
├── .gitignore
│
├── backend/                  <- Go REST API (source of truth)
│   ├── cmd/api/main.go       <- thin entrypoint: config -> db -> router -> serve
│   ├── go.mod
│   └── internal/             <- all real logic (Go's "private to this module")
│       ├── config/           <- typed config loaded from env vars
│       ├── db/
│       │   ├── db.go         <- pgx connection pool
│       │   ├── migrations/   <- 0001_init.sql (the schema)
│       │   └── queries/      <- SQL query strings
│       ├── models/           <- Job, Profile structs + API payload mapping
│       ├── handlers/         <- HTTP handlers (jobs, profile, stats, health)
│       ├── middleware/       <- auth (app token + n8n key), logging, recover
│       ├── service/          <- business logic between handlers and db
│       ├── ingest/           <- normalize API payload + dedup + insert
│       │   └── urlnorm.go    <- URL normalization for dedup (see below)
│       └── scoring/          <- LLM scoring: prompt, call, parse, update
│
├── mobile/                   <- Flutter app (feature-first)
│   ├── pubspec.yaml
│   └── lib/
│       ├── main.dart
│       ├── core/             <- shared plumbing (network, theme, config, error)
│       ├── data/             <- models, datasources, repositories
│       └── features/         <- one folder per screen, each owning its state
│           ├── jobs/         <- list: filter by status, sort by score
│           ├── job_detail/   <- full posting + AI score + CV preview
│           ├── approval/     <- approve / reject / request-edit actions
│           ├── profile/      <- master CV, keywords, score threshold
│           └── dashboard/    <- counts per pipeline stage
│
├── n8n/
│   └── workflows/            <- exported workflow JSON, version-controlled
│
├── infra/
│   ├── docker-compose.yml    <- the whole stack
│   ├── Caddyfile             <- reverse proxy + HTTPS + security headers
│   └── docker/               <- backend Dockerfile
│
└── docs/                     <- architecture, schema, runbook notes
```

**Why `internal/`?** In Go, code under `internal/` cannot be imported by other
modules — it enforces that these are private implementation details. `cmd/`
holds entrypoints; `pkg/` (if present) holds genuinely reusable helpers.

---

## The data model

Three tables plus a log. Full definitions live in
`backend/internal/db/migrations/0001_init.sql`.

### `jobs` — the core table
One row per unique job posting. Holds three kinds of data:

1. **Facts from the API** — title, organization, URL, description, dates,
   location, employment type, seniority.
2. **AI-enriched fields the API already provides** — `ai_key_skills`,
   `ai_keywords`, `ai_requirements_summary`, `ai_work_arrangement`,
   `ai_experience_level`. This is a real cost saver: the source API has already
   extracted skills and summaries, so our own LLM step scores against these
   instead of paying tokens to re-extract them.
3. **Our pipeline fields** — `status`, `score`, `matched_skills`,
   `missing_skills`, `ai_summary` (our scoring model's take), `cv_url`,
   `cover_letter_url`, `prompt_used` (audit trail), `date_applied`,
   `email_used`, and `raw_payload` (the full original JSON, kept so we can
   reprocess without re-fetching).

### `profile` — your config (a single row)
The one place the app writes settings that n8n reads: your `master_cv`,
`search_titles`, `preferred_skills`, `min_score_threshold` (default 70), and
`max_jobs_per_run` (default 10 — a rate-limit guard). Enforced as a singleton
via `CHECK (id = 1)`.

### `errors` — application error log
Every workflow/handler failure lands here: which workflow, which job, the
message, and JSON context. Paired with n8n's error-trigger for alerts.

### `fetch_log` — API consumption tracking
One row per ingest run: how many jobs the API returned, how many were new, how
many were duplicates skipped. This makes your rate-limit usage *visible*
instead of guesswork — important because the source API has a monthly quota.

**On `JSONB`:** list-shaped data (skills, keywords) and the raw payload are
stored as `JSONB` rather than flattened into extra tables. It keeps the schema
simple and lets Postgres query inside the JSON when needed.

---

## Deduplication strategy

The requirement is simple to state — "don't process the same job twice" — but
the real API data makes it subtle, so we use **three layered guards**. If any
one of them matches an existing row, the job is skipped.

1. **`source_job_id`** (the API's own `id`) — the strongest guard. Stable
   across runs. Primary defense.

2. **`linkedin_id`** — the real LinkedIn numeric job id. This catches a case
   that raw-URL matching misses: the *same* posting appears under different
   country subdomains — `vn.linkedin.com/...`, `mx.linkedin.com/...`,
   `www.linkedin.com/...` — with different URL strings but an *identical*
   trailing id. `UNIQUE` but nullable, because non-LinkedIn sources won't have
   one (Postgres allows multiple NULLs under a unique constraint).

3. **`normalized_url`** — a fallback for non-LinkedIn sources with no id. The
   URL is lowercased, query string stripped, and `www.`/country subdomains
   collapsed, so `vn.linkedin.com/jobs/view/x-123` and
   `www.linkedin.com/jobs/view/x-123` reduce to the same key. The logic lives
   in `backend/internal/ingest/urlnorm.go`.

Why bother with all three? Because the sample data proved plain URL-equality
would let the same job through under a different subdomain. Three nets is
cheap insurance against a database full of near-duplicates.

> **Data-quality note:** the source API's title filter is loose — a search for
> "Flutter" can return React, Angular, or even recruiter postings. And its
> derived geo fields (`regions_derived`, `lats_derived`) are sometimes wrong
> (one sample placed a Monterrey job in Chiapas). So: the **scoring** step is
> what actually filters relevance, and we trust the **raw address country**
> over the derived geo fields.

---

## The automation workflows

Each is a separate n8n workflow, triggered on its own schedule, reading its
input state from the database. Keeping them separate (rather than one giant
flow) means each can be re-run from the middle and debugged in isolation.

| ID   | Workflow      | Trigger              | Reads          | Writes              |
|------|---------------|----------------------|----------------|---------------------|
| WF-A | Ingest        | Cron, twice daily    | RapidAPI       | `New` jobs          |
| WF-B | Score         | Cron / after ingest  | `New`          | `Scored`/`LowMatch` |
| WF-C | Generate      | Cron                 | `Scored`       | `AwaitingApproval`  |
| WF-D | Send          | Cron                 | `Approved`     | `Applied`           |
| WF-E | Follow-up     | Daily cron           | `Applied` 7d+  | `FollowUpSent`      |

**Fetching is deliberately limited to twice a day** (e.g. cron `0 7,19 * * *`)
and capped at `max_jobs_per_run`, to stay well inside the API's rate limit.
Each run writes a `fetch_log` row so consumption stays visible.

**On the human-approval step:** rather than pausing an n8n execution for days
(fragile — a container restart can lose a waiting execution), approval is pure
state-machine. WF-C sets a job to `AwaitingApproval` and stops. You approve in
the app, which flips the row to `Approved`. WF-D independently picks up
`Approved` rows on its schedule. No long-lived waits, nothing lost on restart.

**Follow-ups work the same way** — not a multi-day timer, but a daily query
for `Applied` jobs older than 7 days with no reply.

---

## Infrastructure & hardening

The whole stack runs via `infra/docker-compose.yml`. The key hardening
principle: **only Caddy is exposed to the host.** Everything else —
backend, n8n, Postgres, Beszel — is reachable only on the internal Docker
network.

```
host :80/:443 -> Caddy -+-> /api/*      backend:8080
                        +-> /n8n/*      n8n:5678
                        +-> /monitor/*  beszel:8090
                 postgres      (internal only — no host port)
                 beszel-agent  (host network, collects metrics)
```

- **HTTPS in dev:** Caddy's internal CA issues a real (self-signed) cert for
  `localhost`. Your browser/curl will warn — that's expected; use `curl -k`
  locally.
- **HTTPS in prod:** edit the `Caddyfile` — replace `localhost` with your
  domain and remove the `tls internal` line. Caddy then auto-fetches a real
  Let's Encrypt certificate. That is the *only* change.
- **Security headers** (HSTS, nosniff, DENY framing, etc.) are applied to
  every response at the edge.
- **Postgres** publishes no host port. To reach it from the host temporarily,
  either add `ports: ["5432:5432"]` to the service or
  `docker compose exec postgres psql -U jobhunter -d jobhunter`.

---

## Getting started

### Prerequisites
- Docker + Docker Compose
- Go 1.22+ (for backend development)
- Flutter 3.3+ (for app development)
- A RapidAPI key for the LinkedIn job-search API
- An LLM API key and SMTP credentials

### First run
```bash
# 1. Secrets
cp .env.example .env
#    then edit .env — fill in DB password, RapidAPI key, LLM key, SMTP.

# 2. Bring up the stack
cd infra
docker compose up -d

# 3. The schema applies itself — the backend embeds the migrations and runs
#    any that haven't been applied on start. (On a database that was set up by
#    hand before the runner existed, it records the existing migrations as
#    already applied rather than re-running them.) Nothing to do here.

# 4. Reach the services (self-signed cert in dev, so use -k with curl):
#    Backend API:  https://localhost/api/...
#    n8n UI:       https://localhost/n8n/
#    Monitoring:   https://localhost/monitor/
#
#    If infra/docker-compose.override.yml exists, host ports 80/443 were taken
#    on this machine and the edge is remapped — use https://localhost:8443/...

# 4b. Smoke-test it:
#    curl -k https://localhost:8443/api/health
#    curl -k -H "Authorization: Bearer $API_AUTH_TOKEN" https://localhost:8443/api/jobs

# 5. Import workflows from n8n/workflows/ into the n8n UI.

# 6. Run the app
cd ../mobile
flutter pub get
flutter run
```

### Beszel agent
After first boot, open `https://localhost/monitor/`, register the agent, copy
its key into `.env` as `BESZEL_AGENT_KEY`, then
`docker compose up -d beszel-agent`.

---

## Build order / roadmap

The project is built as a **thin vertical slice first** — one job flowing all
the way through — then widened. Recommended order:

1. [done] **Database schema** — the state machine everything hangs off.
2. [done] **Project skeleton + infra** — compose, Caddy, Beszel, layout.
3. [done] **Go backend foundation** — config, pgx pool, embedded migrations,
   two-token auth, request logging, panic recovery, graceful shutdown; the
   ingestion path (`POST /internal/jobs/ingest`: normalize a RapidAPI batch,
   compute the dedup keys, `INSERT ... ON CONFLICT DO NOTHING`, write a
   `fetch_log` row); read endpoints (`GET /jobs`, `/jobs/{id}`, `/profile`,
   `/stats`, `/statuses`, `/fetch-logs`, `/errors`); `PATCH /jobs/{id}` with
   state-machine enforcement. See `backend/README.md` for the API surface.
4. [~] **WF-A in n8n** — call RapidAPI twice daily -> hand the batch to the
   backend. *First runnable end-to-end piece: rows land in Postgres.*
   Workflow is written and committed (`n8n/workflows/01_ingest.json`); the
   backend half is verified. Blocked on two things only you can do: create the
   n8n owner account, and put a real `RAPIDAPI_KEY` in `.env` (the current
   value is still the `your_new_key_here` placeholder). See `n8n/README.md`.
5. [ ] **Minimal app screen** — the read endpoints exist, so this is Flutter
   work only: list ingested jobs from `GET /jobs`.
6. [ ] **Scoring** (WF-B + `/internal/scoring`) with strict JSON validation.
7. [ ] **CV/cover-letter generation** (WF-C).
8. [ ] **Approval in the app** -> `PATCH /jobs/{id}`.
9. [ ] **Sending + follow-ups** (WF-D, WF-E).
10. [ ] **Dashboard, audit trail, polish.**

---

## Security notes

- **Never commit `.env`** — it's gitignored. All secrets live there.
- **Rotate any API key that has ever been shared or pasted** into a chat,
  commit, or screenshot. Treat exposed keys as burned.
- **Two auth tokens:** the app sends an app token; n8n uses a separate API
  key. Both are validated by backend middleware.
- **Prefer APIs over scraping** — respect each source's terms and robots.txt.
- **Keep the human approval gate.** Beyond legality, blasting out
  auto-generated applications tends to work badly for you anyway.
- The backend is the single database gatekeeper, so validation and dedup can't
  be bypassed by a misbehaving client.

---

## Glossary

- **State machine** — the pattern where each job has a `status` and workflows
  act only on specific statuses. Makes the system re-runnable and crash-safe.
- **Idempotent** — running the same step twice produces the same result (no
  duplicates, no double-sends). The dedup guards and status checks give us this.
- **HITL (human-in-the-loop)** — the approval gate; no application sends
  without your explicit yes.
- **Normalization** — reshaping messy API data into our clean schema, incl.
  collapsing URL variants for dedup.
- **Reverse proxy** — Caddy, sitting in front of everything, terminating HTTPS
  and forwarding requests internally.

---

*Built incrementally. Each piece is validated before the next is added — the
difference between "works" and "an impressive diagram that breaks on run 2."*
