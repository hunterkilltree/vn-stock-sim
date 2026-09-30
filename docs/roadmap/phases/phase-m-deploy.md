# Phase M · 4. Deploy

Part of phase-m.md.

**Why:** the app runs nowhere. The target is one VM running Docker
Compose plus Caddy:
- it reuses the compose file that already exists;
- it works on any host;
- Caddy sets up HTTPS automatically.

## Decisions

1. **`deploy/docker-compose.prod.yml`**, an override that:
   - adds `caddy`;
   - sets `COOKIE_SECURE=true` and `TRUSTED_PROXIES`;
   - stops publishing the database port;
   - adds `restart: unless-stopped`.
2. **`deploy/Caddyfile`**, with the domain read from an env var.
3. **A nightly `pg_dump`** into a volume, keeping 7 days.
4. **`docs/guides/DEPLOY.md`**:
   - the steps and env vars;
   - how to restore a backup;
   - Render/Fly plus hosted Postgres as an alternative, documented but
     not built.

The host, domain and account are yours (phase-m.md, open question 2).

## Verify

- The override passes `docker compose config`.
- Where Docker is available, run the whole stack behind Caddy's internal
  certificate authority on `localhost`, then write a backup and restore
  it.
- Newman still passes.
