# VN Stock Sim — full application implementation plan

Source: the design canvas at https://claude.ai/artifact/AfX6TBap6w7bSLpGCutrVp
("VN Stock Sim — bo giao dien cot loi"), 20 distinct desktop screens plus
English translations, mobile layouts, and a crypto-mode reskin (55 files
total). This plan was written after reading every core screen file in
full (not just the thumbnail), so the backend/frontend breakdown below is
grounded in the actual markup, sample data shapes, and interaction notes
left on the canvas, not a guess from the screen names.

**Superseded as the design-system source of truth, 2026-09-22:** the
user supplied a precise, machine-readable export of this same canvas at
`design/` (copied into this repo -- see `design/README.md`), with exact
hex values, exact token names, and an exact type/spacing/radius scale in
`design/tokens.css`/`design/tokens.json`, plus `design/DESIGN-SYSTEM.md`
and `design/SCREENS.md`. Section 1 below (written from the earlier,
approximate artifact-reading pass) is kept for narrative context, but
every later phase (C onward) must read `design/DESIGN-SYSTEM.md` and the
relevant `design/screens/*.dc.html` file directly as the literal spec,
not this section's prose summary. See phase-design-alignment.md for the
correction this triggered in Phase A's already-built tokens/shell.

How this fits the rest of the repo: CLAUDE.md still governs architecture
conventions (Go Handler to Service to Adapter, feature-based packages,
api-spec.md response envelope, Next.js Server Components fetching
server-side). RESUME.md still tracks what is actually built and verified
at any given moment; this file is a forward-looking plan, not a status
log. When a step here is completed, record it in RESUME.md the same way
every prior feature in this repo was recorded, including what was
verified and how.

Ground rule carried over from every phase already built in this repo: a
step is not done because the code compiles. It is done when it has been
run (via Docker or the dev server), the actual output was inspected (curl,
browser pane, or both), and that verification is written down. Several
bugs already found in this repo (the stock detail page and its own chart
disagreeing on price, the frontend build-time-prerender trap) were only
caught this way, not by reading the diff.

---

## 0. Screen inventory

Every screen the canvas defines, whether it already has a rough
equivalent in this repo, and which phase below builds it.

| Design file | What it is | Current state | Phase |
|---|---|---|---|
| Login.dc.html | Sign in | Built (frontend/src/app/login), different visual style | A |
| Signup.dc.html | Create account, starting capital picker | Built (register page), missing capital picker + market interest | A, G |
| Account-Menu.dc.html | Avatar popover, multi-portfolio switcher | Not built | G |
| Main.dc.html | Market overview: indices, heatmap, movers, account summary | Only a plain stock table exists (/stocks) | B, C |
| Detail.dc.html | Ticker detail: chart, indicators, order ticket, order book | Partially built (chart + AI insight), no order ticket/book | D |
| Replay.dc.html | Candle-by-candle replay with masked future and skill score | Not built at all | F |
| Portfolio.dc.html | KPIs, equity curve, positions, pending orders, journal | Backend exists (portfolio/order), no frontend page | E |
| Quant.dc.html | LLM chat: screener + strategy drafting | Not built (current AIInsightCard is rule-based, not chat) | H |
| Quant-Chart.dc.html | Select chart region, radial strategy-template menu | Not built | H (stretch) |
| Settings-AI.dc.html | Bring-your-own AI provider key, permissions, usage | Not built | H (build first, before the chat itself) |
| Crypto-Main/Detail/Replay.dc.html | Same three screens reskinned for a pairs market | Not built, no crypto data model at all | I |
| Mobile-*.dc.html (6 screens) | Responsive layouts of Market/Detail/Portfolio/Replay/Quant/Settings | Not built (current frontend is desktop-only) | J |
| Standalone screener, standalone heatmap, Strategy Builder, standalone Backtesting UI, standalone Trade Journal | Marked "WILL" (planned, not drawn) on every nav in the canvas itself | Backend backtest exists but has no UI | K |

---

## 1. Design system, as built (not as guessed)

Pulled from the canvas notes and confirmed against the actual markup in
every screen read:

- Background #0F0F0E, cards #171715, borders #2C2C28, hover/active surface
  #23231F, secondary surface #1A1A17 / #1F1F1C.
- Fonts: Lora for headings, Be Vietnam Pro for UI text, IBM Plex Mono for
  every number (prices, quantities, percentages). All three loaded from
  Google Fonts in the canvas markup.
- Accent #E08A3C (orange), used for primary buttons, the active nav item,
  and the Replay/Quant highlight color.
- VN price color convention, used everywhere a price or change appears:
  up #35C77F, down #FF5C5C, reference price #F0C243, ceiling price
  #C08BFF, floor price #4FD3E8. This is a real Vietnamese exchange
  convention (tran/tham chieu/san), not an arbitrary palette choice, and
  it must stay consistent even when the current design system built in
  PR #9 (also dark, but a different palette) is replaced.
- Locale: the canvas is Vietnamese-first with English variants that reuse
  the exact same layout (a note on the canvas explicitly says Vietnamese
  labels run 15 to 30 percent longer than English, so anything that fits
  the VI layout fits the EN one). Numbers use VN formatting (dot
  thousands, comma decimals, e.g. "1.412,58"), dates as DD/MM/YYYY.
- Two nav shells: a full sidebar (236px, logo + labeled links + virtual
  balance card) on Main/Portfolio/Quant, and a collapsed icon-only rail
  (72px) on Detail/Replay/Quant-Chart/Settings, where screen space for the
  chart matters more. Both link to the same routes; this is a per-page
  layout choice, not two different navigation trees.
- Guest-first, per the canvas notes: Replay, charts, and (eventually) the
  screener should work with no account. The sign-in prompt is meant to
  appear at the first paper order, not on page load, because that is the
  first action that actually needs a persisted account.

---

## 2. Decisions to make before writing code

These are called out explicitly because getting them wrong means
reworking every later phase.

### 2.1 One portfolio per user becomes many

Today, backend/internal/portfolio.MemoryStore keys everything by userID
directly: one cash balance, one position set per user. The canvas design
(Account-Menu.dc.html) shows a user switching between several named
portfolios ("Danh muc chinh", "Thu chien luoc RSI", "Vi crypto"), each
with its own NAV and its own currency (VND for stock portfolios, USDT for
a crypto one). This needs a real Portfolio entity: id, userID, name,
market ("stock" or "crypto"), startingCapital, currency, createdAt. Every
existing portfolio/order/watchlist call needs a portfolioID added
alongside userID. This is the single largest schema change in this plan
and should happen early (Phase B), before Replay and Quant sessions start
assuming "the user has exactly one portfolio."

### 2.2 The AI provider is bring-your-own-key, not a company-funded call

Settings-AI.dc.html is unambiguous about this: the provider list is
"Quant Cloud" (a free company-hosted tier, explicitly shown as the
non-default option in the mockup and requiring VN Stock Sim to pay for
inference — treat this as out of scope, it needs a business decision this
plan cannot make), "Claude (your key)", "OpenAI (your key)", and
"self-hosted OpenAI-compatible endpoint". The key is described as
"encrypted and stored only on this device — VN Stock Sim cannot read your
key." This matches the caution already exercised in this repo (the
existing AI Insight feature was deliberately kept rule-based rather than
silently wiring a paid LLM call). Phase H builds the Settings screen and
BYOK storage first, then the chat itself, so there is never a point where
Quant makes a model call without the user having explicitly supplied and
tested their own key.

### 2.3 Crypto is a second, parallel asset class, not a reskinned stock

Per the canvas note on cryptoRules: crypto mode drops VN-specific
concepts entirely (ceiling/reference/floor, trading sessions, lot sizes,
LO/MP/ATC order types, P/E, ROE) and adds its own (24/7 status, pairs
like BTC/USDT, an order book with cumulative depth, Market/Limit/
Stop/OCO order types, circulating vs max supply). This is enough
divergence that it should be a parallel `internal/crypto` backend feature
set with its own mock provider, not a set of conditionals bolted onto
the existing `symbol`/`market` packages. Phase I is scoped as its own
phase specifically so stock-market work in earlier phases is not blocked
waiting on crypto decisions.

### 2.4 English is a translation pass, not a parallel build

Because the canvas explicitly confirms the EN layout is pixel-identical
to the VI one, i18n should be a string-dictionary swap (next-intl or a
hand-rolled dictionary keyed by the same strings already hardcoded in
each page), applied once the VI version of a screen is done and stable —
not built screen-by-screen alongside VI. This plan treats English as
Phase L (stretch), matching the canvas's own EN rows being a straight
copy of the VI rows.

### 2.5 Order book and Replay scoring both need an explicit algorithm, not just a UI

Two things in the canvas have no natural backend equivalent yet and need
a concrete formula decided now so Phase D and Phase F are not blocked
mid-build:

- **Order book depth** (Detail.dc.html, Crypto-Detail.dc.html): propose
  generating it the same way market.MockProvider.GetBars already does —
  a pure function of (symbol, current price, t) so repeated requests in
  the same second agree, with N price levels above and below the last
  close, spaced by tickSize, and a deterministic pseudo-random volume per
  level.
- **Replay skill score** (Replay.dc.html): the mockup shows an overall
  score plus four sub-scores (entry quality, exit quality, stop-loss
  discipline, position sizing). Propose: entry/exit quality scored by
  how close each fill was to the best price achievable in a small window
  around it (already knowable server-side once the session is over,
  since the full historical range exists even though the client could
  not see it); stop-loss discipline scored by whether a stop was set at
  all and whether it was honored; position sizing scored by variance in
  position size relative to account equity across trades in the session.
  This is a heuristic, not a real skill assessment — label it as such in
  the UI copy, the same way the existing AI Insight feature is labeled
  "rule-based preview" rather than presented as authoritative.

---

## 3. Phase A — Foundation: layout shell and design tokens

Goal: every later screen can be built against a shared shell instead of
each page reinventing the sidebar, without redoing work already merged
(Navbar, Button, session/auth) that still needs to keep working.

1. **Backend output:** none. This phase is frontend-only.
2. Add the VN price color convention and the canvas's font stack as
   Tailwind theme tokens (extend, do not replace, the existing globals.css
   design tokens from PR #9 — check with the user before discarding that
   palette outright, since it was an explicit prior request to follow
   luxalgo.com; this plan assumes the new design supersedes it but that
   is worth confirming before deleting the old one).
   **Output:** `frontend/tailwind.config` (or `globals.css` `@theme`) has
   named colors `price-up`, `price-down`, `price-ref`, `price-ceiling`,
   `price-floor` and the Lora/Be Vietnam Pro/IBM Plex Mono font faces
   loaded. Verified by a throwaway `<div>` on any page rendering all five
   colors and three fonts correctly in the browser pane.
3. Build `frontend/src/components/SidebarNav.tsx` (full 236px variant,
   used on Main/Portfolio/Quant) and `frontend/src/components/RailNav.tsx`
   (72px icon-only variant, used on Detail/Replay/Settings/Quant-Chart),
   both reading the same nav item list (route, label, icon, "WILL" badge
   flag) from one shared `frontend/src/lib/navItems.ts` so the two shells
   never drift out of sync with each other.
   **Output:** both components render with real routes for the six
   screens that will exist by the end of this plan (Main, Detail,
   Replay, Portfolio, Quant, Settings) and a visible but disabled "WILL"
   badge for every screen still in Phase L. Verified visually in the
   browser pane at desktop width.
4. Replace the current flat `Navbar` with `SidebarNav`/`RailNav` on a
   per-page basis (each page picks which shell it uses, matching the
   canvas's own choice per screen).
   **Output:** existing `/stocks`, `/stocks/[symbol]`, `/chart` pages
   still build and render with a shell, even before their content is
   redesigned in later phases. `npx tsc --noEmit && npx eslint . && next
   build` all clean; confirmed in the Docker stack per this repo's
   existing verification habit.

---

## 4. Phase B — Backend data model rework

Goal: give the frontend real endpoints for everything Main.dc.html and
Detail.dc.html need that does not exist yet, and land the multi-portfolio
schema change from 2.1 before anything downstream depends on the old
single-portfolio shape.

1. **`internal/portfolio` rework:** add a `Portfolio` type (id, userID,
   name, market, startingCapital, currency, createdAt) and a
   `PortfolioStore` keyed by portfolio id instead of user id directly;
   `MemoryStore.ApplyFill` and every read method take a portfolioID.
   Add `POST /api/v1/portfolios` (create, called at signup and from the
   Account-Menu "create new portfolio" button), `GET /api/v1/portfolios`
   (list, for the Account-Menu switcher), `GET /api/v1/portfolios/:id`.
   **Output:** curl-verified: create two portfolios for one user, confirm
   they carry independent cash balances after separate `ApplyFill` calls.
2. **`internal/order`, `internal/watchlist`:** thread portfolioID through
   the same way; existing single-portfolio callers (frontend not built
   yet, so no breaking change to ship) get a default portfolio created
   automatically at registration so Phase A/pre-existing `/stocks` flows
   keep working without a portfolio picker yet.
   **Output:** `go build ./... && go vet ./...` clean (verified the same
   way every prior backend change in this repo was verified: no local Go
   install, so via a `golang:1.22-alpine` container).
3. **`internal/market` gains indices:** VN-Index, VN30, HNX-Index,
   UPCOM-Index as a new `GetIndex(name string) IndexSnapshot` on
   `market.MockProvider`, generated the same deterministic way as
   individual symbol bars (a pure function of (name, t), not an iterative
   walk — apply the lesson from the stock-detail price-consistency bug
   directly here rather than re-discovering it). `GET
   /api/v1/market/indices` returns all four with value, day change,
   change percent, and a sparkline-ready short bar series.
   **Output:** curl returns all four indices with plausible values;
   calling it twice a few seconds apart returns identical numbers (this
   is exactly the regression class already fixed once in this repo —
   write it correctly the first time here).
4. **New `internal/screener` package** (backs both the sector heatmap and
   the eventual Phase L standalone screener): `GetSectorHeatmap()`
   groups the existing mock symbol fixtures by `symbol.Sector` and
   returns per-sector and per-ticker day change percent;
   `GET /api/v1/market/heatmap`. `GetTopMovers(direction, limit)` sorts
   by day change percent; `GET /api/v1/market/movers?direction=up|down`.
   **Output:** curl confirms heatmap groups match the five existing mock
   sectors and movers are correctly sorted.
5. **`internal/market` gains an order book:** `GetOrderBook(symbol string)
   []PriceLevel` per the 2.5 algorithm (deterministic, pure function of
   symbol+current price+t). `GET /api/v1/market/orderbook?symbol=`.
   **Output:** curl confirms N levels above/below last close, spaced by
   the symbol's tickSize, stable across repeated calls in the same second.
6. **`internal/symbol` gains ceiling/reference/floor:** VN exchanges
   compute these from the previous close by a fixed percentage band per
   exchange (HOSE +-7%, HNX +-10%, UPCOM +-15%, approximately — confirm
   exact bands before hardcoding, this is real exchange rule, not a
   design choice). Add to `symbol.Detail`.
   **Output:** curl confirms ceiling > lastPrice > floor > 0 for every
   mock symbol, and reference sits between them.
7. **`internal/order` gains order types and fees:** extend `createRequest`
   to accept `LO | MP | ATC | Stop` (today only market/limit exist,
   mapped roughly to MP/LO); compute and return a simulated fee (the
   canvas shows 0.15%) on every fill.
   **Output:** curl-verified: placing each order type returns the
   expected status/fill behavior, and the fee shown matches 0.15% of
   order value.

---

## 5. Phase C — Rebuild Main (market overview)

Goal: replace the current plain `/stocks` table with Main.dc.html's
actual layout, backed entirely by Phase B endpoints.

1. Server Component `frontend/src/app/(app)/market/page.tsx` (or keep the
   existing `/stocks` route if preferred — decide route naming once,
   since Detail/Portfolio/Replay will all link to it) fetches indices,
   heatmap, movers, and (if logged in) the user's default portfolio
   summary and open positions, in parallel via `Promise.all`, the same
   pattern already used on the stock detail page.
   **Output:** page renders with `SidebarNav`, four index cards with
   sparklines, the sector heatmap grid, two movers tables, and (when
   authenticated) the right-column portfolio summary + open positions +
   Replay CTA card — matching Main.dc.html's structure. Verified in the
   browser pane at 1440px, both logged-in and guest states.
2. **Output:** guest state (no session) still renders indices/heatmap/
   movers — per the guest-first rule in section 1 — with the
   account-specific right column replaced by a sign-up prompt instead of
   erroring.
3. Full Docker rebuild + curl/browser verification, same habit as every
   prior frontend phase in this repo.

---

## 6. Phase D — Rebuild Detail (chart + order ticket + order book)

Goal: the existing `/stocks/[symbol]` page already has the candlestick
chart and AI Insight; this phase adds everything Detail.dc.html has that
those do not — the order ticket, order book, ceiling/floor display, and
fundamentals panel — and switches the page to `RailNav`.

1. `StockChart.tsx` gains an RSI and MACD sub-panel mode (the underlying
   `GET /market/indicators` already computes SMA/EMA; add `rsi` and
   `macd` server-side in `internal/market/service.go`, following the
   exact formulas already visible in the canvas's own reference JS
   implementation, which is a useful cross-check for the Go port).
   **Output:** `GET /api/v1/market/indicators?indicator=rsi` and
   `...=macd` return real values; spot-check a few points by hand against
   the canvas's own bundled RSI/MACD calculation for the same synthetic
   series shape.
2. New `frontend/src/components/OrderTicket.tsx` (Client Component: needs
   local state for price/qty/order-type before submit) posting to
   `POST /api/v1/orders` with the portfolioID from Phase B. Shows order
   value, the 0.15% fee, and remaining buying power, computed client-side
   from props for the price/fee preview but always re-validated
   server-side on submit (never trust the client number for the actual
   fill).
   **Output:** placing a market order from this component in the browser
   pane actually creates a position, visible by re-fetching the symbol
   detail page.
3. New `frontend/src/components/OrderBook.tsx` reading Phase B's
   `GET /market/orderbook`.
   **Output:** renders bid/ask levels with cumulative-style visual weight
   matching Detail.dc.html.
4. Ceiling/reference/floor chips and the fundamentals panel (market cap,
   P/E, P/B, ROE, EPS, avg 20-day volume — P/E and P/B already exist on
   `symbol.Detail`; ROE and avg volume are new fields to add to the mock
   fixture and, for avg volume, computable from the last 20 bars
   server-side rather than hand-seeded).
   **Output:** all fields present and non-zero for every mock symbol.

---

## 7. Phase E — Rebuild Portfolio

Goal: Portfolio.dc.html's tabs, KPIs, equity curve, and journal, backed
by real backend computation, not hardcoded numbers.

1. **Backend: equity history.** Nothing today snapshots a portfolio's NAV
   over time — `portfolio.Service.Summary` only computes the current
   instant. Add a `portfolio.MemoryStore` method that appends a
   `(timestamp, nav)` point whenever a fill happens, plus a daily
   snapshot job (simplest V1: compute it on read, from order history plus
   current mark-to-market, rather than a real background job — a real
   scheduled snapshot is a Phase L-level refinement). `GET
   /api/v1/portfolios/:id/equity-history`.
   **Output:** curl after a few test fills shows a plausible, monotonic-ish
   history.
2. **Backend: KPIs.** Win rate, profit factor, and max drawdown already
   exist in `internal/backtest`'s result computation for strategy
   backtests — port the same formulas to compute them over a portfolio's
   real closed-order history instead of a backtest's simulated trades.
   `GET /api/v1/portfolios/:id/stats`.
   **Output:** curl-verified against a hand-computed example (a portfolio
   with a known sequence of test fills, checked by hand).
3. **Backend: sector allocation.** Derivable purely from existing
   positions joined against `symbol.Sector` — no new storage, just a
   computed field on the summary response.
4. Frontend `frontend/src/app/(app)/portfolio/page.tsx`: four tabs
   (Overview/Positions/Pending/Journal) as a Client Component wrapping
   Server-Component-fetched data (fetch once, tab-switch client-side
   rather than a full page reload per tab, since the canvas shows this as
   instant tab switching).
   **Output:** matches Portfolio.dc.html's KPI row, equity curve chart
   (reuse the existing StockChart/lightweight-charts pattern with a line
   series instead of candles), positions table, sector allocation bars,
   pending orders with a working cancel button (`POST
   /orders/:id/cancel` already exists server-side), and journal stats.

---

## 8. Phase F — Replay Mode

Goal: the single feature in this plan with no existing backend surface
at all. Per RESUME.md's own priority order this was already flagged as
V2's signature feature — this phase is the actual build.

1. **Backend: `internal/replay` package.** A `Session` (id, userID,
   portfolioID, symbol, resolution, startBar, totalBars, currentBar,
   status) backed by a `MemoryStore` (same pattern as every other V1
   feature — Postgres migration is a later, separate concern per
   RESUME.md's existing plan, not specific to Replay).
   `POST /api/v1/replay/sessions` (start: symbol + a historical date range
   to replay, pulled from the existing deterministic `market.GetBars`
   data — since that data is already a pure function of time, "replaying
   2021" just means requesting bars for that historical range, which the
   generator already produces consistently).
   `POST /api/v1/replay/sessions/:id/advance` (reveals the next candle,
   returns it).
   `POST /api/v1/replay/sessions/:id/orders` (place an order at the
   current revealed candle only — reject if it references a price/time
   outside what has been revealed, which is the actual mechanism that
   makes "cannot preview the future" real rather than just a UI mask).
   `POST /api/v1/replay/sessions/:id/end` (compute final stats + skill
   score per the 2.5 formula, mark session complete).
   **Output:** curl-scripted full session end to end: start, advance
   several times, place an order, advance to the end, confirm the score
   response has the four sub-scores plus an overall number.
2. **Frontend `frontend/src/app/(app)/replay/page.tsx`.** Candlestick
   chart showing only revealed candles (reuse `StockChart`, but the data
   passed in is exactly the revealed subset — the "future hidden" visual
   mask from Replay.dc.html can be a styled overlay `<div>` positioned
   over the chart's own right edge rather than something the chart
   library needs to know about). Playback controls (step/auto-play/speed/
   scrub) drive calls to `/advance`. Order ticket calls
   `/replay/sessions/:id/orders`. Session results + skill score panel
   render once `/end` has been called.
   **Output:** a full Replay session playable in the browser pane,
   start to finish, with orders actually recorded and a score at the end.
   This is the single most important verification in this entire plan —
   confirm it in the browser pane, not just via curl, since the
   step-by-step reveal and the "cannot see the future" constraint are
   fundamentally UI-and-backend-together behavior.

---

## 9. Phase G — Account menu and Settings shell

Goal: Account-Menu.dc.html's multi-portfolio switcher, and a Settings
page shell with the subnav from Settings-AI.dc.html (seven of its eight
sections stay "WILL" per the canvas itself — only AI Model gets built in
this phase, the rest in Phase L if ever).

1. `frontend/src/components/AccountMenu.tsx` (Client Component popover,
   triggered from the avatar button already present in `SidebarNav`/
   `RailNav`): lists the user's portfolios from Phase B's
   `GET /portfolios`, a "create new portfolio" action, and the existing
   `logoutAction`.
   **Output:** switching portfolios in the popover changes which
   portfolio the rest of the app (Portfolio page, order ticket) operates
   against — store the active portfolioID the same way the session token
   is stored, in an httpOnly cookie set by a small Server Action, so
   Server Components can read it without a client-side fetch.
2. `frontend/src/app/(app)/settings/layout.tsx` with the subnav; only
   `frontend/src/app/(app)/settings/ai/page.tsx` gets real content this
   phase, every other subnav item renders a "WILL" placeholder matching
   the canvas's own badge convention.
   **Output:** the Settings shell navigates correctly between sections
   even though most are placeholders — verified in the browser pane.

---

## 10. Phase H — AI Model settings (BYOK) and Quant

Goal: build the settings surface from 2.2 first, then the actual chat
feature, so a real model call never happens before the user has supplied
and tested their own key.

1. **Frontend-only key storage, deliberately not sent to the backend at
   rest.** Per Settings-AI.dc.html's own privacy copy ("VN Stock Sim
   cannot read your key"), store the API key in the browser
   (`localStorage`, since this is the real app and not a Claude Artifact
   — plain `localStorage` behind the httpOnly-cookie-protected session is
   the right call: the key never needs to leave the user's browser
   except in the direct call to the model provider). Backend only ever
   sees a per-request proxy call if the user opts into "test connection,"
   never a persisted key.
   **Output:** `frontend/src/app/(app)/settings/ai/page.tsx` renders the
   provider radio group (Claude/OpenAI/self-hosted; the "Quant Cloud"
   free-tier option is disabled/greyed with a note that it needs a
   platform decision, per 2.2 — do not silently build a working free
   tier that bills the business), API key input, model/timeout selects,
   temperature slider, permission toggles, and data-scope checkboxes,
   all persisted to `localStorage`.
2. **"Test connection" backend proxy:**
   `internal/quant` package, `POST /api/v1/quant/test` accepts
   `{ provider, apiKey, model }` in the request body (never stored),
   makes one real call to the named provider on the user's behalf, and
   returns success/failure plus latency and token count — matching the
   "test before saving" panel in Settings-AI.dc.html exactly.
   **Output:** with a real key supplied by the user during manual
   testing, confirm a live round trip to the chosen provider succeeds and
   the UI shows the latency/token numbers. Without a key, confirm a clean
   error message, not a crash.
3. **Quant chat backend:** `POST /api/v1/quant/chat` — again, the API key
   travels in the request (from the frontend's localStorage), never
   stored server-side. The backend's job is to: build a system prompt
   describing VN Stock Sim's own domain (available indicators, mock
   symbol universe, the fact this is a paper-trading simulator and
   nothing here is investment advice — mirroring the disclaimer text
   already present throughout Quant.dc.html), call the named provider,
   and parse the response into the structured shapes the frontend needs
   (filter conditions, a screener result set actually run against real
   `internal/screener` data, and/or a drafted strategy in the same
   rule shape `internal/backtest` already understands).
   **Output:** a real prompt like the canvas's own example ("loc co
   phieu HOSE RSI duoi 35, ROE tren 15%...") produces a real filtered
   result set from actual mock data, not a hallucinated one — the model
   should be instructed to emit structured conditions that the backend
   then executes exactly like Phase B's screener, rather than trusting
   the model's own arithmetic on the stock universe.
4. **Frontend Quant chat:** `frontend/src/app/(app)/quant/page.tsx`, chat
   thread UI matching Quant.dc.html (conditions-as-chips, results table,
   drafted-strategy card with "open in backtest"/"try in Replay"
   buttons wired to Phase F's Replay session start and the existing
   `internal/backtest` endpoint).
   **Output:** a full conversation in the browser pane: ask a filter
   question, see real matching tickers from the mock universe; ask for a
   strategy, see a real backtest run against it via the existing
   `POST /backtests` endpoint.
5. **Quant-Chart (stretch within this phase):** the "select a chart
   region, pick a strategy template from a radial menu" interaction is a
   frontend-only affordance (canvas selection math + an SVG radial menu)
   that calls the same `/quant/chat` endpoint with the selected date
   range and a strategy-template hint pre-filled instead of free text.
   Build this only after steps 1 to 4 are solid — it is a UI layer on
   top of the same backend call, not new backend work.

---

## 11. Phase I — Crypto market

Goal: a second, parallel asset class per 2.3's decision, reusing as much
of the stock-market pattern as makes sense without forcing crypto into
concepts that do not apply to it (see cryptoRules note in section 2.3).

1. **Backend: `internal/pair` (crypto's equivalent of `internal/symbol`)**
   and **`internal/cryptomarket` (equivalent of `internal/market`)** —
   separate packages rather than branching the existing stock ones, per
   2.3. Mock fixtures: a handful of real pairs (BTC/USDT, ETH/USDT,
   BNB/USDT), 24/7 bar generation (no session-hours gaps, unlike the
   stock generator which could stay daily-only), Market/Limit/Stop/OCO
   order types, circulating/max supply, distance from all-time high.
   **Output:** `GET /api/v1/pairs`, `GET /api/v1/pairs/:pair`,
   `GET /api/v1/cryptomarket/bars` all curl-verified with real-looking
   24/7 continuous data (no weekend gaps, unlike the stock bars).
2. **Backend: crypto portfolios.** A portfolio's `market` field from
   Phase B (2.1) already distinguishes stock vs crypto — crypto orders
   post against a crypto-market portfolio's USDT balance instead of a
   stock portfolio's VND balance. Reuse `internal/order` and
   `internal/portfolio` with the pair/symbol type made generic rather
   than duplicating the whole order/portfolio stack.
3. **Frontend: mode switch** in `SidebarNav`/`RailNav` (the STK/CRY pair
   toggle from the canvas), `frontend/src/app/(app)/crypto/...` routes
   mirroring the stock routes' structure (market overview, pair detail,
   crypto Replay), each page a thinner wrapper reusing
   `StockChart`/`OrderTicket`/`OrderBook` components with crypto-shaped
   props rather than rebuilding them.
   **Output:** switching STK to CRY in the nav and landing on a working
   BTC/USDT detail page with a real order book and a placeable order,
   verified in the browser pane.

---

## 12. Phase J — Mobile responsive layouts

Goal: per the earlier read of Mobile-Market.dc.html, mobile is a genuine
responsive layout of the same data and endpoints already built in Phases
C to I, not new backend surface. This phase is Tailwind breakpoint work.

1. Bottom tab navigation component (`frontend/src/components/MobileTabBar.tsx`)
   matching the canvas's five-tab-plus-center-Replay-FAB layout, shown
   only below a chosen breakpoint (the canvas mobile frames are 390px
   wide — use that as the design reference point, standard `sm:`/`md:`
   Tailwind breakpoints for the actual responsive rules).
2. For each of Main/Detail/Portfolio/Replay/Quant/Settings, add
   responsive variants of the existing desktop layout rather than
   separate mobile page files, per this repo's existing mobile-check
   habit (every phase in this repo so far has been verified at 375x812
   in the browser pane in addition to desktop widths — continue that,
   do not skip it for this phase specifically since it is the entire
   point of the phase).
   **Output:** every screen built in Phases C to I checked at 375px width
   in the browser pane, confirmed no horizontal overflow, tap targets are
   reasonably sized, and the bottom tab bar does not overlap content.

---

## 13. Phase L — Stretch: the "WILL"-badged items

> **Re-scoped (2026-09-25):** checking master after Phase J showed four V1
> gaps still open (orders never filling, fees not charged, no watchlist
> UI, no backtest page). A phase called "Phase K" closed those instead
> (finish-V1 work, not the stretch list below); see `phases/phase-k.md`.
> The stretch list that was originally planned as Phase K moved here,
> to **Phase L**; its item 2 (standalone heatmap) was done in Phase I
> and item 5 (backtesting UI) in Phase K.

Everything the canvas itself marks as planned-but-not-drawn. No screen
exists yet for any of these in the design artifact, so building them
means designing the screen first (or asking for a canvas update), not
just wiring a known layout to a known endpoint. Listed in the rough order
the canvas's own badges imply usefulness:

1. **Standalone stock screener** — `internal/screener` already exists
   from Phase B (it currently only backs the Main heatmap and Quant's
   filter execution); a dedicated screener UI with saved filter presets
   is mostly frontend work once B is in place.
2. **Standalone heatmap** (full-screen version of Main's embedded one) —
   same backend, a dedicated route.
3. **Standalone Trade Journal** (full-screen version of Portfolio's
   Journal tab) — same backend from Phase E, a dedicated route with more
   room for per-trade notes than the tab currently allows.
4. **Strategy Builder** (no-code rule builder, distinct from Quant
   drafting a strategy from natural language) — needs its own UI design
   before backend work starts; the underlying rule shape can reuse
   `internal/backtest`'s existing rule format.
5. **Standalone Backtesting UI** — `internal/backtest` already has a real
   endpoint; this is purely a frontend form + results page that has
   never been built (RESUME.md has flagged this gap since the backend
   was first scaffolded).
6. **Remaining Settings sections** (Account, Market Data, Paper Account,
   Replay & Scoring, Notifications, Appearance, Privacy) — each needs its
   own screen design; none exist in the current canvas beyond the AI
   Model one built in Phase H.
7. **English (EN) localization** — per 2.4, a dictionary-swap pass once
   every VI screen above is stable, not built alongside each screen.

---

## 14. What this plan deliberately does not decide

- Whether the old luxalgo.com-inspired design system from PR #9 is kept
  anywhere (e.g. as a lighter "marketing" theme) or fully replaced by
  this darker, VN-price-convention palette — flagged in Phase A step 2,
  worth a direct answer before that step starts.
- The real Vietnamese market-data vendor (RESUME.md's existing plan) and
  the real crypto data vendor for Phase I — both still explicitly need
  the user to choose and pay for a licensed provider; nothing in this
  plan assumes that has happened.
- Whether "Quant Cloud" (the company-funded free AI tier shown in the
  mockup) ever gets built — Phase H builds only the bring-your-own-key
  path and leaves that option visibly disabled with an explanatory note,
  per 2.2.
- Postgres persistence (RESUME.md's existing plan) — every new store
  introduced in this plan (Portfolio, Replay Session, crypto Pair/Order)
  follows the same in-memory `MemoryStore` pattern already used
  everywhere else in this repo, on the same understanding that a real
  database swap is a later, separate migration behind the same small
  interfaces, not a redesign.
