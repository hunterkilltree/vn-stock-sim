# Phase B — planning and analysis

Scope per FULL-APP-PLAN.md section 4: backend data model rework. Land
the multi-portfolio schema change (2.1) and give the frontend real
endpoints for everything Main.dc.html/Detail.dc.html need. No frontend
work in this phase (that's Phase C/D) -- backend only, verified with curl
against the actual Docker image, per this repo's standing habit (no
local Go install on this machine).

## Current state (read before writing code)

- `portfolio.MemoryStore` (`backend/internal/portfolio/store.go`) keys
  everything by `userID` directly: `accounts map[string]*account`, one
  cash balance + one position set per user, lazily opened with
  `StartingCash` (a package constant, 100,000,000) on first touch via
  `ensure(userID)`.
- `order.Service.Create` calls `s.ledger.ApplyFill(userID, sym, side,
  qty, price)` -- the `Ledger` port is keyed by `userID`, matching the
  store above.
- `order.createRequest` only accepts `type: market|limit` (binding
  `oneof=market limit`); no fee field on `Order`.
- `symbol.Detail` has `LastPrice/Change/ChangePercent` (already derived
  live from `market.Service.LatestClose` per the 2026-09-21 bug fix in
  RESUME.md) but no ceiling/reference/floor.
- `market.MockProvider` only implements `GetBars`; no indices, no order
  book. `market` package has zero import dependency on `symbol` (by
  design, per CLAUDE.md's layering rule) -- worth preserving.
- No `screener` package exists yet.
- `watchlist` is a flat per-user list of symbols
  (`byUse map[string][]entry]`), with no portfolio concept at all.
- `cmd/api/main.go` wires every service by hand; `symbolSvc` is the
  `QuotePort` for both `portfolio.Service` and `order.Service`.
- Response envelope convention (httpx.go, matches api-spec.md): lists as
  `{data, meta}`, single resources as the bare JSON object, errors as
  `{code, message}` / `{code, message, fields}`.

## Decisions for this phase

1. **Multi-portfolio store, but a lazy "default portfolio" keeps existing
   callers working.** `PortfolioStore` moves from `map[userID]*account`
   to `map[portfolioID]*account` plus a `Portfolio` registry
   (`id, userID, name, market, startingCapital, currency, createdAt`).
   Nothing in the frontend sends a `portfolioID` yet (that's Phase G's
   Account-Menu switcher) -- so `portfolio.Service` gains
   `DefaultPortfolioID(userID) string`, which lazily creates a
   "Danh muc chinh" stock portfolio (`StartingCash` VND) the first time
   it's asked for a given user, exactly mirroring the old `ensure()`
   lazy-open behavior but through the new multi-portfolio storage
   underneath. The existing `GET /portfolio` / `GET /portfolio/positions`
   routes keep working unchanged (resolve to the default portfolio
   internally); new `POST/GET /portfolios` (plural) routes expose the
   real multi-portfolio API for later phases.
2. **`order` gets portfolioID threading; `watchlist` does not.**
   FULL-APP-PLAN.md's Phase B step 2 named both, but on inspection
   watchlist is a plain per-user saved-symbols list with no per-portfolio
   meaning anywhere in the design canvas (no screen shows a
   per-portfolio watchlist) -- threading a portfolioID through it would
   be unmotivated complexity. Deviation from the written plan, flagged
   here rather than silently done: only `order.createRequest` gains an
   optional `portfolioId` field (falls back to
   `DefaultPortfolioID(userID)` when omitted), `watchlist` is untouched.
3. **Order book stays inside `market`, using a hardcoded 100 VND tick
   step, not a real symbol lookup.** The plan's own wording puts
   `GetOrderBook` on `market.MockProvider`. `market` has no import
   dependency on `symbol` today (CLAUDE.md's layering rule: services
   depend on small ports they define themselves, not on each other's
   packages) and every current mock symbol's `tickSize` happens to be
   100 -- so rather than break that layering for one field, the order
   book generator hardcodes a 100 VND step and documents it as a known
   simplification to revisit if a symbol with a different tick size is
   ever added.
4. **Ceiling/reference/floor bands, by exchange:** HOSE +-7%, HNX +-10%,
   UPCOM +-15% off the reference price (previous close, already available
   via `QuotePort.LatestClose`'s `yesterday` return value) -- these are
   the real HOSE/HNX/UPCOM daily price-band rules, not a design choice.
   Rounded to the nearest 100 VND tick.
5. **New order types (LO/MP/ATC/Stop) map onto the existing fill model
   conservatively.** `market` already means `MP` and `limit` already
   means `LO` (existing behavior unchanged). `atc` and `stop` are new
   `createRequest.Type` values that -- like `limit` today -- get accepted
   and stored as `"queued"` rather than actually filled; there is no real
   matching engine yet (this repo's existing, already-documented limit-
   order gap, see RESUME.md's Future Work). The 0.15% fee is computed and
   stored on `Order.Fee` at the moment of an actual fill; queued orders
   show `fee: 0` until a real matching engine later fills them.
6. **Screener package depends on `symbol.Service`'s existing public
   methods only** (`Search`, `Detail`) -- no new method added to
   `symbol`. `screener.Service.GetSectorHeatmap`/`GetTopMovers` call
   `Search("", "", 1, 100)` to enumerate the (small, fixture-sized)
   universe, then `Detail` each one for sector + live change%.

## Files to add

- `backend/internal/market/orderbook.go` -- `PriceLevel` type,
  `GetOrderBook(sym string, lastPrice float64) []PriceLevel` on
  `MockProvider`, deterministic in `(sym, lastPrice, time.Now().Unix())`.
- `backend/internal/market/index.go` -- `IndexSnapshot` type,
  `GetIndex(name string) IndexSnapshot` on `MockProvider` (VN-Index,
  VN30, HNX-Index, UPCOM-Index), deterministic in `(name, t)`, same
  sine-wave technique as `closeFor` so it's reproducible and never
  iterative.
- `backend/internal/screener/` (new package) -- `types.go` (`SectorGroup`,
  `TickerChange`), `service.go` (`GetSectorHeatmap`, `GetTopMovers`),
  `handler.go` (`GET /market/heatmap`, `GET /market/movers`).

## Files to change

- `backend/internal/portfolio/store.go` -- rework to
  `portfolios map[string]*Portfolio` + `accounts map[string]*account`
  both keyed by portfolio ID, `byUser map[string][]string` for listing;
  `Create`, `List`, `Get`, `DefaultFor` methods; `ApplyFill`/`Cash`/
  `Positions` re-keyed to `portfolioID`.
- `backend/internal/portfolio/types.go` -- add `Portfolio` struct.
- `backend/internal/portfolio/service.go` -- add `CreatePortfolio`,
  `ListPortfolios`, `GetPortfolio`, `DefaultPortfolioID`; re-key
  `Summary`/`Positions` to take a portfolio ID.
- `backend/internal/portfolio/handler.go` -- keep `GET /portfolio` and
  `GET /portfolio/positions` working via `DefaultPortfolioID`; add
  `POST /portfolios`, `GET /portfolios`, `GET /portfolios/:id`.
- `backend/internal/order/types.go` -- `Type` binding gains `atc`/`stop`;
  `Order` gains `Fee float64`; `createRequest` gains optional
  `PortfolioID string` (`json:"portfolioId"`).
- `backend/internal/order/service.go` -- `Ledger.ApplyFill` re-keyed to
  `portfolioID`; resolve `portfolioID` from the request or
  `DefaultPortfolioID`; compute `Fee` (0.15% of fill value) on actual
  fills only.
- `backend/internal/symbol/types.go` -- `Detail` gains
  `Ceiling/Reference/Floor float64`.
- `backend/internal/symbol/service.go` -- compute the three band fields
  from `yesterday` + the exchange band table whenever `QuotePort` has
  data.
- `backend/internal/market/handler.go` -- add `GET /market/indices`,
  `GET /market/orderbook`.
- `backend/cmd/api/main.go` -- wire `screener.NewService`, register its
  routes; no other wiring changes (portfolio/order constructors keep the
  same signature shape).

## Verification plan

All via curl against the actual Docker image (`docker compose build &&
docker compose up -d`), matching this repo's standing verification
habit:

1. `go build ./... && go vet ./...` clean, run inside a
   `golang:1.22-alpine` container (no local Go install).
2. Register a user, curl `GET /api/v1/portfolio` twice -- confirm the
   same default portfolio's cash balance both times (lazy-create is
   idempotent).
3. `POST /api/v1/portfolios` twice with different `name`/`market` for
   the same user, then `GET /api/v1/portfolios` -- confirm three
   portfolios now exist (the lazy default + the two new ones), each with
   an independent cash balance confirmed by placing a fill against two
   different portfolio IDs and checking they don't cross-contaminate.
4. `GET /api/v1/market/indices` twice a few seconds apart -- confirm
   identical values (same regression class as the 2026-09-21 price-
   consistency bug; verifying it's not reintroduced here).
5. `GET /api/v1/market/heatmap` -- confirm sector groups match the five
   existing mock sectors. `GET /api/v1/market/movers?direction=up` and
   `...=down` -- confirm correctly sorted.
6. `GET /api/v1/market/orderbook?symbol=VNM` -- confirm N levels each
   side, spaced by 100 VND, stable across two calls in the same second.
7. `GET /api/v1/symbols/VNM` -- confirm `ceiling > lastPrice > floor > 0`
   and `floor < reference < ceiling`, for all five mock symbols, and that
   the band width matches each symbol's exchange (HOSE vs HNX).
8. Place one order of each new `type` (`atc`, `stop`) -- confirm they
   come back `status: "queued"`, `fee: 0`, same as `limit` today; place a
   `market` order -- confirm `fee` equals 0.15% of `filledPrice *
   quantity`.
9. Record the outcome in RESUME.md, same format as every previous
   completed unit of work in this repo.

## Verification (done)

- `portfolio` package reworked from one account per user to real
  multi-portfolio storage: `Portfolio{id, userId, name, market,
  startingCapital, currency, createdAt}`, `MemoryStore` now keyed by
  portfolio ID with a `byUser` index for listing. `Service` gained
  `DefaultPortfolioID(userID)`, which lazily opens a "Danh muc chinh"
  stock portfolio (100,000,000 VND) the first time it's asked for a
  given user -- the old single-portfolio `ensure()` behavior, now
  sitting on top of the multi-portfolio store, so the existing
  `GET /portfolio` / `/portfolio/positions` routes work unchanged. New:
  `POST /api/v1/portfolios`, `GET /api/v1/portfolios`,
  `GET /api/v1/portfolios/:id`, plus `.../summary` and `.../positions`
  (added beyond the original phase-b.md file list, during verification,
  once it became clear there was no way to curl-confirm two portfolios'
  balances stay isolated without a per-ID summary endpoint).
- `order` gained `portfolioId` (optional on the request, resolved via
  `DefaultPortfolioID` when omitted) and two new order types, `atc` and
  `stop`, alongside the existing `market`/`limit` (LO/MP/ATC/Stop from
  Detail.dc.html's order ticket) -- like `limit` today, `atc`/`stop` are
  accepted and stored `"queued"`, not actually filled (no real matching
  engine yet, unchanged from before). `Order` gained `fee` -- 0.15% of
  notional value, computed only on an actual (`market`) fill.
  `watchlist` was deliberately NOT threaded with a portfolioId, a
  documented deviation from FULL-APP-PLAN.md's literal wording (see
  phase-b.md decision 2): the design canvas shows no per-portfolio
  watchlist anywhere, so it stayed user-scoped.
- `symbol.Detail` gained `reference`/`ceiling`/`floor` -- the real
  HOSE (+-7%)/HNX (+-10%)/UPCOM (+-15%) daily price-band rules, derived
  from the previous close (already available via the existing
  `QuotePort.LatestClose`) and rounded to the nearest 100 VND tick.
- `market` gained `GetIndex` (VN-Index/VN30/HNX-Index/UPCOM-Index, same
  deterministic sine-wave-plus-jitter technique as the existing bar
  generator -- a pure function of (name, t), reproducible across calls)
  behind `GET /api/v1/market/indices`, and `GetOrderBook` (6 levels each
  side, deterministic in (symbol, lastPrice, time.Now().Unix())) behind
  `GET /api/v1/market/orderbook?symbol=`. The order book uses a
  hardcoded 100 VND tick step rather than a real per-symbol lookup,
  since `market` has no import dependency on `symbol` by design
  (CLAUDE.md's layering rule) and every current mock symbol's tickSize
  happens to be 100 anyway -- documented simplification, see phase-b.md
  decision 3, revisit if a differently-ticked symbol is ever added.
- New `internal/screener` package (`GetSectorHeatmap`, `GetTopMovers`),
  depending only on `symbol.Service`'s existing public `Search`/`Detail`
  methods (no new method added to `symbol`) -- backs
  `GET /api/v1/market/heatmap` and `GET /api/v1/market/movers?
  direction=up|down`.

Verified, not just read -- all via curl against the real Docker image
(`docker compose build && docker compose up -d`, no local Go install on
this machine, same as every prior backend phase):

- `go build ./... && go vet ./...` clean inside a `golang:1.22-alpine`
  container.
- Registered a user, curled `GET /portfolio` twice -- identical default
  portfolio and cash balance both times (lazy-create is idempotent).
- Created two more portfolios via `POST /portfolios` (one stock, one
  crypto/USDT) -- `GET /portfolios` listed all three. Bought VNM against
  the default portfolio (`POST /orders` with `portfolioId`) and confirmed
  via the new `.../summary` endpoints that its cash balance dropped by
  the fill cost while a second, untouched portfolio's cash stayed exactly
  at its starting capital -- real isolation, not just two IDs sharing
  state.
- `GET /market/indices` called twice a few seconds apart returned byte-
  identical values for all four indices (deliberately re-checked the
  same regression class as the 2026-09-21 stock-price-consistency bug --
  not reintroduced here).
- `GET /market/heatmap` grouped all 5 mock symbols into their correct 4
  sectors with correct average change%; `GET /market/movers?
  direction=up` and `...=down` both correctly sorted.
- `GET /market/orderbook?symbol=VNM` returned 6 levels each side spaced
  exactly 100 VND apart, byte-identical across two calls moments apart.
- `GET /symbols/:symbol` for all 5 mock symbols: confirmed
  `floor < reference < ceiling`, and specifically checked the band math
  by hand for VNM (HOSE, reference 64599.52 -> ceiling 69100/floor
  60100, matching +-7% rounded to the nearest 100) and SHB (HNX,
  reference 55172.99 -> ceiling 60700/floor 49700, matching +-10%).
- Placed a `market` order and confirmed `fee` equals 0.15% of
  `filledPrice * quantity` (by hand: 63240.57 * 100 * 0.0015 = 9486.08,
  matching the existing truncating `round2` helper's convention); placed
  one `atc` and one `stop` order and confirmed both came back
  `status: "queued"`, `fee: 0`.

Branch: phase-b-backend-data-model (off master, after PR #13 merged).
Merged as PR #14.
