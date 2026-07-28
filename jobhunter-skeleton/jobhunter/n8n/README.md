# n8n workflows

Workflows are exported as JSON and committed here, so the automation is
version-controlled alongside the code rather than living only inside a
container's SQLite file.

| File              | ID   | Status | What it does |
|-------------------|------|--------|--------------|
| `01_ingest.json`  | WF-A | built  | Cron → read profile → RapidAPI → `POST /internal/jobs/ingest` |
| `02_score.json`   | WF-B | todo   | `New` jobs → LLM score → `Scored` / `LowMatch` |
| `03_generate.json`| WF-C | todo   | `Scored` → CV + cover letter → `AwaitingApproval` |
| `04_send.json`    | WF-D | todo   | `Approved` → email → `Applied` |
| `05_followup.json`| WF-E | todo   | Daily → `Applied` 7d+ with no reply → `FollowUpSent` |

---

## First-time setup

### 1. Create the n8n owner account

Open `https://localhost:8443/n8n/` (or `https://localhost/n8n/` if host ports
80/443 were free on your machine) and complete the setup screen. n8n has no
account yet, so nothing can be imported until this is done.

### 2. Create the two credentials

Both are the generic **Header Auth** type. The workflow references them by
name, so use these names exactly and the import will bind them automatically.

| Credential name                 | Header name      | Value                        |
|---------------------------------|------------------|------------------------------|
| `JobHunter backend (n8n key)`   | `X-API-Key`      | `N8N_API_KEY` from `.env`    |
| `RapidAPI (x-rapidapi-key)`     | `x-rapidapi-key` | `RAPIDAPI_KEY` from `.env`   |

Use the **n8n key**, not the app token. The backend rejects the app token on
`/internal/*` with a 403 — that separation is the point of having two.

### 3. Import the workflow

Workflows → ⋮ → **Import from File** → `n8n/workflows/01_ingest.json`.

The three `REPLACE_ME_*` credential ids in the JSON are placeholders; n8n
resolves them by name once the credentials above exist. If a node shows a red
credential warning, open it and pick the credential from the dropdown.

### 4. Test it before enabling the schedule

Hit **Execute Workflow** — the manual trigger runs the identical path the cron
will. Then check the result in the app's data:

```bash
curl -k -H "Authorization: Bearer $API_AUTH_TOKEN" \
  https://localhost:8443/api/fetch-logs?limit=1
```

A successful run writes one `fetch_log` row per search title, with
`returned` / `inserted` / `skipped` counts. Only activate the workflow once a
manual run looks right.

---

## How WF-A is put together

```
Twice daily ─┐
             ├─▶ Load profile ─▶ Build searches ─▶ Loop over searches ─┐
Run now ─────┘                                            ▲            │
                                                          │            ▼
                                                          │       Fetch jobs ──(error)──┐
                                                          │            │                │
                                                          │            ▼                ▼
                                                          └───── Wrap batch ─▶ Ingest ─▶ Record failure
                                                                                  │(error) │
                                                                                  └────────┘
```

**Search terms are not in the workflow.** `Load profile` reads `search_titles`
and `max_jobs_per_run` from the database. Changing what you search for is an
edit in the app, not a workflow change — and `max_jobs_per_run` stays the one
place the rate-limit cap is defined.

**Backend URLs have no `/api` prefix.** That prefix belongs to Caddy, which
strips it before forwarding. These calls go direct over the internal Docker
network to `http://backend:8080`, so they must use the unprefixed path.

**One search at a time.** The loop's batch size of 1 serialises calls to a
rate-limited endpoint, and keeps each search's results paired with its own
title — a parallel fan-in would merge everything into one list and the
`fetch_log` rows would lose their meaning.

**Errors don't abandon the run.** `Fetch jobs` and `Ingest` route failures to
`Record failure` (which writes to the backend's `errors` table, surfacing in
`GET /stats` and the app) and then continue the loop. One failing search should
not cost you the other searches' results. `Record failure` itself is set to
continue on error, so a logging failure can never become the thing that breaks
the run.

**Zero results is a normal outcome, not a failure.** A search that matched
nothing still consumed an API call, so it still flows through and still writes
a `fetch_log` row. Likewise, a run reporting `inserted: 0, skipped: 40` means
the dedup guards did their job — do **not** add a retry on a low insert count.

---

## Editing workflows

Edit in the UI, then re-export over the file here and commit:

Workflow → ⋮ → **Download** → replace `n8n/workflows/01_ingest.json`.

Exports include credential *ids* specific to your instance. Before committing,
either replace them with the `REPLACE_ME_*` placeholders or accept that they're
local-only ids — the credential **names** are what make an import portable. The
secrets themselves are never in the export.
