# Infra

## Topology (local dev)
Only Caddy is exposed to the host. Everything else is internal.

    host :7080/:7443  ->  Caddy  ->  api.sakajob.home  ->  backend:2623
                                 ->  n8n.sakajob.home  ->  n8n:5678
                          postgres      (internal only, no host port)

## Ports: why 7080/7443, not 80/443

Another stack on this machine publishes `0.0.0.0:80` and `0.0.0.0:443`. Docker
Desktop does **not** honour per-loopback-IP binds — asking for
`-p 127.0.0.2:443:80` still tries to allocate `0.0.0.0:443` and fails with
"port is already allocated" — so a dedicated IP is not a way around it. A
distinct port pair is the only option.

Caddy listens on the **same** numbers inside the container as it publishes
(`7080:7080`, not `7080:80`). Mapping to different internal ports would make
Caddy's redirects and its `Host` matching disagree with what the client sees.

There is no `docker-compose.override.yml` any more. It existed only to remap
80/443 onto other ports; now that the base file uses unique ports it is
redundant, and leaving it would have forced the ports back.

## Hostnames

Each service has its own name rather than a path prefix. An app hosted under a
path prefix has to rewrite every absolute URL it emits, and both n8n and beszel
are fiddly about it; on its own hostname each app is simply at `/`.

| URL                              | Service            |
|----------------------------------|--------------------|
| `https://api.sakajob.home:7443`  | backend API (at `/`) |
| `https://n8n.sakajob.home:7443`  | n8n editor         |
| `https://localhost:7443/api/…`   | API fallback, path-prefixed |
| `https://localhost:7444`         | n8n fallback |



**These names need a hosts-file entry to resolve** — already added on this
machine. To reproduce elsewhere, append to
`C:\Windows\System32\drivers\etc\hosts`:

    # ---- JobHunter (saka-job) local hostnames ----
    127.0.0.1  api.sakajob.home
    127.0.0.1  n8n.sakajob.home
    127.0.0.1  hub.sakajob.home

Note: PowerShell writes to that file hang under Windows Defender's Controlled
Folder Access, but a plain shell append works. Verify with
`ping api.sakajob.home` — it should answer from `127.0.0.1`.

Fallbacks are kept for a machine without the entries: the API at
`https://localhost:7443/api/…` and n8n at `https://localhost:7444`. You can also test hostname routing without the hosts
file at all:

    curl -k --resolve api.sakajob.home:7443:127.0.0.1 https://api.sakajob.home:7443/health

Note the two path conventions: on `api.sakajob.home` the API is at the **root**
(`/jobs`), because there is no prefix to strip. On the `localhost` fallback it
stays under `/api/` (`/api/jobs`).

## Files
- `Caddyfile`            reverse proxy + HTTPS + security headers + vhosts
- `docker-compose.yml`   full stack
- `docker/`              backend Dockerfile

## Run
1. `cp ../.env.example ../.env` and fill secrets.
2. `docker compose up -d`
3. Add the hosts entries above (once, as Administrator).
4. Backend API:  `https://api.sakajob.home:7443/health`
5. n8n UI:       `https://n8n.sakajob.home:7443`
6. Monitoring: nothing to do — see "Monitoring" below.

## HTTPS notes
- Dev uses Caddy's internal CA (`tls internal`) — real HTTPS, self-signed, so
  browsers and curl warn. Use `curl -k` locally; that is expected.
- To remove browser warnings, install Caddy's root CA into the Windows trust
  store:

      docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt .
      certutil -addstore -f "ROOT" root.crt      # run as Administrator

- Production: edit the Caddyfile — replace the `.home` names with real domains,
  drop the `:7443` suffixes, remove `tls internal` and the `local_certs` block,
  and publish 80/443. Caddy then fetches Let's Encrypt certificates itself.

## Postgres access from host
No host port is published (hardening). To run psql from the host temporarily,
add `ports: ["5432:5432"]` to the postgres service, or exec into the container:

    docker compose exec postgres psql -U jobhunter -d jobhunter

## Monitoring

**There is no beszel service in this stack.** Another project on this machine
runs a Beszel hub *and* an agent with `/var/run/docker.sock` mounted. Because
that is the same Docker daemon, JobHunter's containers already appear in that
hub with no configuration at all — a second agent would report identical stats
on a second port.

If you move this stack to a host that is *not* already monitored, add:

```yaml
  beszel-agent:
    image: henrygd/beszel-agent:latest
    network_mode: host
    restart: unless-stopped
    volumes: ["/var/run/docker.sock:/var/run/docker.sock:ro"]
    environment:
      LISTEN: "45876"
      KEY: "${BESZEL_AGENT_KEY}"
```

Then in the hub UI: **Add System** → name it, host = the agent host's address
reachable *from the hub* (`host.docker.internal` if the hub is a container on
the same machine, otherwise the LAN IP), port `45876`. Copy the public key it
shows into `.env` as `BESZEL_AGENT_KEY` and `docker compose up -d beszel-agent`.
Until the key is set the agent exits with "no key provided" — expected, not a
fault.

Beszel covers system/resource health only. Application errors go to the
`errors` table and surface via `GET /errors` and `GET /stats`.


## n8n restart durability
For workflows that wait, run n8n with persisted execution mode + the mounted
volume (already configured). Prefer state-machine polling over long waits.
