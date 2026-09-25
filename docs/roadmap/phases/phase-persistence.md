# Phase: Persistence (Postgres)

Branch `phase-persistence`, from master `13a78db` (Phases A–K merged).
RESUME.md's plan put this first after Phase K. Every user-owned record
lives in a `MemoryStore` that is lost on restart:
- accounts;
- portfolios, positions and the equity history;
- orders, including queued ones the matcher would fill;
- the watchlist;
- backtests;
- Replay sessions.

That also rules out every free host except an always-on VM. This is an
unlettered phase, like `phase-vci-market-data.md`; the stretch list stays
"Phase L".

## Decisions

1. **Postgres via pgx v5** (`github.com/jackc/pgx/v5/pgxpool`), no ORM.
   api-spec.md's stack is Postgres; pgx is the standard pure-Go driver.
2. **Opt-in by `DATABASE_URL`.** Set, the backend opens a pool, runs the
   migrations and uses Postgres stores. Unset, it keeps today's
   in-memory stores unchanged: `go run` with no setup still works, and so
   do the unit tests and a quick demo. Startup logs which one is active.
3. **Each feature owns its store port.** `auth`, `watchlist`, `portfolio`,
   `order`, `backtest` and `replay` each define a `Store` interface
   matching today's `MemoryStore` methods, and the service depends on the
   interface. A `PGStore` next to each `MemoryStore` is the adapter.
   This is the Handler → Service → Adapter shape CLAUDE.md asks for.
   Only the wiring in `main.go` chooses.
4. **Migrations are embedded SQL** (`internal/db/migrations/*.sql`)
   applied in order inside a transaction and recorded in
   `schema_migrations`. No migration tool dependency; a new file is a
   new version.
5. **Same IDs and JSON as today.** IDs keep their shapes (`u12`, `pf_3`,
   `ord_1f`, `bt_a`, `rp_4`), generated from Postgres sequences, so the
   frontend, Postman and saved links don't change. Money and quantities
   stay `double precision`, like the in-app float64 (phase-i.md decision
   5).
6. **One schema:**
   - `users`: bcrypt hash, as now.
   - `portfolios`: including cash and an `is_default` flag with a partial
     unique index (one default per user).
   - `positions`, `equity_points`.
   - `orders`: queued ones indexed for the matcher's scan.
   - `watchlist_items`.
   - `backtests`: summary columns + a JSONB body for equity and trades.
   - `replay_sessions`: the whole session as JSONB. A session is read and
     written as one unit, and its bars are the history it was started
     with.
7. **The ledger is transactional.** `ApplyFill` locks the position and
   portfolio rows (`SELECT … FOR UPDATE`), checks cash or shares, and
   writes both in one transaction, so two fills can't overspend the same
   cash. The in-memory store got this from its mutex.
8. **Replay sessions are saved explicitly.** The service used to mutate
   the `*Session` the memory store handed out. It now calls
   `store.Save(sess)` after every change (advance, order, stop-loss fill,
   end). The memory store's Save is a no-op, so behaviour there is
   unchanged.
9. **Errors:**
   - Store methods that already return an error (register, login,
     `ApplyFill`) return database errors as they are.
   - Methods that don't (reads, appends, the watchlist) log them with the
     package name and degrade (empty list, not found), rather than
     changing every service signature in this phase.
   - `GET /healthz` reports `db: ok|down` so a host can detect an outage.
     This is recorded as a known limit.
10. **Docker**: `docker-compose.yml` gains a `postgres:16` service with a
    named volume, and the backend gets `DATABASE_URL`, so `./run.sh`
    keeps data across restarts. RUNNING.md documents running a local
    Postgres (or leaving `DATABASE_URL` unset).

## Out of scope

- Moving market data or caches into the database (they're derived and
  refetchable).
- Multi-instance deployment. The order matcher assumes one backend
  process; two would both scan queued orders. The fill itself is
  transactional, but an order could be booked twice. Noted for later
  (`FOR UPDATE SKIP LOCKED` on the scan).
- Backups, retention, and changing the JWT scheme.

## Verification plan

- **Store contract tests** run the same cases against `MemoryStore` and
  `PGStore` for every feature. The Postgres run needs
  `TEST_DATABASE_URL` and is skipped otherwise, so plain `go test ./...`
  stays database-free. Here they run against a local Postgres 16
  cluster. Cases include the concurrent-fill case from decision 7.
- **The restart test:**
  - Register, then place market and queued orders, add to the watchlist,
    run a backtest, and play a Replay session.
  - Kill the backend and start it again.
  - Everything is still there: login works with the same token, cash,
    positions, equity history, the queued order (which the matcher then
    fills), the watchlist, the backtest, and the Replay session at the
    same candle.
- **Newman** and the Phase I / J / K browser suites against the
  Postgres-backed backend.
- `go vet`, `gofmt`, `tsc`, `eslint`, `build`; `docker compose config`
  validates.

## Changed while building

- **pgx v5.7.6, not the latest v5.11.** v5.11 raises the module to Go
  1.25, which the Dockerfile's `golang:1.24` image can't build. v5.7.6
  keeps `go 1.24`. For the same reason, a bare `go mod tidy` on a Go 1.25
  toolchain must be avoided (it also bumps the `go` line). pgx was marked
  direct by hand.
- **The Postgres store tests use a schema per package**
  (`internal/db/dbtest`): `go test` runs packages in parallel, and one
  shared schema being dropped and re-migrated would collide.

## Verification (done)

Against a local PostgreSQL 16.13 cluster (the Ubuntu package already in
this environment).

- **Go tests:**
  - `go test ./...` with no database: 49 top-level tests pass, and the
    Postgres halves are simply not run.
  - With `TEST_DATABASE_URL`: the same 49 pass, plus 8 `postgres`
    subtests (auth, watchlist, backtest, order, replay, and three for
    the portfolio ledger).
  - The ledger's concurrent test starts 20 simultaneous buys of 1,000
    against 10,000 cash. Exactly 10 succeed, cash ends at 0 and the
    position at 0.1, on both stores.
- **The restart test** (`restart_test.py`, backend with
  `DATABASE_URL`):
  - Setup:
    - register;
    - a market buy, a far limit and a near stop;
    - watchlist FPT + BTCUSDT;
    - a 3-year backtest;
    - a Replay session advanced to candle 24 with a buy.
  - After killing and restarting the backend, all ten snapshots were
    identical: `/auth/me`, the portfolio, summary, positions, equity
    history length, all 4 orders, the watchlist, the backtest, the Replay
    session and the portfolio list.
  - Login works, and the pre-restart token is still valid.
  - The Replay session advanced 24 → 25.
  - The queued stop was picked up by the restarted matcher and filled at
    52.690,38 on the 02:50 bar. The far limit is still queued.
- **Newman** against the Postgres backend: 67 requests, 39 assertions, 0
  failures.
- **Browser suites** against the Postgres backend (production frontend
  build), no page errors:
  - Phase I desktop (crypto wallet, fractional order, account menu,
    crypto Replay);
  - Phase J phone flows (trade from Detail, Replay, Quant via the stub
    model, Settings, heatmap);
  - Phase K's background stop fill (227 s).
- `GET /healthz` → `{"status":"ok","db":"ok"}`; the log says
  `storage: postgres (schema migrated)`.
- **`docker compose config`** validates the new `db` service, health
  check and volume. The compose stack itself was **not run**: this
  sandbox has the Docker CLI but no daemon.

### Still open

- Store errors on read and append paths are logged, not returned
  (decision 9); `/healthz` is the signal.
- One backend process only: the matcher's scan isn't
  `FOR UPDATE SKIP LOCKED` yet (see Out of scope).
- `./run.sh` with the new `db` service needs a run on a machine with
  Docker.
- **Tokens outlive a switch of storage.** IDs restart at `u1` in every
  in-memory run, and the default `JWT_SECRET` is a fixed dev value, so a
  token issued by an earlier in-memory backend would be accepted by a
  Postgres backend as whichever user holds that ID there. This is
  harmless in local development. Any deployment must set its own
  `JWT_SECRET` (already in RESUME.md's deploy step), which invalidates
  such tokens.

