# Infra

## Topology (local dev)
Only Caddy is exposed to the host. Everything else is internal.

  host :80/:443  ->  Caddy  ->  /api/*     backend:8080
                              ->  /n8n/*     n8n:5678
                              ->  /monitor/* beszel:8090
                     postgres  (internal only, no host port)
                     beszel-agent (host network, collects metrics)

## Files
- `Caddyfile`            reverse proxy + HTTPS + security headers
- `docker-compose.yml`   full stack
- `docker/`              backend Dockerfile

## Run
1. `cp ../.env.example ../.env` and fill secrets.
2. `docker compose up -d`
3. Backend API:  https://localhost/api/...   (self-signed cert in dev — your
   browser/curl will warn; use `curl -k` locally. This is expected.)
4. n8n UI:       https://localhost/n8n/
5. Monitoring:   https://localhost/monitor/  (register the agent, then put its
   key in .env as BESZEL_AGENT_KEY and `docker compose up -d beszel-agent`)

## HTTPS notes
- Dev uses Caddy's internal CA (`tls internal`) — real HTTPS, self-signed.
- Production: edit Caddyfile — replace `localhost` with your domain, remove
  `tls internal` and the `local_certs` block. Caddy auto-fetches Let's Encrypt.

## Postgres access from host
No host port is published (hardening). To run psql from the host temporarily,
add `ports: ["5432:5432"]` to the postgres service, or exec into the container:
  docker compose exec postgres psql -U jobhunter -d jobhunter

## Beszel scope
System/resource monitoring only (CPU, RAM, disk, container stats). App-level
errors still go to the `errors` table + n8n error-trigger workflow.

## n8n restart durability
For workflows that wait, run n8n with persisted execution mode + the mounted
volume (already configured). Prefer state-machine polling over long waits.
