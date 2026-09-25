# Phase 0 — initial V1 MVP scaffold (pre-Phase-A)

Not a lettered phase: this is the original backend + frontend build, from
before FULL-APP-PLAN.md's phase-by-phase rebuild started (Phase A onward).
Kept as its own file so RESUME.md doesn't have to carry it. Code comments
citing "RESUME.md"'s dated 2026-09-21 entries mean the content below.

## What shipped

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
- internal/insight — GET /symbols/:symbol/insight, a deterministic,
  rule-based "AI Insight" (momentum/trend/volume signals computed from
  the symbol detail + bars + SMA(20) the backend already has) --
  explicitly NOT a real LLM call, see the Visual design / AI Insight
  section below for why and what a real integration would need.
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

Bug found and fixed 2026-09-21: the API client's fetch used
next: { revalidate: 10 }, which let Next.js statically prerender /stocks
at `next build` time. In Docker, the frontend image builds before the
backend container exists, so that build-time fetch always failed and
baked the "could not reach the API" error page into the image -- it only
self-healed after the first background ISR revalidation succeeded once
the backend was actually running (up to 10s, or longer if nobody hit the
page to trigger a revalidation). Fixed by switching to cache: "no-store"
in src/lib/api.ts, which forces /stocks to render per-request instead of
at build time (confirmed via the next build route table: /stocks changed
from "○ Static" to "ƒ Dynamic"). Verified with a fresh
docker compose build --no-cache + ./run.sh -d + an immediate curl (no
delay) against /stocks and /stocks/VNM -- both returned real data on the
very first request.
- /chart — step 1 of the TradingView integration plan (see below).
- /stocks/[symbol] now renders a real candlestick chart
  (src/components/StockChart.tsx, using the lightweight-charts npm
  package -- deliberately not TradingView's own product, see the
  TradingView section below) with a 20-period SMA line overlay, for the
  last 6 months of daily bars. Data is fetched server-side in the page
  component (GET /market/bars and /market/indicators?indicator=sma) and
  passed to the chart as props, not fetched client-side -- the browser
  cannot resolve the "backend" hostname used inside Docker Compose, so a
  client-side fetch would fail there even though the server-side one
  works. If the chart fetch fails, the page still renders (price/
  fundamentals come from a separate, independent fetch) with a plain
  text fallback instead of a broken chart.

Note: frontend/AGENTS.md (written by next dev, not by this session) warns
that Next.js 16 has breaking API/convention changes vs older training
data — it was consulted (node_modules/next/dist/docs/) before writing the
pages above, e.g. to confirm params is still a Promise in dynamic routes.
Also hit and worked around eslint-config-next's new react-hooks/purity
rule, which flags Date.now() called directly in a component body (even a
Server Component, which isn't re-rendered the way the rule's rationale
assumes) -- fixed by moving it into a plain helper function.

Not built yet: indicators beyond SMA/EMA on the chart (RSI, MACD,
Bollinger, VWAP), auth pages, watchlist/portfolio/order/backtest UI.

Bug found and fixed 2026-09-21: every page under /stocks and /chart used
`<main className="mx-auto max-w-XX p-8">` directly inside the root
layout's `<body className="flex flex-col">`. Because auto margins
disable flexbox's default stretch behavior for the cross axis, `main`'s
width collapsed to shrink-to-fit its content (confirmed via computed
styles: ~412px) instead of filling up to its max-width -- so on a large
monitor the whole app rendered as a narrow, off-looking column with huge
empty margins on both sides, even though it was technically centered.
Fixed by adding `w-full` alongside `mx-auto max-w-*` on all three pages.
While fixing this, also widened the max-widths (/stocks: 3xl -> 5xl;
/stocks/[symbol]: 2xl -> 6xl; /chart: kept 5xl but now actually fills
it) and gave /stocks/[symbol] a real two-column layout at the lg
breakpoint (info sidebar + wide chart) instead of one long centered
column, and made the chart itself taller on lg screens
(StockChart.tsx: h-[400px] lg:h-[480px]). Verified visually in the
browser pane at 1920x1080 (all three pages now use the available width
correctly) and at 375x812 mobile (still stacks into one readable
column, no regression).

Visual design, 2026-09-21 — follows luxalgo.com (fetched and inspected
live via the browser pane, including computed styles -- not guessed):
a clean monochrome palette (white background, near-black text), fully
pill-shaped buttons (solid black primary, off-white-with-border
secondary), and dark near-black cards specifically for chart/product-
preview panels. Not their licensed "Aeonik" font -- Geist (already in
use) is a similar-feeling free alternative.

- src/components/Navbar.tsx -- new sticky top nav (logo, Stocks/Chart
  links, "Browse Stocks" pill CTA), added to the root layout so every
  page has it, matching luxalgo.com's persistent nav pattern.
- src/components/Button.tsx -- small shared pill-button component
  (primary/secondary variants) instead of repeating the styling inline
  everywhere.
- Home page (/) rebuilt as a proper hero: eyebrow badge, bold heading,
  subtext, two pill CTAs, then a dark rounded-2xl card below showing a
  *live* candlestick chart (VNM, reusing StockChart) as the product
  preview -- unlike LuxAlgo's static marketing graphic, this one is
  backed by real (mock) data, so it doubles as a demo of the app
  actually working. Server-fetches with cache: "no-store" (same fix as
  the earlier /stocks bug), confirmed via `next build`'s route table
  that / is "ƒ Dynamic" not statically prerendered.
- StockChart.tsx gained a `theme="dark"` prop (grid/text colors switch)
  and an optional `heightClassName` override, so it can be reused
  inside dark cards; /stocks/[symbol]'s chart panel and the new home
  page hero both use it. TradingViewWidget already had a theme prop
  from earlier work, reused the same way on /chart.
- globals.css: removed the leftover prefers-color-scheme dark-mode
  media query (this app is intentionally single-theme, matching
  LuxAlgo's default light theme, with dark treatment reserved for chart
  cards specifically -- not a full OS-driven dark mode toggle) and
  fixed body's font-family, which was hardcoded to "Arial, Helvetica,
  sans-serif" and silently overriding the Geist font already set up via
  next/font -- a leftover create-next-app default bug, not something
  this session introduced.

Bug found and fixed while building this: the stock detail page's stats
grid (grid-cols-2) had gap-y-4 but no gap-x, so the two columns butted
directly against each other with zero horizontal space -- "Market Cap"'s
wrapped value text ran straight into "P/E"'s value with no gap
(confirmed visually in the browser pane, not just in code review).
Fixed by adding gap-x-6.

Chart panels are now user-resizable (StockChart.tsx, TradingViewWidget.tsx)
-- DONE. Both chart containers use native CSS `resize: vertical`
(Tailwind's resize-y + overflow-auto + a min-height), which draws a
drag handle in the bottom-right corner. No custom drag-handling code
needed: lightweight-charts' autoSize and TradingView's own autosize
widget option both already watch their container's size (ResizeObserver
under the hood) and redraw to fit, so dragging the native handle just
works. Verified by actually dragging the handle in the browser pane on
both /chart (TradingView widget) and /stocks/[symbol] (StockChart) --
confirmed via getBoundingClientRect() that the container's height
changed and the chart canvas redrew to fill the new size with no
stretching artifacts, not just that the drag gesture completed.

AI Insight, 2026-09-21 -- DONE, but read this before assuming it is
what it sounds like. Follows luxalgo.com's "Quant" panel visually (dark
card, sparkle mark, short written read below the chart), per a
follow-up request to add this feature. It is deliberately NOT a real
LLM call: that needs an API key and has a real per-call cost, which is
the user's decision to make explicitly, not something to assume or
wire up silently. Instead:

- backend/internal/insight/service.go computes three deterministic
  signals from data the backend already has -- Momentum (today's
  change%), Trend (last close vs. last SMA(20) value), Volume (latest
  bar vs. trailing-20 average) -- each labeled bullish/bearish/neutral,
  plus a one-paragraph summary that counts bullish vs. bearish signals
  and picks a lean ("leans constructive" / "leans cautious" / "is
  mixed"). All thresholds and wording are in that one file, easy to
  read end to end.
- The API response has a `source: "rule-based"` field and the summary
  text itself says "not generated by a language model" -- both by
  design, so this stays honest at every layer (API contract, not just
  UI copy) and so a future real integration is a visible, deliberate
  upgrade (swap Source to "llm", replace buildSignals/buildSummary with
  an actual model call) rather than a silent one.
- frontend/src/components/AIInsightCard.tsx renders it with a visible
  "rule-based preview" badge -- the honesty is in the UI too, not just
  the API.
- GET /api/v1/symbols/:symbol/insight, public (same reasoning as
  symbol/market). Rendered on /stocks/[symbol] below the chart,
  independently fetched (try/catch) so an insight failure doesn't take
  down the rest of the page, same pattern as the chart data fetch.

Verified: curl against the live endpoint for VNM/VCB/HPG returned
correctly varying signals (e.g. VCB's Momentum came back bearish,
matching its actual negative changePercent). Visually verified in the
browser pane on both a bullish (VNM, "leans constructive") and a mixed
(VCB, "leans cautious") symbol, at desktop and mobile widths, plus
re-verified inside the full Docker Compose stack.

Real next step if the user wants an actual LLM behind this: pick a
provider, add the API key as a backend env var (never exposed to the
client -- the call would stay server-side, same as this rule-based
version), and replace insight.Service's buildSignals/buildSummary with
a call to that model, keeping the same Insight/Signal response shape
so the frontend needs no changes.

Bug found and fixed 2026-09-21: the stock detail page's top-of-page
quote (lastPrice/change/changePercent) and its own candlestick chart
could show two unrelated prices for "now" -- reported by the user
comparing /stocks/VNM's price to its chart. Root cause was two layers
deep:

1. symbol.MockProvider seeded lastPrice/change/changePercent as fixed,
   independent values per symbol, entirely unrelated to
   market.MockProvider's generated bars -- e.g. VNM's seeded lastPrice
   was 68500, while its chart's actual last close (from a random walk
   starting at a different seeded base) had drifted to ~38911 after
   180 simulated days. Fixed by making symbol.Service derive
   lastPrice/change/changePercent from market data instead: added
   symbol.QuotePort (LatestClose) and market.Service.LatestClose, and
   symbol.Service.Detail now overrides the provider's seeded price
   fields with the quote source's when available (main.go wires
   marketSvc into symbol.NewService as its QuotePort; symbol has no
   import dependency on market, only main.go connects them). The
   provider's seeded values remain as a fallback.
2. Even after that, GetBars/LatestClose still disagreed with each
   other: market.MockProvider.GetBars computed each bar as a running
   product accumulated iteratively from whatever "from" the caller
   passed, starting at a fixed seed price -- so the "current" close at
   a given instant depended on how far back the requested window
   started (the chart requests 180 days, LatestClose requested 10),
   producing two different answers for "today" from the same
   generator. Rewrote GetBars so each bar's open/close comes from
   closeFor(sym, t), a pure function of (symbol, absolute time) using
   two symbol-seeded sine waves plus small jitter -- no iteration, no
   dependency on the request's window length, so any window ending at
   the same instant now reports the same close for that instant. Also
   aligned from/to to the resolution's time grid (multiples of step)
   so two calls issued a few seconds apart (each computing "now"
   independently) still land on identical bar timestamps.

Verified end to end, not just by reading the diff: curl against
/symbols/VNM and /market/bars (both a 180-day and a 10-day window)
after the fix returned the exact same lastPrice/last-close
(64599.52 in all three), for VNM/VCB/HPG, inside the actual Docker
image (docker compose build + ./run.sh -d), and visually in the
browser pane where the page's displayed price and the chart's
right-edge price label now agree.

Also, per a follow-up request, chart resizing now works in both
directions (previously vertical-only): both StockChart.tsx and
TradingViewWidget.tsx use `resize: both` (Tailwind's `resize` class)
instead of `resize-y`, with a min-width added alongside the existing
min-height. The three dark card wrappers around these charts
(home page hero, /chart, /stocks/[symbol]) had `overflow-hidden`,
which would have invisibly clipped horizontal growth past the card's
original width -- removed it from all three. Verified in the browser
pane: StockChart's container measurably changed in both width and
height from a single drag (430x400 -> 295x280 in one test), and the
chart canvas redrew cleanly with no overflow past the card at the new
size, in both directions.

Follow-up 2026-09-21: user asked for confirmation that chart data
reaches "now" in UTC. It already did (verified: daily bars align down
to the start of the current UTC day, so the last bar is always dated
today -- confirmed via curl showing 2026-09-21T00:00:00 as the last
bar for a "now" of 2026-09-21 15:58 UTC), but nothing in the UI made
that visible -- the x-axis only shows month labels
(StockChart.tsx: timeVisible: false), so there was no way to tell at a
glance that the rightmost bar was actually current rather than stale.
Added a small "Data through <date> UTC" caption (computed from the
last bar's own timestamp, not a separate "now" call, so it can never
drift from what the chart is actually showing) next to both chart
panels that use StockChart.tsx (the stock detail page and the home
page hero preview) -- not added to /chart, which is TradingView's own
live widget and already shows real-time data with its own UI. Verified
in the browser pane on both pages, and inside the built Docker image
(the "Data through Sep 21, 2026 UTC" string is present in the served
HTML/RSC payload).

Auth pages (register/login/logout), 2026-09-21 -- DONE. The backend
already had auth (POST /auth/register, /auth/login, GET /auth/me) --
this wires the frontend to it:

- The JWT lives in an httpOnly cookie (frontend/src/lib/session.ts,
  cookie name vss_token), never exposed to client-side JS. Every
  authenticated call happens server-side, same pattern the rest of the
  app already uses for data fetching. secure: false on the cookie
  because this demo runs over plain HTTP everywhere (no TLS configured
  in docker-compose.yml) -- flip to true behind a real HTTPS
  deployment; noted directly in the code, not just here.
- frontend/src/lib/authActions.ts -- Server Actions (loginAction,
  registerAction, logoutAction) using useActionState's
  (prevState, formData) signature. Not routed through api.ts's
  apiFetch: these need the raw { code, message } error body on
  failure to show a real message on the form (e.g. "invalid email or
  password"), which apiFetch's generic thrown Error doesn't preserve.
- /login and /register pages (Client Components, useActionState +
  React 19's built-in pending state -- confirmed this is still the
  right pattern for Next.js 16 against the bundled docs, not assumed).
- Navbar.tsx is now async and reads the session (GET /auth/me,
  re-verified against the backend on every request rather than trusting
  a locally-decoded JWT -- V1 has no token revocation, so this is the
  only way to notice an expired/invalid cookie) to show either
  "Log In / Sign Up" or the user's display name + "Log Out". Because
  Navbar is in the root layout and reads cookies(), every route is now
  dynamically rendered (confirmed via next build's route table: even
  /chart, previously static, is now "Dynamic") -- expected and fine,
  every page already depends on live backend data anyway.
- api.ts's apiFetch gained an optional token option (used by the new
  getMe) for the Authorization: Bearer header the rest of V1's
  authenticated endpoints (watchlist/portfolio/order/backtest) will
  need next.

Verified end to end in the browser pane, not just by reading the code:
registered a real account, got redirected to /stocks with the Navbar
showing the display name; logged out, Navbar reverted to
Log In/Sign Up; logged back in with the same credentials, redirected
correctly; tried logging in with a wrong password and confirmed the
"invalid email or password" error renders inline on the form. Checked
mobile width too (form fields stack, nav collapses cleanly). Also
confirmed POST /api/v1/auth/register works end to end inside the
actual built Docker image via curl (not just the dev server).

TradingView integration, step 1 (src/components/TradingViewWidget.tsx,
/chart page) — DONE. charting-library-integration.md describes the
self-hosted Charting Library, which needs TradingView's GitHub-gated
access approval (a human has to request it, not something this session
can do). As a first, actionable step instead, this embeds TradingView's
public "Advanced Chart" widget (https://www.tradingview.com/widget/advanced-chart/,
free, no approval needed, just a script tag) showing BITSTAMP:BTCUSD.

Verified 2026-09-21 via the Next.js dev server in the browser pane:

- Hit a real bug and fixed it: the effect originally cleared and rebuilt
  the widget's DOM on every run, which under React Strict Mode's dev-only
  double effect invocation detached TradingView's script node mid-flight
  and crashed inside their code (Cannot read properties of null (reading
  querySelector), because document.currentScript.parentElement was null
  on the detached node). Fixed by making the effect idempotent (skip if
  the container already has content) instead of clearing/rebuilding.
- After the fix: a fresh page load in a new tab showed the widget's
  toolbar (symbol search, timeframe buttons, indicators button) render
  correctly with no console errors, and the accessibility tree showed an
  "advanced chart TradingView widget" iframe node, confirming TradingView's
  script executed and mounted its widget.
- NOT fully verified: the chart canvas itself stayed blank with zero OHLC
  values (O0 H0 L0 C0) in this environment's sandboxed preview browser --
  most likely that browser can't reach TradingView's real-time data
  backend (a network/sandbox limitation, not a code bug, since the UI
  shell that TradingView's own JS renders loaded correctly). Re-check in
  an ordinary browser with normal internet access before trusting this
  further.

Explicitly not started: the actual licensed integration
(TVDatafeedAdapter implementing IDatafeedChartApi/IExternalDatafeed
against our own /api/v1/market/bars, per charting-library-integration.md)
-- that needs the self-hosted Charting Library files, which needs
TradingView's approval first. See Future work below.

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

Full-app rebuild against the design canvas (FULL-APP-PLAN.md), 2026-09-22
-- IN PROGRESS. The user asked for the entire 20-screen design at
https://claude.ai/artifact/AfX6TBap6w7bSLpGCutrVp to be implemented,
phase by phase, with a planning/analysis .md file written before each
phase (phase-a.md, phase-b.md, ...) and the outcome recorded here. See
FULL-APP-PLAN.md for the full 11-phase breakdown (A-K).
