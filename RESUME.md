# RESUME — VN Stock Sim implementation status

This file tracks what's actually been built, what's planned next, and what's
deferred, so a future session (human or Claude Code) can pick this up without
re-deriving context. See [CLAUDE.md](CLAUDE.md) for the repo orientation and
the design docs (`vn-stock-sim-summary.md`, `vn-stock-sim-version-highlights.md`,
`api-spec.md`, `charting-library-integration.md`) for the full spec.

Last updated: 2026-09-21.

---

## Done

**Backend (`backend/`) — Go + Gin, V1 MVP feature set, scaffolded and code-complete.**

Follows the architecture in `api-spec.md`/`CLAUDE.md`: Handler → Service → Adapter
per feature, feature-based package layout under `internal/`, everything under
`/api/v1`, response envelope `{ data, meta }` / `{ code, message }` per spec.

Implemented, with in-memory stores/mock adapters (no database yet):

- `internal/auth` — register/login/me, bcrypt password hashing, hand-rolled
  HMAC-SHA256 bearer tokens (`internal/authtoken`) instead of a third-party JWT
  lib, to keep the dependency surface small.
- `internal/symbol` — `MockProvider` seeded with 5 real HOSE/HNX tickers
  (VNM, VCB, HPG, FPT, SHB) with plausible fundamentals; search + detail.
- `internal/market` — deterministic synthetic OHLCV bar generator (seeded by
  symbol+time, not `math/rand`, so repeated calls are reproducible) behind a
  `MarketDataProvider` port; SMA/EMA indicators implemented, others return
  empty for now.
- `internal/watchlist`, `internal/portfolio`, `internal/order` — full paper
  trading loop: an account opens with 100,000,000 VND virtual cash
  (`portfolio.StartingCash`), market orders fill immediately against the mock
  quote and update the portfolio ledger (`portfolio.MemoryStore.ApplyFill`),
  limit orders are accepted but stored as `"queued"` (no real matching yet).
- `internal/backtest` — EMA-crossover rule (`internal/backtest/rule.go`), run
  synchronously on `POST /backtests` and stored already `"completed"` (the
  spec's "queue → poll" contract is preserved in the response shape, but
  there's no real worker pool behind it yet).
- `cmd/api/main.go` wires all of the above into Gin route groups.

**⚠️ Not yet verified to compile or run.** Go is not installed on this
machine (checked: `go version` → command not found). The code was written
carefully against Go's stdlib + the two declared dependencies
(`github.com/gin-gonic/gin`, `golang.org/x/crypto/bcrypt`), and reviewed by
eye for import/interface consistency, but nobody has run `go build` on it.
**First thing to do in a Go-capable environment:**

```bash
cd backend
go mod tidy   # fetches gin + x/crypto, generates go.sum (currently missing)
go build ./...
go run ./cmd/api   # serves on :8080
```

Expect to fix a handful of small compile errors — treat this repo as an
unverified first draft of the backend, not a tested one.

**Frontend (`frontend/`) — Next.js, scaffolding in progress via
`create-next-app` (TypeScript, Tailwind, App Router, `src/` dir) — check
whether that command finished; if `frontend/` is empty or partial, rerun:**

```bash
npx create-next-app@latest frontend --typescript --tailwind --eslint --app --src-dir --import-alias "@/*" --use-npm
```

No application pages/components have been built yet beyond whatever
`create-next-app`'s template ships.

**Repo/tooling:**

- `CLAUDE.md` and `.claude/settings.json` (plugin config) merged via PR #1.
- Branch for this work: `v1-mvp-scaffold` (off `master`), not yet pushed/PR'd
  as of this writing.

---

## Plan (where to pick up)

Roughly in priority order for reaching a demoable V1 MVP
(`vn-stock-sim-version-highlights.md` §Version 1):

1. **Get the backend compiling.** Install Go, run `go mod tidy`, fix
   whatever `go build ./...` surfaces. This is the immediate blocker on
   everything else backend-related.
2. **Confirm/finish the frontend scaffold**, then build the V1 pages against
   the backend's mock data:
   - Stock browser / search (`GET /symbols`)
   - Stock detail page (`GET /symbols/:symbol`) with a candlestick chart —
     start with a plain chart lib (e.g. lightweight-charts) reading
     `GET /market/bars`; defer the full TradingView Charting Library
     integration (`charting-library-integration.md`) until access is
     requested/granted (see Future work below) — a simpler chart is enough
     to prove the V1 loop end-to-end.
   - Auth pages (register/login), storing the bearer token and attaching it
     to authenticated requests.
   - Watchlist table, portfolio summary/positions, paper trade order form,
     trade history, and a basic backtest form+result view.
3. **Wire a real Postgres database** behind `auth`, `watchlist`, `portfolio`,
   and `order` (currently all in-memory `MemoryStore`s that reset on
   restart). Each store already sits behind a small interface-shaped API
   (not literally a Go interface everywhere yet, but close) — swapping the
   backing store is the intended seam, not a rewrite.
4. **Replace the market-data mock** with the real thing: sign up for a
   licensed Vietnamese market-data provider (`api-spec.md` explicitly rules
   out scraped data), write an adapter implementing `symbol.Provider` and
   `market.MarketDataProvider`, same pattern the mock already follows.
5. **Docker/dev-compose** for backend + Postgres + frontend, so `docker
   compose up` is enough to run V1 locally — not started yet.

---

## Future work (explicitly deferred, not started)

- **Indicators beyond SMA/EMA**: RSI, MACD, Bollinger, VWAP
  (`vn-stock-sim-summary.md` lists all of these for V1's chart).
- **Limit order matching** — currently orders of type `limit` are accepted
  and stored as `"queued"` forever; nothing ever fills them.
- **Real backtest worker pool** — `POST /backtests` runs synchronously
  today; the response shape already matches the spec's async "queue → poll"
  contract, so adding a real queue later shouldn't require an API change.
- **TradingView Charting Library integration** per
  `charting-library-integration.md` — needs the (free but access-gated)
  library files from TradingView's GitHub approval process before
  `TVDatafeedAdapter` can be built; kick off that request early since
  approval isn't instant.
- **Real-time WebSocket price feed** (`ws.Hub`) — nothing here yet; V1's
  `GET /market/bars` is pull-only.
- **Everything in Version 2+** (screener, heatmap, Strategy Builder, Replay
  Mode, Trade Journal, alerts, and later the Quant AI assistant, order flow,
  broker integration, etc.) — intentionally out of scope until V1 actually
  works end-to-end, per the phased build sequence in
  `vn-stock-sim-version-highlights.md`.
- **`.cursorrules`' Java/Spring Boot stack** is not being followed — see
  CLAUDE.md's note on why `api-spec.md`'s Go+Gin stack takes precedence.
