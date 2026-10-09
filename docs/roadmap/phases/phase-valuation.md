# Phase: Valuation buy prices, and testing them against buy-and-hold

Status: **V-1 built (2026-10-09), together with holding-horizon H1; V-2 not started.** Companion to phase-holding-horizon.md (the
long-horizon tab is where a "buy zone" would show) and phase-rating.md.

## Goal

Answer "at what price is this stock worth buying?" with the well-known
fair-value rules, each turned into a **buy price = fair value × (1 −
margin of safety)**, snapped to the VN price grid. Then let the user
**backtest each rule against lump-sum buy-and-hold and monthly DCA** with
the same capital. The research is clear that waiting for a price usually
loses to investing at once (Vanguard 2012: lump sum beat DCA about two
times in three; Maggiulli: buying only at dips usually lost to plain DCA,
even with perfect timing). The comparison is the honest part of the
feature, not an afterthought.

## The rules

| Method | Fair value V | Buy when | Use for |
|---|---|---|---|
| Graham Number | √(22.5 × EPS × BVPS) | price ≤ V × (1 − MOS) | mature, asset-heavy, profitable |
| Graham formula (1974) | EPS × (8.5 + 2g) × 4.4 / Y | price ≤ V × (1 − MOS) | steady earners; Y = VN 10-year yield |
| Lynch fair value | EPS × g (PEG = 1) | price ≤ V | growers with g ≥ 5%; not cyclicals |
| Rule #1 sticker | EPS·(1+g)¹⁰·min(2g, avg hist P/E) / 1.15¹⁰ | price ≤ 0.5 × V | steady growers; very sensitive to g |
| Weiss yield | cash DPS / (0.9 × historical high yield) | price ≤ V | cash-dividend payers |
| P/E band (P/B for banks) | EPS × (mean P/E − 1σ), 5–7 years | price ≤ V | stable multiples only |
| Buy the dip | 52-week high × (1 − d) | price ≤ V | baseline for comparison only; weak evidence |
| Lump sum / monthly DCA | — | day 1 / every month | the baselines every rule is shown against |

g is expected EPS growth in percent (12 = 12%), except inside (1+g)¹⁰,
where it is a fraction. MOS is typically 0.25–0.5. Every method returns
"not applicable" rather than a number when its inputs don't fit (EPS ≤ 0,
g < 5 for Lynch, no cash dividend for Weiss). The UI shows *why* it
doesn't apply instead of hiding the row.

Labelled like AI Insight: "rule-based, not investment advice".

## Checked against the code (2026-10-09)

These are where the original proposal's assumptions about this repo
don't match the code, and what the plan does about each:

1. **Backtest rule format** (`internal/backtest/types.go`): `Rule{Type,
   Params map[string]int}`, and `Type` is a Gin `oneof=ema_crossover
   rsi_reversion` binding. Params are **ints**, so MOS is passed as a
   percentage (`marginOfSafety: 25`), and each new rule type must be added
   to the `oneof`.
2. **The backtest is all-in/all-out with fractional shares and no fees**
   (`rule.go` `tracker.buy`: `shares = cash / price`). Known gap in
   RESUME.md. The proposed sizing formula,
   `floor(cash ÷ (price × 1.0015) ÷ 100) × 100`, matches the real fee
   (`order.FeeRate = 0.0015`) and HOSE's 100-share lots. Adopting it fixes
   the "backtests ignore fees" gap for every rule, not just these.
3. **One benchmark exists, DCA doesn't.** `tracker.mark` already
   computes buy-and-hold from the first bar (that *is* lump sum). DCA
   needs the tracker to support partial buys, which it currently can't.
4. **Tick size is wrong today.** Every seeded symbol has `TickSize: 100`,
   and `symbol.roundToTick` rounds to the *nearest* tick. HOSE is
   tiered: 10 VND below 10,000; 50 VND from 10,000 to 49,950; 100 VND
   from 50,000. HNX and UPCOM use 100. So today's ceiling/floor for a
   HOSE stock under 50,000 is on the wrong grid. Fix once in a shared
   `TickFor(exchange, price)`. Ceiling rounds down and floor rounds up,
   so both stay inside the band. Use it for the band, the order ticket
   and buy targets (rounded **down**).
5. **Bars are a pure function of (symbol, t) only in the mock.** Live
   bars come from VCI's `gap-chart` endpoint. Whether those are adjusted
   for stock dividends and bonus issues is **unverified** (the sandbox
   can't reach VCI). This matters: on unadjusted prices every ex-date
   looks like a crash and falsely triggers dip/band rules.
6. **There are no historical fundamentals at all.** `symbol.Detail` is
   one hand-seeded snapshot (EPS, P/E, P/B, ROE, dividend yield). That's
   enough for *today's* fair value. It is not enough for a backtest:
   evaluating 2019 with 2026 EPS is look-ahead bias. A mock
   `GetFundamentalsAsOf(symbol, date)` would be internally consistent
   with **mock** prices, but paired with **live VCI** prices it would
   produce made-up buy signals on real stocks. So the backtest rules wait
   for a real point-in-time source (decision 3).

## Decisions

1. **`internal/valuation`: pure functions, no I/O.** `FairValue(method,
   Fundamentals, Params)`, `BuyPrice(...)`, `Applicable(...)` with a
   reason. One package feeds the Detail card, a screener field (price ÷
   Graham Number), a fair-value line on the chart, a Replay buy-zone
   overlay, and the conditions Quant generates.
2. **Margin of safety per method, not one global value.** The original
   sketch had a single `MarginOfSafety`, but Rule #1 is defined at 50%
   while Graham-style rules use 25–33%. Each method has its default MOS,
   and the user can override it per method.
3. **Ship in two stages, by data.**
   - **V-1, today's buy zone (current snapshot, no backtest).** Methods
     whose inputs exist now: Graham Number (BVPS = price ÷ P/B), plus
     Graham formula / Lynch / Rule #1 once g exists. g comes from
     phase-holding-horizon.md H2's seeded `Financials` (EPS CAGR), so
     both features share one data addition. Y = a single configured VN
     10-year yield, labelled with its date. Weiss and P/E bands need
     history, so they show "cần dữ liệu lịch sử" (needs historical data)
     in V-1.
   - **V-2, backtest entry rules.** Needs point-in-time fundamentals
     (EPS visible from the *report publication date*, not quarter-end),
     a VN 10-year yield **time series** (it was ~4.4% in Sept 2026 but in
     the high teens in mid-2008, a figure the user's note gives and I
     haven't checked; hard-coding today's yield would massively overvalue
     2008), and adjusted prices. Built only once a real source is chosen.
     Gated on open question 1.
4. **Backtest results show cash drag, not just return.** Every run is
   shown next to lump sum and monthly DCA, with the same capital, fees
   and lots, plus **% of time in cash** and **days to first fill**. A rule
   that sat in cash through a rally must look bad, because it was.
5. **Order mechanics in the backtest:** each session, compute BuyPrice →
   round down to tick → if below that session's floor, skip (an LO order
   outside the band can't be placed) → else LO at that price, filled if
   the bar's low ≤ target, sized by the lot/fee formula in item 2 above.
6. **Guard against value traps and misuse.** A quality gate (EPS > 0
   for several years and ROE above a threshold) applies before any
   method. Each method is matched to company type (table above; banks
   use P/B bands, not P/E). Where several apply, the default buy price is
   the **lowest** (most conservative), and the others are listed.
   g is capped (e.g. 25%) so one boom year can't produce an absurd
   value.

## Work breakdown

V-1:
1. `symbol`: `TickFor(exchange, price)` with tiered HOSE ticks; fix
   ceiling/floor rounding direction; tests. (A standalone bug fix, worth
   doing first.)
2. `valuation`: Graham Number, Graham formula, Lynch, Rule #1, Weiss
   and P/E band as pure functions with `Applicable` reasons. Per-method
   MOS. `RoundDownToTick` via `symbol.TickFor`. Table tests with
   hand-computed values, including the Rule #1 g-sensitivity case
   (g 15% → 10% cuts the sticker price by ~57%).
3. `GET /symbols/:symbol/valuation`: today's fair value / buy price per
   method, the conservative pick, and current price vs. buy zone.
4. Frontend: a "Vùng giá mua" (buy zone) section on the 1–3 năm tab
   (phase-holding-horizon.md), plus a dashed fair-value line on the
   Detail chart.

V-2 (after open question 1):
5. Backtest: lot/fee-aware sizing and partial buys in `tracker`; DCA and
   lump-sum baselines; % in cash and days to first fill.
6. `valuation_entry` rule type (`method`, `marginOfSafety`, exit =
   price ≥ fair value or quality gate fails).
7. Point-in-time fundamentals port + VN 10-year yield series; adjusted
   prices.

## Built (V-1) — what changed from the plan

- `backend/internal/valuation` (pure functions plus a table test of
  every method, including Rule #1's g sensitivity: 30,000 → 12,823 when
  g goes from 15% to 10%, a 57% cut).
- **Served inside `GET /symbols/:symbol/rating`** as `valuation`, not a
  separate endpoint. It renders on the same card (1–3 năm tab) and
  reuses the same fetch. The package stays standalone, so a screener or
  backtest can call it directly.
- Tick fix landed in `symbol`: `TickFor`, `FloorToTick` and `CeilToTick`;
  ceiling rounds down and floor rounds up; `Detail.TickSize` is now
  price-dependent on HOSE. **Not changed:** `market/orderbook.go` still
  steps the synthetic order book by 100 VND.
- Live today: **Graham Number** only (BVPS from the seed's price ÷ P/B).
  Graham 1974, Lynch and Rule #1 show "Thiếu dữ liệu tăng trưởng EPS"
  until growth data exists. Weiss and P/E band show "Cần lịch sử …".
  Quality gate: EPS > 0 and ROE ≥ 10%.
- Bond yield is the constant 4.43% (2026-09-09) from the user's note,
  shown with its date, and **not independently verified**.
- **Not done from V-1 step 4:** the dashed fair-value line on the Detail
  chart.

## Verification (V-1)

- `go test ./internal/valuation ./internal/symbol` (new tick tests:
  tier boundaries; band rounding stays inside the band).
- Rendered on `/stocks/VCB`: buy price 49,500 (Graham Number 66,031 ×
  0.75 = 49,523, rounded down to the 100 VND step), "Chưa tới vùng mua",
  and every other method listed with its reason.

## Open questions (for the user)

1. **Point-in-time fundamentals source** for V-2: a real vendor (needs a
   choice and probably a licence; ties into Phase M's data-vendor
   question), or accept mock-only backtests that are disabled on live
   VCI data?
2. **Adjusted prices:** can someone check, outside this sandbox, whether
   VCI's `gap-chart` bars are adjusted for stock dividends? If they
   aren't, the dip and band rules must stay off until they are.
3. **Order:** build V-1 together with holding-horizon H1/H2 (they share
   the g data and the 1–3 năm tab), or valuation first?
