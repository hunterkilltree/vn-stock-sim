# Running VN Stock Sim locally (without Docker)

For a one-command demo instead, see docs/guides/DOCKER.md (`./run.sh`).
This doc is for day-to-day development: running the backend and frontend
directly on your machine, with hot reload.

## Prerequisites

- Go 1.24 or newer (`go version`) -- required by the Anthropic Go SDK used by Trợ lý Quant
- Node.js 20 or newer and npm (`node --version`, `npm --version`)
- Optional: PostgreSQL 16 (or any recent version), if you want data to
  survive a backend restart -- see "Keeping data: Postgres" below.

`go.mod`/`go.sum` in `backend/` are the source of truth for Go
dependencies. Avoid a bare `go mod tidy` on Go 1.25+: it can raise the
`go` line above 1.24, which the Dockerfile's `golang:1.24` image can't
build.

## 1. Start the backend

```bash
cd backend
go run ./cmd/api
```

Serves the API on `http://localhost:8080`. Without `DATABASE_URL`,
everything is in memory and resets every time you restart this process;
with it, accounts, portfolios, orders, watchlists, backtests and Replay
sessions are stored in Postgres (phase-persistence.md).

Environment variables (all optional, with defaults from
`internal/config/config.go`):

| Variable                        | Default                | Purpose |
| ------------------------------- | ---------------------- | ------- |
| `PORT`                          | `8080`                 | HTTP port |
| `JWT_SECRET`                    | `dev-secret-change-me` | HMAC signing key for bearer tokens |
| `MARKET_DATA_SOURCE`            | `vci`                  | `vci` (live: VCI for stocks, Binance's public API for crypto; each falls back to the mock per call) or `mock` |
| `DATABASE_URL`                  | unset (in-memory)      | Postgres connection URL, e.g. `postgres://vss:pw@localhost:5432/vss?sslmode=disable`. The schema is created and migrated on startup. |
| `ORDER_MATCH_INTERVAL`          | `20s`                  | How often queued limit/stop/ATC/OCO orders are checked against new 5-minute bars (Go duration, e.g. `5s`) |
| `QUANT_ALLOW_PRIVATE_ENDPOINTS` | unset (off)            | `true` lets Trợ lý Quant's "Máy chủ riêng" provider reach localhost/private addresses (e.g. Ollama on the same machine). Leave off on any shared server. |

Verify it is up:

```bash
curl http://localhost:8080/api/v1/symbols
```

You should see the app's 40 HOSE/HNX/UPCOM tickers.
`curl http://localhost:8080/healthz` also reports storage:
`{"status":"ok","db":"ok"}` with Postgres, `"db":"off"` in memory.

### Keeping data: Postgres

```bash
createdb vss   # any empty database works
DATABASE_URL="postgres://$USER@localhost:5432/vss?sslmode=disable" go run ./cmd/api
```

The log says `storage: postgres (schema migrated)`. Migrations live in
`backend/internal/db/migrations/` and are applied once each, in order, and
recorded in `schema_migrations`. To start over, drop and recreate the
database.

The store tests run against both the in-memory stores and Postgres when
you point them at a scratch database (each package uses its own
`test_<package>` schema, dropped and recreated per run):

```bash
TEST_DATABASE_URL="postgres://$USER@localhost:5432/vss_test?sslmode=disable" go test ./...
```

Without `TEST_DATABASE_URL`, the Postgres half is skipped.

## 2. Start the frontend

In a second terminal:

```bash
cd frontend
npm install
npm run dev
```

Serves the app on `http://localhost:3000`. By default it talks to the
backend at `http://localhost:8080` (see `src/lib/api.ts`'s fallback) --
that only needs overriding if your backend runs somewhere else:

```bash
cp .env.example .env.local
# edit .env.local if NEXT_PUBLIC_API_BASE_URL needs to point elsewhere
```

## 3. Look at it

- `http://localhost:3000/stocks` -- stock browser, reads live from the
  backend's `GET /api/v1/symbols`.
- `http://localhost:3000/stocks/VNM` -- stock detail page (or any of VCB,
  HPG, FPT, SHB).
- `http://localhost:3000/chart` -- TradingView public widget proof of
  concept (BTC/USD, not wired to our own data yet -- see RESUME.md).

If `/stocks` shows a red "could not reach the API" message, the backend
is not running or is on a different port than the frontend expects.

## Running both from one terminal (optional)

There is no root-level script for this yet. Simplest options: two
terminal tabs as above, or run the backend in the background:

```bash
(cd backend && go run ./cmd/api &)
cd frontend && npm run dev
```
