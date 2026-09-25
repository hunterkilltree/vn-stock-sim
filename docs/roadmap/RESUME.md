# RESUME — VN Stock Sim implementation status

This file is a short index, not a history: what's built, what's verified,
what's deferred, and where to read the detail for any one feature. See
CLAUDE.md for the repo orientation and the design docs
(vn-stock-sim-summary.md, vn-stock-sim-version-highlights.md, api-spec.md,
charting-library-integration.md) for the full spec. See RUNNING.md to run
backend + frontend locally with hot reload, or DOCKER.md for a
one-command demo.

Every row below has its own doc under `docs/roadmap/phases/` with the
real plan, decisions and verification results -- open the one you need
instead of reading this file's history. Code comments cite these docs by
bare filename (e.g. "phase-f.md decision 4"); filenames are unique.

Last updated: 2026-09-26 (Quant Gemini bridge).

## Status

Everything through Persistence is done and merged to master. All in
Vietnamese unless noted; English localization is deferred (Phase L).

| # | What | Doc |
|---|---|---|
| 0 | Initial V1 MVP scaffold: Go+Gin backend, Next.js frontend, Docker demo, auth pages, rule-based AI Insight, TradingView embed step 1 | phase-0-mvp.md |
| VCI | Live VCI market-data adapter (decorator over the mock, per-call fallback) -- why VCI and not TradingView's data API | phase-vci-market-data.md |
| A | Dark design-system tokens + two nav shells | phase-a.md |
| B | Multi-portfolio backend rework | phase-b.md |
| — | Design-system correction pass against the real design export | phase-design-alignment.md |
| C | Main/market-overview screen; VN timezone bugfix; the Postman collection | phase-c.md |
| D | Detail/ticker screen: real SVG charts, RSI/MACD | phase-d.md |
| E | Portfolio screen: equity history, allocation, KPIs/journal | phase-e.md |
| F | Replay Mode: candle-by-candle historical replay, skill score | phase-f.md |
| G | Account menu, portfolio switcher, Settings shell, Signup capital picker | phase-g.md |
| H | AI Model settings (BYOK) and Trợ lý Quant | phase-h.md |
| I | Crypto market, real stock universe, full heatmap dashboard | phase-i.md |
| J | Mobile responsive layouts | phase-j.md |
| K | Finish V1: order matching, fees, watchlist, backtest page | phase-k.md |
| — | Postgres persistence for all user data (`DATABASE_URL`) | phase-persistence.md |
| — | Reference "Máy chủ riêng" server: Python/FastAPI bridge to Gemini | phase-quant-gemini-bridge.md |

Not yet started: Phase L (FULL-APP-PLAN.md's stretch list -- see Plan below).

## Known gaps (see each doc's own "Still open"/"Future work" for more)

- No database until Persistence; still true if `DATABASE_URL` is unset.
- Order book stays synthetic even on live VCI data; symbol fundamentals
  (P/E, sector, ...) are hand-seeded, not from a real data vendor.
- Real-time price feed is pull-only (no WebSocket `ws.Hub` yet).
- Indicators beyond SMA/EMA/RSI/MACD (Bollinger, VWAP) are not built.
- The TradingView **Charting Library** (self-hosted, licensed) isn't
  integrated -- needs TradingView's GitHub-gated approval; the public
  embed widget (phase-0-mvp.md) is a placeholder, not that integration.
- Queued buy orders don't reserve cash; backtests ignore fees; a second
  backend process could double-fill a matched order (see phase-k.md /
  phase-persistence.md).
- `.cursorrules`' Java/Spring Boot stack is not followed -- see CLAUDE.md.

## Plan (where to pick up)

1. **Deploy** -- now possible on a free host with a managed Postgres
   (e.g. an always-free VM with `./run.sh`, or Render/Fly + a hosted
   Postgres): set `DATABASE_URL`, a real `JWT_SECRET`, secure cookies
   behind HTTPS, and `NEXT_PUBLIC_API_BASE_URL`. Check `./run.sh` with
   the `db` service on a machine with Docker (not available in every
   sandbox this project has been built in).
2. **Real-world checks no sandbox so far could do**: live VCI/Binance
   prices, Quant with a real API key, phone layouts on a real device.
3. **Phase L** (FULL-APP-PLAN.md's stretch list): standalone screener,
   trade journal, Strategy Builder, remaining Settings sections, English
   localization. Also: crypto Quant, a crypto Portfolio page, crypto
   backtests, Quant-Chart (Phase H's deferred stretch item).
4. Smaller follow-ups, listed in more detail in phase-k.md and
   phase-persistence.md: reserve cash for queued buys; fees in backtests;
   return store errors instead of logging them; `SKIP LOCKED` on the
   matcher's scan before running more than one backend.
