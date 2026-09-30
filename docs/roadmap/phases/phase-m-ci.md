# Phase M · 1. CI and tests

Part of phase-m.md.

**Why:** there's no `.github/`, so every check so far was run by hand.
The Phase I–K browser scripts lived in a scratch folder and are gone.
`market` (907 lines), `authtoken`, `middleware` and `symbol` have no
tests.

## Decisions

1. **`.github/workflows/ci.yml`**, on every push and PR:
   - **backend:** Go 1.24 (the Dockerfile's version); `gofmt -l`,
     `go vet`, and `go test ./...` with a `postgres:16` service and
     `TEST_DATABASE_URL` set, so the Postgres contract tests run.
   - **frontend:** Node 20 (the Dockerfile's version); `npm ci`, `tsc`,
     `eslint`, `npm run build`.
   - **e2e:** backend in memory with `MARKET_DATA_SOURCE=mock` and
     `ORDER_MATCH_INTERVAL=2s`, the production frontend, then Playwright.
   - Also `docker compose config`. Never `go mod tidy`
     (phase-persistence.md).
2. **A Playwright suite** in `frontend/e2e/`. It fails on any page error
   and covers:
   - register → market buy → queue a limit → cancel it;
   - a stop order filled by the matcher;
   - watchlist add/remove on Detail, and the list on Main;
   - a `/backtest` run;
   - Replay: advance, buy, end, score;
   - a fractional crypto buy;
   - every page at 375 px with no horizontal overflow.
3. **Unit tests**, written before parts 2–3 change the code:
   - `authtoken`: round trip, expiry, wrong secret, and a token that
     isn't HS256 (e.g. `alg: none`) is rejected.
   - `middleware`: a missing or bad header gets a 401; public routes pass.
   - `market`:
     - the mock returns the same numbers for the same inputs
       (phase-0-mvp.md);
     - VCI falls back per call when an `httptest` server fails;
     - the circuit breaker opens and closes;
     - the Hanoi session helpers (the phase-c.md bug).
   - `symbol`: search; ceiling and floor rounded to the tick size.

## Verify

CI is green on the PR, and the Postgres subtests ran instead of being
skipped.
