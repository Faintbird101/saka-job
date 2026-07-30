# Extra CA certificates

`extra-ca.crt` is mounted into n8n and trusted via `NODE_EXTRA_CA_CERTS`.

## Why this exists

TLS-inspecting antivirus (Avast Web/Mail Shield, Kaspersky, ESET, Bitdefender
and others) works by intercepting outbound TLS, terminating it locally, and
re-signing the traffic with its own root certificate. That root is installed in
the *Windows* trust store — but a container has its own, so Node inside n8n
sees a chain it cannot verify and the SMTP credential fails with:

    self-signed certificate in certificate chain

Gmail's real certificate is of course not self-signed. Confirmed by asking the
container directly what it was served:

    docker exec infra-n8n-1 openssl s_client -connect smtp.gmail.com:465 \
      -servername smtp.gmail.com </dev/null 2>/dev/null | grep issuer

## The fix, and why this one

Trusting that single extra root. The alternative — `NODE_TLS_REJECT_UNAUTHORIZED=0`
— disables certificate verification for *every* connection n8n makes, including
the ones that carry your API keys. Trading all TLS verification for one mail
notification is a bad deal.

This file is machine-specific and **gitignored**: every antivirus install
generates its own root, so a committed copy would be useless elsewhere and
misleading here.

## Regenerating it

```bash
docker exec infra-n8n-1 sh -c \
  'openssl s_client -showcerts -connect smtp.gmail.com:465 \
   -servername smtp.gmail.com </dev/null 2>/dev/null' > chain.txt
# take the LAST certificate block (the self-signed root) into extra-ca.crt
```

Then `docker compose up -d n8n`.

## Or remove the interception instead

In Avast: **Menu → Settings → Protection → Core Shields → Mail Shield**, and
turn off SSL/TLS scanning. Then this file is unnecessary. That is the cleaner
outcome if you are willing to give up mail scanning.
