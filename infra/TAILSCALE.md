# Reaching the API from outside the house

The stack listens on your laptop. This makes it reachable from your phone
anywhere, without opening anything to the public internet.

## What this does and does not fix

It makes the API **reachable**. It does not make it **available** — if the
laptop is asleep, off, or Docker is stopped, nothing answers. No tunnel of any
kind changes that. If that starts to hurt, the fix is running the stack
somewhere that stays on, not a different tunnel.

Some comfort: the pipeline is cron-driven, so an offline laptop means jobs pile
up rather than being lost. WF-A simply fetches more on its next run.

## Why Tailscale rather than a public tunnel

The Flutter app will reject Caddy's self-signed certificate outright. Whatever
we use has to provide a certificate a phone already trusts.

Tailscale issues a real Let's Encrypt certificate for your machine's `.ts.net`
name, *and* keeps everything on a private network only your own devices can
join. A public tunnel would need a domain to get a stable hostname, and would
put the API on the open internet for no benefit here.

---

# Step by step

## 1. Install on the laptop

Download from <https://tailscale.com/download/windows> and run the installer.
Sign in with whichever account you prefer (Google, Microsoft, GitHub, email) —
just remember which, because the phone must use the SAME one.

Check it worked, from any terminal:

```
tailscale status
```

If `tailscale` is not on your PATH, it lives at
`C:\Program Files\Tailscale\tailscale.exe`.

The first line of the output is this machine: note its **name** and its
**100.x.y.z** address.

## 2. Install on the phone

Tailscale from the App Store or Play Store. **Sign in with the same account.**
Both devices should now appear in `tailscale status` and in the admin console.

At this point the phone can already reach the laptop by its `100.x` address —
but only over plain HTTP or with the self-signed certificate, which is not good
enough for the app. Steps 3 and 4 fix that.

## 3. Turn on MagicDNS and HTTPS certificates

Both are off by default and both are required.

1. Open <https://login.tailscale.com/admin/dns>
2. Under **Nameservers**, enable **MagicDNS**
3. Scroll to **HTTPS Certificates** and click **Enable HTTPS**

Enabling HTTPS assigns your tailnet a name like `tail1a2b3c.ts.net`, and your
machine becomes `<machine>.tail1a2b3c.ts.net`. That full name is what the app
will use.

> Note: enabling HTTPS publishes your machine names to the public Certificate
> Transparency log. The names become public; nothing else does, and nothing
> becomes reachable. If that bothers you, say so and we can look at pinning a
> certificate instead.

## 4. Point Tailscale at the API

The compose stack already publishes a plain-HTTP port for this, on loopback
only. Run, on the laptop:

```
tailscale serve --bg --https=443 http://127.0.0.1:7081
```

Then confirm:

```
tailscale serve status
```

You should see `https://<machine>.<tailnet>.ts.net` mapped to
`http://127.0.0.1:7081`.

`--bg` keeps it running in the background and it persists across reboots. To
undo everything later: `tailscale serve reset`.

## 5. Test it

From the **laptop**:

```
curl https://<machine>.<tailnet>.ts.net/health
```

Note: no `-k`. If that works without a certificate warning, the app will accept
it too — that is the whole point of this exercise.

From the **phone**, with Tailscale connected, open the same URL in a browser.
You should get `{"status":"ok","database":"ok",...}`.

Then try it on mobile data with wifi off. That is the real test.

## 6. Log in from the phone

```
POST https://<machine>.<tailnet>.ts.net/auth/login
{"email":"...","password":"..."}
```

Returns a session token for the app to store. That token is revocable —
`POST /auth/logout-all` kills every session, which is what to do if the phone
goes missing.

---

## Optional: n8n as well

Port 7082 is already wired for the editor. Put it on a second HTTPS port
because n8n does not survive being served under a path prefix:

```
tailscale serve --bg --https=8443 http://127.0.0.1:7082
```

Reachable at `https://<machine>.<tailnet>.ts.net:8443`.

---

## Troubleshooting

**`curl` says connection refused on 7081** — the stack is down, or Caddy did
not pick up the config. `docker compose up -d caddy`.

**404 from a `.ts.net` URL but 200 from `127.0.0.1:7081`** — the Caddy site
address has been given a hostname. It must stay a bare `:7081`, because Caddy
matches on the Host header and tailscaled forwards the `.ts.net` name.

**Certificate errors on the phone** — HTTPS Certificates is not enabled in the
admin console (step 3), or you are using the `100.x` address rather than the
`.ts.net` name. The certificate is issued for the name, not the address.

**Works on wifi, not on mobile data** — check the phone's Tailscale app is
actually connected; some battery optimisers suspend the VPN. On Android,
exclude Tailscale from battery optimisation.

**Everything stops when the laptop sleeps** — expected, see the top. Windows
power settings, or move the stack to a host that stays on.
