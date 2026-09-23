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
