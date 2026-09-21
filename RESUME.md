# RESUME — VN Stock Sim implementation status

This file tracks what has actually been built, what is planned next, and
what is deferred, so a future session (human or Claude Code) can pick this
up without re-deriving context. See CLAUDE.md for the repo orientation and
the design docs (vn-stock-sim-summary.md, vn-stock-sim-version-highlights.md,
api-spec.md, charting-library-integration.md) for the full spec. See
DOCKER.md for how to run the whole thing as a demo.

Last updated: 2026-09-21.

---

## Done

Backend (backend/) — Go + Gin, V1 MVP feature set, scaffolded and
CONFIRMED WORKING via Docker.

Follows the architecture in api-spec.md/CLAUDE.md: Handler to Service to
Adapter per feature, feature-based package layout under internal/,
everything under /api/v1, response envelope { data, meta } / { code,
message } per spec.

Implemented, with in-memory stores/mock adapters (no database yet):

- internal/auth — register/login/me, bcrypt password hashing, hand-rolled
  HMAC-SHA256 bearer tokens (internal/authtoken) instead of a third-party
  JWT lib, to keep the dependency surface small.
- internal/symbol — MockProvider seeded with 5 real HOSE/HNX tickers
  (VNM, VCB, HPG, FPT, SHB) with plausible fundamentals; search + detail.
- internal/market — deterministic synthetic OHLCV bar generator (seeded by
  symbol+time, not math/rand, so repeated calls are reproducible) behind a
  MarketDataProvider port; SMA/EMA indicators implemented, others return
  empty for now.
- internal/watchlist, internal/portfolio, internal/order — full paper
  trading loop: an account opens with 100,000,000 VND virtual cash
  (portfolio.StartingCash), market orders fill immediately against the
  mock quote and update the portfolio ledger
  (portfolio.MemoryStore.ApplyFill), limit orders are accepted but stored
  as queued (no real matching yet).
- internal/backtest — EMA-crossover rule (internal/backtest/rule.go), run
  synchronously on POST /backtests and stored already completed (the
  spec async queue-then-poll contract is preserved in the response
  shape, but there is no real worker pool behind it yet).
- cmd/api/main.go wires all of the above into Gin route groups.

Verified 2026-09-21: this machine still has no local Go install, but
Docker Desktop is available, so the backend was actually built and run
inside golang:1.22-alpine / alpine:3.20 containers (see
backend/Dockerfile and docker-compose.yml) — this was the first real
compile of this code, not just eye review. go.mod/go.sum were generated
by running go mod tidy inside a golang:1.22-alpine container against the
real files on disk (docker run with a bind mount), so both are now
committed and complete. docker compose up was run and
curl localhost:8080/api/v1/symbols and
curl localhost:8080/api/v1/symbols/VNM both returned correct JSON from
the mock fixtures.

Frontend (frontend/) — Next.js 16 (App Router, TypeScript, Tailwind,
src/ dir), scaffolded via create-next-app. Verified both standalone
(npx tsc --noEmit, npx eslint ., next build all clean) and inside Docker:
built with a standalone-output multi-stage Dockerfile
(frontend/Dockerfile, next.config.ts output: standalone), and with the
backend also running in Compose, curl localhost:3000/stocks rendered
real symbol data server-side (all 5 mock tickers present in the HTML, no
error fallback), and /stocks/VNM rendered the detail page correctly too.

Built beyond the template:

- src/lib/api.ts — typed client for the Go backend
  (NEXT_PUBLIC_API_BASE_URL, default http://localhost:8080 outside
  Docker, baked to http://backend:8080 at image build time inside
  Compose — see frontend/.env.example and docker-compose.yml).
- /stocks and /stocks/[symbol] — Server Components covering
  GET /api/v1/symbols and GET /api/v1/symbols/:symbol. The list page
  shows a visible error message instead of crashing when the backend is
  not running.
- Home page (/) links into /stocks.

Note: frontend/AGENTS.md (written by next dev, not by this session) warns
that Next.js 16 has breaking API/convention changes vs older training
data — it was consulted (node_modules/next/dist/docs/) before writing the
pages above, e.g. to confirm params is still a Promise in dynamic routes.

Not built yet: candlestick chart, indicators on the chart, auth pages,
watchlist/portfolio/order/backtest UI.

Docker (docker-compose.yml, backend/Dockerfile, frontend/Dockerfile,
DOCKER.md) — DONE, see above. docker compose up --build runs the full V1
demo (frontend on :3000, backend on :8080). No database service yet
(everything is in-memory, resets on restart).

Repo/tooling:

- CLAUDE.md and .claude/settings.json (plugin config) merged via PR #1.
- Backend + frontend scaffold merged via PR #2.
- Branch for this work: docker-compose-demo (off master).
- gh auth login was completed partway through this session (an earlier
  device code expired and was retried successfully by the user).

---

## Plan (where to pick up)

Roughly in priority order for reaching a demoable V1 MVP
(vn-stock-sim-version-highlights.md, Version 1 section):

1. Build out the remaining V1 frontend pages against the backend (the
   stock browser slice above is the first one done):
   - Candlestick chart on the stock detail page — start with a plain
     chart lib (e.g. lightweight-charts) reading GET /market/bars; defer
     the full TradingView Charting Library integration
     (charting-library-integration.md) until access is requested/granted
     (see Future work below) — a simpler chart is enough to prove the V1
     loop end-to-end.
   - Auth pages (register/login), storing the bearer token and attaching
     it to authenticated requests.
   - Watchlist table, portfolio summary/positions, paper trade order
     form, trade history, and a basic backtest form+result view.
2. Wire a real Postgres database behind auth, watchlist, portfolio, and
   order (currently all in-memory MemoryStores that reset on restart).
   Each store already sits behind a small interface-shaped API (not
   literally a Go interface everywhere yet, but close) — swapping the
   backing store is the intended seam, not a rewrite. Add it as a db
   service in docker-compose.yml alongside backend/frontend.
3. Replace the market-data mock with the real thing: sign up for a
   licensed Vietnamese market-data provider (api-spec.md explicitly rules
   out scraped data), write an adapter implementing symbol.Provider and
   market.MarketDataProvider, same pattern the mock already follows.

---

## Future work (explicitly deferred, not started)

- Indicators beyond SMA/EMA: RSI, MACD, Bollinger, VWAP
  (vn-stock-sim-summary.md lists all of these for V1 chart).
- Limit order matching — currently orders of type limit are accepted and
  stored as queued forever; nothing ever fills them.
- Real backtest worker pool — POST /backtests runs synchronously today;
  the response shape already matches the spec async queue-then-poll
  contract, so adding a real queue later should not require an API
  change.
- TradingView Charting Library integration per
  charting-library-integration.md — needs the (free but access-gated)
  library files from TradingView GitHub approval process before
  TVDatafeedAdapter can be built; kick off that request early since
  approval is not instant.
- Real-time WebSocket price feed (ws.Hub) — nothing here yet; V1
  GET /market/bars is pull-only.
- Everything in Version 2+ (screener, heatmap, Strategy Builder, Replay
  Mode, Trade Journal, alerts, and later the Quant AI assistant, order
  flow, broker integration, etc.) — intentionally out of scope until V1
  actually works end-to-end, per the phased build sequence in
  vn-stock-sim-version-highlights.md.
- .cursorrules Java/Spring Boot stack is not being followed — see
  CLAUDE.md note on why api-spec.md Go+Gin stack takes precedence.
