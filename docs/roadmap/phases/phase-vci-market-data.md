# Real market-data adapter (VCI) — planning and analysis

Not a FULL-APP-PLAN.md phase -- this is the user's explicit priority
redirect (see RESUME.md's top-of-file note), superseding the phase
sequence for this unit of work. User's instruction, verbatim: "go with
this way" (accepting the unofficial/undocumented-endpoint category
`api-spec.md` otherwise rules out, an explicit, informed override) "...
the module is 'Inversion Principle' mean work as adapt easy to change
later" (the swap must sit behind the existing port, not be wired in
directly).

## What was actually found (not guessed)

Researched via WebSearch/WebFetch, then a real live test request, in
the immediately preceding turns of this session:

- `vnstock` (the free, no-registration Python library found in the
  provider research) wraps several underlying sources. Cloned its
  GitHub source (`thinh-vu/vnstock`) and read `vnstock/explorer/vci/
  quote.py` and `const.py` directly -- this is VCI (Vietcap Securities,
  a real, licensed Vietnamese brokerage)'s own internal trading-platform
  API, not an officially published third-party developer API. Its own
  README says so explicitly: "Vnstock cấp quyền sử dụng phần mềm, không
  cấp quyền sử dụng dữ liệu của nguồn" (licenses the software, not the
  source data) -- this is the exact "undocumented public endpoint"
  category flagged earlier and now explicitly accepted by the user.
- Concrete, verified request shape:
  `POST https://trading.vietcap.com.vn/api/chart/OHLCChart/gap-chart`,
  JSON body `{"timeFrame": "ONE_MINUTE"|"ONE_HOUR"|"ONE_DAY", "symbols":
  ["FPT"], "to": <unix seconds>, "countBack": <int bars>}`, headers
  `Content-Type: application/json`, `Referer`/`Origin:
  https://trading.vietcap.com.vn/`, a browser-like `User-Agent`. No API
  key, no auth.
- Live-tested with real `curl` calls against the real endpoint (not
  assumed): `FPT` returned 5 real daily bars ending on a real, current
  trading day (Sep 22 2026, correctly skipping the Sep 19-20 weekend),
  plausible real prices (~65,000-68,000 VND). `VNINDEX` returned real
  index values (~1,798-1,841) and real total-market volume
  (~500-900M shares/day) via the same endpoint with `symbols: ["VNINDEX"]`.
  Response shape: one array element per requested symbol, each holding
  parallel arrays `t`/`o`/`h`/`l`/`c`/`v` (not row objects) plus
  `accumulatedVolume`/`accumulatedValue` (unused here).
- Index symbol codes (from `_VCI_INDEX_MAPPING` in `const.py`):
  `VNINDEX`, `VN30`, `HNXIndex`, `HNXUpcomIndex` -- not the same strings
  this app displays ("VN-Index", "HNX-Index", "UPCOM-Index"), needs a
  translation table.

## Why this is a high-leverage swap, not a narrow one

This app's `market.MarketDataProvider` port is already the single choke
point every price-dependent feature reads through, by design (CLAUDE.md's
dependency-inversion rule, applied consistently since the backend was
first scaffolded):

- `symbol.Service.Detail`'s `lastPrice`/`change`/`changePercent` and
  the Phase B `reference`/`ceiling`/`floor` bands are derived from
  `QuotePort.LatestClose`, which calls `GetBars` -- becomes real
  automatically.
- `screener.Service`'s heatmap/movers (`ChangePercent`, and Phase C's
  `Price`/`Volume` via `MarketPort.LatestQuote`) -- becomes real
  automatically.
- `market.Service.GetIndicator` (SMA/EMA, and Phase D's RSI, on the
  other branch) and `GetMACD` all compute from `GetBars` -- becomes
  real automatically.
- `backtest.Service` runs against `GetBars` -- becomes a real backtest
  against real historical prices automatically.
- `market.Service.GetIndex` -- swapped explicitly in this phase (below).

So swapping one adapter at the wiring root (`main.go`) makes nearly
every screen's numbers real, with zero changes to `symbol`, `screener`,
`backtest`, or any handler. This is exactly the payoff the user asked
for from the dependency-inversion shape.

## Decisions

1. **New adapter lives in its own file(s), implements the existing
   `market.MarketDataProvider` interface exactly** -- `GetBars`,
   `GetIndex`, `GetOrderBook`. No interface change, no caller change
   outside `main.go`'s wiring.
2. **Order book stays synthetic.** VCI's real order-book/depth data
   (its `market-watch`/intraday endpoints) was not confirmed in this
   research pass, and chasing it now would blow the scope of "swap the
   data source" into "reverse-engineer a second undocumented API."
   `GetOrderBook` keeps delegating to the existing deterministic
   generator. Flagged, not silently left inconsistent -- the order
   book will look "mock" (round, evenly-spaced levels) next to real
   candles; worth a follow-up if it matters later.
3. **Symbol static fundamentals stay hand-seeded mock.** VCI does have
   company/financials endpoints (`company.py`, `financial.py` in the
   cloned source), but swapping `symbol.Provider` (company name, sector,
   marketCap, P/E, P/B, ROE, EPS, dividendYield) is a separate adapter
   swap with its own response-shape research, out of scope for this
   pass. `symbol.MockProvider`'s fixture data is unchanged.
4. **Resilience: wrap VCI in a fallback decorator, not a bare swap.**
   `LiveProvider` tries the VCI adapter first; on any error (network,
   non-200, unexpected JSON shape -- all realistic for an undocumented
   upstream that owes this app nothing) it falls back to the existing
   `MockProvider` for that call and logs a warning, rather than
   5xx-ing the whole page. This is itself another adapter behind the
   same port -- composing two `MarketDataProvider`s -- so it doesn't
   compromise the "easy to swap later" property; it's what makes the
   swap safe to ship at all given the upstream is unofficial.
5. **Short-lived in-memory cache (per process, a few seconds TTL) in
   front of the live calls.** Partly performance (a page load fans out
   to several `GetBars`/`GetIndicator`/`GetIndex` calls that often
   share the same underlying series), partly good citizenship -- this
   is someone else's undocumented internal API with no published rate
   limit; hammering it on every request is the kind of behavior that
   gets endpoints changed or blocked, which would break this adapter
   for everyone using it, not just this app.
6. **Config: `MARKET_DATA_SOURCE` env var, default `vci`.** The user
   asked for this to be live now, not opt-in-later; defaulting to `vci`
   fulfills that directly, with `mock` still available (e.g. `docker-
   compose.yml` could pin it back to `mock` for a fully offline demo)
   by construction of the port, not because this phase hard-codes a
   fallback path only.
7. **Resolution mapping is approximate for intraday, flagged as such.**
   VCI's own interval map (`_INTERVAL_MAP` in the cloned source) sends
   `1m`/`5m`/`15m`/`30m` all to the same `ONE_MINUTE` granularity server-
   side -- so this app's `"5"`/`"15"` resolutions will come back as real
   1-minute bars (more granular than requested, not less), not a
   dropped feature. `"60"` maps to `ONE_HOUR`, `"1D"`/default to
   `ONE_DAY`, both exact.
8. **`countBack` capped, not unbounded.** VCI's endpoint takes a bar
   count, not a date range; this app's callers pass `(from, to)`. The
   adapter estimates a bar count from the requested span (business
   days for daily, a generous per-day estimate for intraday) and caps
   it (2000) -- both to keep single requests reasonable for an
   unofficial endpoint (decision 5's citizenship concern) and because
   `to` (Phase D's 5-year timeframe pill) could otherwise imply an
   enormous `countBack` for 1-minute data.

## Files to add

- `backend/internal/market/vciprovider.go` -- `VCIProvider` struct
  implementing `GetBars`/`GetIndex` against the real endpoint;
  `GetOrderBook` delegates to a wrapped `*MockProvider` (decision 2).
- `backend/internal/market/liveprovider.go` -- `LiveProvider` decorator
  (VCI-first, mock-fallback-on-error, decision 4) + the short-TTL cache
  (decision 5).

## Files to change

- `backend/internal/config/config.go` -- `MarketDataSource` field, env
  var `MARKET_DATA_SOURCE`, default `"vci"`.
- `backend/cmd/api/main.go` -- wire `LiveProvider(VCIProvider, MockProvider)`
  when `MarketDataSource == "vci"`, else the existing bare
  `MockProvider`, based on the config value.
- `docker-compose.yml` -- pass `MARKET_DATA_SOURCE` through to the
  backend service (defaults to `vci` if unset, matching decision 6).

## Verification plan

1. `go build ./... && go vet ./...` (Docker, no local Go).
2. Curl `GET /api/v1/market/bars?symbol=FPT&resolution=1D&from=...&to=...`
   against the real running backend and confirm real, current prices
   come back (cross-check against a fresh direct `curl` to VCI's own
   endpoint for the same window, same technique already used to verify
   the endpoint above).
3. Curl `GET /api/v1/market/indices` and confirm VN-Index/VN30/HNX-
   Index/UPCOM-Index all return real, current values via the index
   symbol mapping.
4. Force a failure path: point `MARKET_DATA_SOURCE` at an unreachable
   host or break the adapter's URL temporarily, confirm `GetBars` falls
   back to the deterministic mock generator instead of the endpoint
   erroring -- then restore it and re-confirm real data returns. This
   is the one behavior that can't be verified by reading the code; it
   has to actually be exercised.
5. Browser-pane check: `/stocks` (indices, heatmap, movers) and
   `/stocks/FPT` (chart, price header, bands) all show real, current
   numbers, cross-checked by eye against a fresh direct request to
   VCI's own endpoint for the same symbol/window.
6. Full Docker rebuild + curl, matching this repo's standing habit.
7. Record the outcome in RESUME.md, replacing the top-of-file priority
   note with what was actually built and verified.

## Background: why VCI, not TradingView's data API

The user's first instruction here was to use "provider tradingview api as already plan." Researched before writing any code (not guessed):

The user's explicit instruction: **"next time we take data from provider
tradingview api as already plan"**, clarified in-session to mean option
(2) below -- they want TradingView itself to be the actual source of
market data, not just the charting widget. Researched via WebSearch/
WebFetch in this session (not guessed); findings below.

**Research findings (2026-09-23):**

- TradingView does not sell a self-serve "market data API" the way a
  licensed data vendor (SSI, VNDIRECT, TCBS, etc.) does. Their three
  real developer surfaces are: the Charting Library (free, self-hosted
  chart widget -- you supply the data), the Datafeed API (the spec for
  piping data INTO that widget from your own source), and the Broker
  REST API (for brokerages applying to let TradingView's own users trade
  through them -- not a way to pull data out).
- TradingView's own platform DOES cover Vietnamese exchanges -- HOSE and
  HNX (which UPCoM listings fall under) are both listed on
  https://www.tradingview.com/data-coverage/. So the data exists on
  their platform; the question is only whether/how it can be licensed
  out to a third-party app like this one.
- That page has no self-serve licensing/API-access flow for third-party
  applications -- it only shows individual-user subscription tiers
  (delayed / non-professional real-time / professional real-time). The
  only stated path is **contacting TradingView's sales/partnerships team
  directly** -- a business conversation the user has to initiate, with
  unknown pricing/terms, not something this session can request or
  estimate.
- Search results also surfaced third-party services (e.g. a site
  branded "TradingView Data API" with $0-$80/mo tiers, and an
  unofficial GitHub scraper hitting TradingView's undocumented internal
  endpoints) claiming to resell/scrape TradingView's data. **These are
  explicitly out of scope for this project** -- api-spec.md already
  rules out scraped data, and neither is an official, licensed
  TradingView product.

**Bottom line at the time this was written: there was no code to write
yet** -- the only real next action was the user contacting TradingView's
sales/partnerships team. Separately, still true and unresolved: the
self-hosted Charting Library (the UI-only path from charting-library-
integration.md, distinct from this data-source question) still needs
TradingView's GitHub-gated access approval, which the user has not
requested.

**Resolved below, same day:** rather than wait on that business
conversation, the user redirected to a different, real, free-to-access
Vietnamese data source found via further research (VCI). See the next
entry.

**Resolved same day**: rather than wait on that business conversation, the user redirected to VCI (the decisions below).

## Verification (done)

Resolution of the TradingView note above -- the user chose not to wait
on TradingView's sales/partnerships process and redirected to a
different real data source instead.

**User's explicit decision**, verbatim: "go with this way" (accepting
the unofficial/undocumented-endpoint category `api-spec.md` otherwise
rules out -- an informed, deliberate override, not something this
session decided unilaterally) "... the module is 'Inversion Principle'
mean work as adapt easy to change later." Full research trail and every
decision is in phase-vci-market-data.md; this entry is the outcome.

`market.MarketDataProvider` now has a real, live adapter --
`VCIProvider` (backend/internal/market/vciprovider.go) -- against
Vietcap Securities (VCI)'s own trading-platform API
(`https://trading.vietcap.com.vn/api/chart/OHLCChart/gap-chart`, POST,
no API key -- the same endpoint their own web app calls, not an
officially published third-party developer API; `vnstock`'s own README
says as much: "licenses the software, not the source data"). Request/
response shape was verified with real `curl` calls before writing any
Go code, not assumed from reading the (cloned, inspected) `vnstock`
Python source.

Because every price-dependent feature in this backend already reads
through the single `MarketDataProvider` port (CLAUDE.md's dependency-
inversion rule, applied consistently since the backend was first
scaffolded), swapping this one adapter made all of the following real
simultaneously, with zero changes to `symbol`, `screener`, `backtest`,
or any handler: `symbol.Detail`'s live price/change/reference/ceiling/
floor, `screener`'s heatmap/movers, `market.GetIndicator`'s SMA/EMA (and
RSI on the separate Phase D branch), backtests, and `market.GetIndex`
(VN-Index/VN30/HNX-Index/UPCOM-Index, swapped explicitly). The **order
book stays synthetic** (VCI's real depth data wasn't confirmed in this
research pass -- flagged, not silently inconsistent) and **symbol
static fundamentals stay hand-seeded mock** (company name, sector,
marketCap, P/E, P/B, ROE, EPS, dividendYield -- a separate adapter swap,
out of scope here).

Built as a decorator, not a bare swap, per the user's explicit
Inversion-Principle instruction: `LiveProvider`
(backend/internal/market/liveprovider.go) tries `VCIProvider` first,
falls back per-call to the existing `MockProvider` on any error or empty
result (a real risk for an undocumented upstream that owes this app
nothing), and caches successful results for 5 seconds both for
performance and so this app isn't hammering someone else's unofficial
endpoint on every page load. `LiveProvider` is itself just another
`MarketDataProvider`, so `main.go` still wires exactly one implementation
into `market.NewService` -- swapping to a licensed vendor later (e.g.
SSI FastConnect, already researched as a real, likely-free alternative
if the user follows up on that registration) means writing one new file
and changing one line in `main.go`, not touching callers.

Config: `MARKET_DATA_SOURCE` env var (`config.Config.MarketDataSource`),
default `"vci"` (the user asked for this live now, not opt-in-later);
`"mock"` still available (e.g. for a fully offline demo) by construction
of the port. `docker-compose.yml` sets it explicitly to `vci`.

Verified, not just written:

- `go build ./... && go vet ./...` clean (Docker, no local Go).
- Curled the real backend's `GET /market/bars?symbol=FPT` and
  `GET /market/indices` directly and confirmed real, current values
  (FPT closing at 66,600 VND on 2026-09-22, VN-Index at 1,816.93) --
  matching independently-verified direct calls to VCI's own endpoint.
- **Actually exercised the fallback path**, not just read the code:
  temporarily pointed `VCIProvider`'s base URL at an unreachable host,
  rebuilt the Docker image, confirmed `GET /market/bars` still returned
  a full, valid response (11 bars, FPT closing at 51,751.75 -- the
  exact known mock-generator value, confirming this really was the
  fallback and not a coincidence) with a clear "falling back to mock
  data" log line instead of an error or empty response. Reverted,
  rebuilt again, confirmed real data returned once more.
- Browser-pane check at 1440x960 on `/stocks`: all four index cards,
  the sector heatmap, and both movers tables show real, current VCI
  data (VN-Index 1.816,93, VN30 1.965,36, HNX-Index 276,93, UPCOM-Index
  126,56, real per-symbol change percentages and prices) -- confirming
  the dependency-inversion payoff described above actually happened in
  the running app, not just in theory.
- Full Docker image rebuild (both services) + curl: all 6 routes return
  HTTP 200, real VN-Index value present in the served `/stocks` HTML.
