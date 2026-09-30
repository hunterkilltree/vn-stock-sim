# Phase M · 2. Security

Part of phase-m.md.

**Why:**
- `config.Load` silently falls back to `JWT_SECRET=dev-secret-change-me`
  (see phase-persistence.md "Still open").
- The backend has no rate limiter anywhere. Quant only passes on the
  provider's own rate-limit errors.
- `secure: false` is hard-coded in `authActions.ts`, `accountActions.ts`
  and `cryptoActions.ts`.

## Decisions

1. **No default secret on a real database.** With `DATABASE_URL` set and
   the default `JWT_SECRET`, the backend logs why and exits. In-memory
   runs still accept the default. `./run.sh` writes a random secret to a
   git-ignored `.env` on first run, and compose reads it.
2. **Rate limits on login and register**, answering 429
   `{code: "rate_limited"}`:
   - **Per client IP**, about 10 a minute. Server Actions mean the
     backend only sees the Next.js server's IP. So the frontend forwards
     the browser's IP in `X-Forwarded-For`, which the backend trusts only
     from `TRUSTED_PROXIES` (default: loopback and the compose network).
   - **Per email:** 5 failed logins in 15 minutes lock that email for the
     rest of the window.
   - A small in-process limiter in `internal/middleware`. That's enough
     for one backend.
3. **`Secure` cookies.** One `cookieOptions()` helper in `session.ts`
   replaces the three `secure: false`. It reads `COOKIE_SECURE`, which
   defaults to false so local HTTP still works.

## Verify

- Through the frontend, the 11th login from one IP in a minute gets a
  429, while another IP still works.
- An untrusted `X-Forwarded-For` is ignored.
- 5 bad passwords lock that email.
- The backend exits with `DATABASE_URL` set and no secret, and starts in
  memory without one.
- Cookies are `Secure` only with `COOKIE_SECURE=true`.
