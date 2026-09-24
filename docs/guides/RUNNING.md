# Running VN Stock Sim locally (without Docker)

For a one-command demo instead, see docs/guides/DOCKER.md (`./run.sh`).
This doc is for day-to-day development: running the backend and frontend
directly on your machine, with hot reload.

## Prerequisites

- Go 1.22 or newer (`go version`)
- Node.js 20 or newer and npm (`node --version`, `npm --version`)

The backend has only been built and run inside Docker so far (see
RESUME.md) -- this machine has no local Go install. The commands below are
the standard `go run`/`go build` workflow and should work as written on a
machine with Go installed, but they have not been run outside a container
yet. If something does not match, `go.mod`/`go.sum` in `backend/` are the
source of truth; `go mod tidy` will fix a stale `go.sum`.

## 1. Start the backend

```bash
cd backend
go run ./cmd/api
```

Serves the API on `http://localhost:8080`. Everything is in-memory (see
RESUME.md) -- state resets every time you restart this process.

Environment variables (both optional, with defaults from
`internal/config/config.go`):

| Variable     | Default                 | Purpose                              |
| ------------ | ------------------------ | ------------------------------------- |
| `PORT`       | `8080`                   | HTTP port                             |
| `JWT_SECRET` | `dev-secret-change-me`   | HMAC signing key for bearer tokens    |

Verify it is up:

```bash
curl http://localhost:8080/api/v1/symbols
```

You should see the 5 mock HOSE/HNX tickers (VNM, VCB, HPG, FPT, SHB).

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
