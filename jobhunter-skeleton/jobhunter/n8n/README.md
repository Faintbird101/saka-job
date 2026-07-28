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

Open **`https://n8n.sakajob.home:7443`** (or the fallback
`https://localhost:7443/n8n/` if you haven't added the hosts entries yet — see
`infra/README.md`). Accept the self-signed certificate warning.

n8n has no account at all yet, so its first screen is a setup form asking for
an email, a name, and a password. These are **local to your n8n instance** —
they are not an Anthropic, GitHub, or RapidAPI login, and nothing is sent
anywhere. Fill them in and continue. Until this is done, nothing can be
imported and the API is locked.

### 2. Give n8n access to the backend and to RapidAPI

This is the part that answers "how do I give n8n access?" — n8n reaches both
services over plain HTTP headers, and you store those headers once as
**credentials** so no secret is ever written into a workflow file.

In the n8n UI: **Credentials → Add credential → search "Header Auth" → Header
Auth**. Create it twice:

| Credential name (use exactly)   | Header Name      | Header Value                 |
|---------------------------------|------------------|------------------------------|
| `JobHunter backend (n8n key)`   | `X-API-Key`      | the `N8N_API_KEY` value from `.env` |
| `RapidAPI (x-rapidapi-key)`     | `x-rapidapi-key` | the `RAPIDAPI_KEY` value from `.env` |

Copy the values out of `jobhunter/.env`. The names matter: the workflow
references its credentials by name, so matching them exactly means the import
binds automatically instead of leaving you to fix three red nodes by hand.

Use the **n8n key**, not the app token. The backend answers `/internal/*` with
a 403 for the app token — that separation is the whole point of having two
credentials, and it is what stops a leaked phone token from injecting rows into
the jobs table.

Nothing else is needed to "give access": the backend authorises purely on that
header, and n8n reaches it at `http://backend:8080` over the internal Docker
network (already verified working), never through Caddy.

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
  "https://api.sakajob.home:7443/fetch-logs?limit=1"
# or, without hosts entries:
curl -k -H "Authorization: Bearer $API_AUTH_TOKEN" \
  "https://localhost:7443/api/fetch-logs?limit=1"
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
