# Phase M — Ship it

From master `a2bb3b0`. Code branch: `phase-m-ship`. Runs before Phase L
(that name stays with FULL-APP-PLAN.md's stretch list).

**Goal:** V1 is built but runs nowhere and nothing checks it. Make it
safe to put online, then deploy it.

## Parts (one file each, in this order)

| # | File | What |
|---|---|---|
| 1 | phase-m-ci.md | GitHub Actions, a Playwright suite, tests for untested code |
| 2 | phase-m-security.md | Refuse the default JWT secret, rate limits, `Secure` cookies |
| 3 | phase-m-ledger.md | Reserve cash for queued orders, claim before fill, return write errors |
| 4 | phase-m-deploy.md | Compose + Caddy, nightly backups, DEPLOY.md |

CI goes first so every later part is checked. Last: a RESUME.md row, and
in CLAUDE.md the CI note plus fixing the stale "design only, not yet
implemented" heading.

## Out of scope

- A licensed market-data feed (open question 1).
- WebSocket prices: wait for a real-time feed.
- Running more than one backend, beyond making the matcher safe.
- Phase L features; the `quant-gemini-bridge` branch.

## Open questions (none block the phase)

1. **Data:** stock prices come from Vietcap's public web API, not a
   licensed feed. Keep it for a private or educational demo, or pay for a
   licence before a public launch?
2. **Host and domain** for phase-m-deploy.md.
3. **`quant-gemini-bridge`** (unmerged, 2026-09-26): merge or drop?
4. **Cleanup:** delete `.cursorrules` and the seven merged branches?

## Done when

CI is green on the phase's PR with the Postgres tests running, and every
part file's "Verify" list passes.
