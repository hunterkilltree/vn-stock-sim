# Phase D — planning and analysis

Scope per FULL-APP-PLAN.md section 6, corrected per the design-alignment
decision: rebuild `/stocks/[symbol]` (the Detail/ticker screen) reading
`design/screens/Detail.dc.html` as the literal spec. This is the
largest phase so far -- it requires two real backend additions (RSI,
MACD) and a full chart-rendering rewrite, not just new frontend
components over existing endpoints.

## Reading design/screens/Detail.dc.html

Read in full (488 lines). Key findings:

1. **Charts are inline SVG, not a chart library -- this is an explicit,
   already-known rule the current page violates.**
   design/DESIGN-SYSTEM.md section 5: "All charts are inline SVG
   generated from real computed series -- no image assets, no chart
   library." The current `/stocks/[symbol]` page renders its chart with
   `StockChart.tsx`, which wraps the `lightweight-charts` npm package.
   Detail.dc.html's own script block contains the exact candle/SMA/RSI/
   MACD geometry math (`py()` price-to-pixel scale, `poly()` polyline
   builder, Wilder RSI, EMA-based MACD) -- design/README.md explicitly
   says this math "is real and can be lifted directly." This phase
   builds a new SVG chart component porting that math against real
   backend data, used only on this page; `StockChart.tsx` (still used by
   the home-page hero) is untouched.
2. **RSI(14) and MACD(12,26,9) need real backend support that doesn't
   exist yet.** `market.Service.GetIndicator` only implements `sma`/
   `ema` (RESUME.md's own "Not built yet" list names RSI/MACD/Bollinger/
   VWAP). Both were always in V1's original scope
   (vn-stock-sim-summary.md: "candlestick chart with basic indicators
   (SMA/EMA/VWAP/RSI/MACD/Bollinger)") -- this is finishing already-
   planned V1 work the screen cannot be built faithfully without, not
   scope creep. Bollinger/VWAP are NOT shown on this screen, so they
   stay out of scope for this phase.
3. **The order ticket ("Dat lenh giay") is a real, functioning form**,
   not a static mockup panel: Mua/Ban toggle, LO/MP/ATC/Stop order type,
   price/quantity inputs, quick-fill lot buttons (25/50/75/Toi da),
   order value + 0.15% fee + remaining buying power, and a submit
   button. Every field maps onto `POST /api/v1/orders` (already built,
   Phase B) except the lot quick-fill math, which is new.
4. **The order book ("Buoc gia") is exactly Phase B's `GetOrderBook`**,
   3 levels each side (not 6 -- the design shows fewer rows than the
   backend returns).
5. **The fundamentals grid needs two fields `symbol.Detail` doesn't
   have:** ROE and 20-day average volume ("Von hoa/P/E/P/B/ROE/EPS 4
   quy/KL TB 20 phien"). See Decision 3.
6. **The icon rail's item list differs from Main's sidebar list** --
   7 items (Tong quan, Bieu do va chi bao, Bo loc co phieu, Xay chien
   luoc, Che do Replay, Giao dich giay, Tro ly Quant), missing 3 of
   Main's 10 (Ban do nhiet, Kiem thu lich su, So giao dich) and in a
   different order. See Decision 1.
7. **Timeframe pills** (1 ngay/1 tuan/1 thang/3 thang/1 nam/5 nam,
   default "1 thang" active) are real controls in the design, not
   decoration.
8. **"Them chi bao" (add indicator) and "Replay ma nay" header buttons**
   have no real destination yet -- Them chi bao has no indicator-picker
   UI designed anywhere, Replay ma nay points at `/replay` (Phase F, not
   built). Both render disabled, same treatment as Phase C's promo
   cards.
9. The design has no AI Insight panel at all -- this repo's existing
   rule-based Insight feature predates the design system. See Decision 4.

## Decisions

1. **Keep one shared nav list (navItems.ts), do not fork a second one
   for the rail.** The 7-vs-10-item, different-order discrepancy between
   Main's sidebar and Detail's rail reads as an artifact of two
   independently-drawn artboards (design/README.md describes each screen
   as self-contained), not a deliberate information-architecture
   decision -- nothing in design/DESIGN-SYSTEM.md documents a rail-
   specific subset. Maintaining two divergent nav configurations for a
   dubious, undocumented benefit would make navigation unpredictable
   for a user moving between screens. RailNav keeps rendering the same
   10-item list SidebarNav uses; this is a flagged deviation, not a
   silent one.
2. **RSI: Wilder smoothing, period 14, added to `GetIndicator`'s
   existing switch** (`sma`/`ema`/`rsi`), reusing the single-value
   `IndicatorPoint` shape -- no API shape change needed.
   **MACD: a new `GetMACD` method + `GET /market/macd` endpoint**,
   because MACD needs three series (macd/signal/histogram) per point,
   which doesn't fit `IndicatorPoint`. `market/types.go` already defines
   `IndicatorMultiPoint{Time, Values map[string]float64}` -- currently
   unused anywhere in the codebase, clearly anticipating exactly this.
   `Values` keys: `"macd"`, `"signal"`, `"histogram"`.
3. **`symbol.Detail` gains `ROE`** (a hand-seeded plausible value per
   mock symbol, same treatment as the existing hand-seeded MarketCap/
   PERatio/PBRatio/EPS/DividendYield -- not a new category of
   fabrication, matching existing practice). **20-day average volume is
   NOT a new backend field** -- it's computed in the page itself from
   the `Bar[]` already being fetched for the chart (`avg of the last 20
   bars' volume`), no new endpoint needed.
4. **The existing AI Insight panel stays, rendered below the design's
   own 1440x960 frame**, not deleted. The design not drawing a feature
   this repo already built and verified isn't grounds to remove it --
   "follow the design" governs what gets built to match the spec, not a
   license to delete already-working, already-honest functionality the
   spec is simply silent on. Clearly separated so the primary screen
   still matches Detail.dc.html exactly above the fold.
5. **Order submission is a real Server Action** (`orderActions.ts`,
   same httpOnly-cookie pattern as `authActions.ts` -- the token never
   reaches client JS), not a client-side fetch. **Guest state**: per
   design/DESIGN-SYSTEM.md section 9 ("the sign-in prompt fires at the
   first paper order") and design/SCREENS.md's own "Also undesigned"
   list ("a modal over the order ticket is the natural shape" -- named
   as NOT YET DESIGNED), a guest sees the order ticket's fields but the
   submit area is replaced with an inline "Dang nhap de dat lenh" prompt
   + link to `/login`, not a fabricated modal this session would have to
   invent un-designed UI for.
6. **Timeframe pills are real, URL-driven navigation** (Next.js
   `searchParams`, a Server Component re-fetching on each pill click),
   not client-side state -- consistent with this repo's Server-
   Component-first pattern used everywhere else. Mapping: 1 ngay ->
   resolution 5/1 day window, 1 tuan -> 60/7 days, 1 thang (default) ->
   1D/30 days, 3 thang -> 1D/90 days, 1 nam -> 1D/365 days, 5 nam ->
   1D/1825 days.
7. **Order book: first 3 levels each side** of Phase B's 6-level
   `GetOrderBook` response, matching the design's 3-row-each-side table
   exactly rather than changing the backend's level count (Detail is
   not the only consumer of that endpoint going forward -- Crypto-Detail
   in Phase I wants 8 levels, so the backend stays generic and each
   screen slices what it needs).

## Files to add

- `backend/internal/market/rsi.go` -- Wilder RSI, wired into
  `GetIndicator`'s switch.
- `backend/internal/market/macd.go` -- `GetMACD`, `IndicatorMultiPoint`
  builder; `handler.go` gains `GET /market/macd`.
- `frontend/src/components/DetailChart.tsx` -- the full inline-SVG
  candlestick + SMA20/50 + volume + RSI + MACD panel, math ported from
  Detail.dc.html's script block, fed real Bar/IndicatorPoint/
  IndicatorMultiPoint data.
- `frontend/src/components/OrderTicket.tsx` (Client Component) +
  `frontend/src/lib/orderActions.ts` (Server Action).
- `frontend/src/components/OrderBookPanel.tsx`.
- `frontend/src/components/FundamentalsGrid.tsx`.
- `frontend/src/components/PriceBandChips.tsx` (Tran/Tham chieu/San).
- `frontend/src/components/TimeframePills.tsx`.

## Files to change

- `backend/internal/market/service.go`, `handler.go` -- wire rsi/macd.
- `backend/internal/symbol/types.go`, `provider.go` -- add `ROE`.
- `frontend/src/lib/api.ts` -- `getMACD`, `IndicatorMultiPoint` type,
  `SymbolDetail.roe`.
- `frontend/src/app/stocks/[symbol]/page.tsx` -- full rebuild against
  the design, `searchParams`-driven timeframe, all new components
  assembled, existing AIInsightCard kept below the fold.
- `backend/postman/vn-stock-sim.postman_collection.json` -- add
  `Get MACD` request under Market, matching the new endpoint (the
  collection should stay a complete, accurate mirror of the real API
  per the earlier explicit request to document it).

## Verification plan

1. `go build ./... && go vet ./...` (Docker); curl `/market/indicators?
   indicator=rsi` and `/market/macd`, sanity-check values are in [0,100]
   for RSI and macd/signal/histogram are numeric.
2. `npx tsc --noEmit && npx eslint . && npm run build`.
3. Browser-pane check at 1440px: chart renders candles/SMA/volume/RSI/
   MACD as real SVG (inspect the DOM, not just a screenshot, to confirm
   no canvas/lightweight-charts element is present); switch timeframe
   pills and confirm the chart actually re-fetches a different range;
   place a real market order as a logged-in test user and confirm the
   portfolio position appears afterward; confirm the guest state shows
   the sign-in prompt instead of a submit button.
4. Full Docker rebuild + curl.
5. Record the outcome in RESUME.md.

## Verification (done)

- **Charts are now real inline SVG, not a chart library.**
  design/DESIGN-SYSTEM.md section 5 requires this; the old page used
  `StockChart.tsx` (wraps `lightweight-charts`). New
  `frontend/src/components/DetailChart.tsx` ports the design's own
  candle/SMA/RSI/MACD pixel geometry (`py()` scale, `poly()` builder,
  volume/RSI/MACD panel math) directly from Detail.dc.html's script
  block, fed real backend data. Verified in the browser pane via
  `document.querySelectorAll('canvas').length === 0` and 18 real `<svg>`
  elements present -- not just a visual screenshot check.
  `StockChart.tsx` itself is untouched (still used by the home-page
  hero).
- **Backend gained real RSI and MACD**, finishing already-planned V1
  scope (RESUME.md's own "Not built yet" list, vn-stock-sim-summary.md's
  original indicator list) that this screen could not be built
  faithfully without:
  - `market` package: `rsi()` (Wilder smoothing, ported from the
    design's `rsiCalc`), wired into `GetIndicator`'s existing `sma`/
    `ema` switch as a third case -- no API shape change.
  - `market`: `GetMACD` + `GET /api/v1/market/macd` (new endpoint,
    finally using `IndicatorMultiPoint`, which had been defined in
    types.go since Phase B but never used by anything) -- MACD needs
    three series per point (macd/signal/histogram), which doesn't fit
    the single-value `IndicatorPoint` shape.
  - `symbol.Detail` gained `ROE` (hand-seeded per mock symbol, same
    treatment as the existing MarketCap/PERatio/PBRatio/EPS/
    DividendYield -- FPT's value, 27.9%, deliberately matches the
    design's own FPT sample exactly).
- **Real order ticket**, not a static panel: new `OrderTicket.tsx`
  (Client Component: Mua/Ban toggle, LO/MP/ATC/Stop order type, price/
  qty inputs, 25/50/75/Toi da lot quick-fill computed from real buying
  power, live fee calc) + new `orderActions.ts` Server Action (same
  httpOnly-cookie pattern as `authActions.ts`, posting to the existing
  `POST /api/v1/orders`). Guest state (no session) replaces the submit
  area with a sign-in prompt instead of inventing a modal design/
  SCREENS.md explicitly lists as not-yet-designed.
- New `OrderBookPanel.tsx` (3 levels each side of Phase B's 6-level
  `GetOrderBook`, matching the design's row count without changing the
  backend's), `FundamentalsGrid.tsx` (20-day average volume computed
  from the already-fetched bars, not a new endpoint), `PriceBandChips.tsx`,
  `TimeframePills.tsx` (real `searchParams`-driven navigation, not
  client state -- 1 ngay/1 tuan/1 thang/3 thang/1 nam/5 nam each mapping
  to a real resolution+range).
- New `lib/format.ts` helpers: `formatThousandsVN` (the Detail screen
  quotes every price in thousands of VND, matching its own "Gia
  (nghin d)" label -- this app's backend returns raw VND everywhere) and
  `formatMarketCapVN`.
- Existing AI Insight panel kept, rendered below the design's own
  frame rather than deleted -- the design not drawing a feature this
  repo already built isn't grounds to remove it.
- Deliberate, flagged deviation: RailNav keeps the same shared 10-item
  nav list SidebarNav uses, rather than forking a second list to match
  Detail.dc.html's own smaller 7-item rail -- that discrepancy reads as
  an artifact of two independently-drawn artboards, not documented
  IA intent anywhere in design/DESIGN-SYSTEM.md.

Hit and fixed a real bug during verification, not just by reading the
code: the MACD panel initially rendered as a flat line pinned near the
baseline. Root cause was a unit mismatch -- `macdAbs` (the value used to
scale the y-axis) was computed from the backend's raw-VND-scale MACD
values, while the polylines/histogram it scaled had already been
converted to thousands-VND scale, so real values (~4) were divided by a
~1000x-too-large denominator (~4000) and collapsed near zero. Fixed by
converting to thousands before computing `macdAbs` too; re-verified in
the browser pane that both the MACD and signal lines now visibly
diverge and the histogram bars have real height.

Verified end to end, not just written: `go build ./... && go vet ./...`
clean (Docker); curled `/market/indicators?indicator=rsi` and
`/market/macd` directly and sanity-checked real values (RSI in [0,100],
MACD/signal/histogram numeric and trending). `npx tsc --noEmit`,
`npx eslint .`, `npm run build` all clean (hit and fixed one
react-hooks/purity violation the same way this repo already fixed it
once before -- `Date.now()` pulled out of the component body into a
helper function). Browser-pane check at 1440x960: switched timeframe
pills and confirmed the chart actually re-fetched a different candle
count/range (30d vs 90d, both real); registered a real test account,
confirmed the order ticket switches from the guest sign-in prompt to
real buying power once logged in, and placed a real limit order through
the actual UI -- confirmed via the dev server's own request log that
`placeOrderAction` fired and posted to the backend, and the ticket
displayed "Lenh da duoc dat va dang cho khop" (order placed, pending),
matching a queued (non-market) order's real backend status. Also
updated backend/postman/vn-stock-sim.postman_collection.json with the
new RSI/MACD requests, keeping the earlier explicit "document the API"
deliverable accurate. Full Docker rebuild + curl: all 7 routes return
200, the Detail screen's real content (order ticket, order book,
fundamentals, RSI/MACD panel labels) is present in the served HTML, and
RSI/MACD/ROE all return real data directly from the backend container.

Branch: phase-d-detail-screen (off master, after PR #18 merged).
