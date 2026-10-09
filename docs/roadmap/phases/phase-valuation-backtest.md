# Phase: Stage 2 — fundamentals history, valuation backtests, H2 factors

Status: **planned, not built.** Follows stage 1 (phase-holding-horizon.md
H1 + phase-valuation.md V-1, built 2026-10-09). This doc is the plan for
what those two docs left open: H2 (growth/debt factors) and V-2
(backtesting the buy-price rules against lump sum and DCA), plus the
leftover V-1 item (fair-value line on the chart).

## Goal

1. Let the user **test** each buy-price rule (Graham, Lynch, Rule #1,
   Weiss, P/E band, buy-the-dip) against **lump sum on day 1** and
   **monthly DCA**, with the same capital, real fees and 100-share lots.
   Show **% of time in cash** and **days to first fill**, so a rule that
   sat out a rally visibly loses.
2. Turn on the long-horizon factors that wait on data: EPS growth,
   debt/equity, ROE consistency. Also unlock the growth- and
   history-based valuation methods on the detail page.
3. Draw the fair value / buy price as lines on the Detail chart.

## Branches: one per slice, named for what's in it

Branch names say **what the branch contains**, not a random session
name. Format: `claude/<area>-<what>`, lowercase, hyphens. One branch =
one PR = one slice below, so each can be reviewed and merged on its own.

| Branch | Contains |
|---|---|
| `claude/stage1-rating-horizon-valuation` | Stage 1, already built (same commits as `claude/trusting-turing-k8v9v6`) |
| `claude/plan-stage2-valuation-backtest` | This planning doc |
| `claude/backtest-fees-lots-dca` | Slice 1 |
| `claude/fundamentals-history-port` | Slice 2 |
| `claude/valuation-backtest-rules` | Slice 3 |
| `claude/horizon-growth-debt-factors` | Slice 4 |
| `claude/chart-fair-value-line` | Slice 5 |

Each slice branches from the one it depends on (see "Order"), and each
phase doc and commit names its branch.

## Slices

### Slice 1 — realistic backtest engine + baselines (`claude/backtest-fees-lots-dca`)

Needs no new data, so it can start now. It also fixes RESUME.md's known
gap "backtests ignore fees" for the existing EMA/RSI rules.

- `tracker` (`internal/backtest/rule.go`) moves from all-in fractional
  shares to **whole 100-share lots**:
  `qty = floor(cash ÷ (price × (1 + FeeRate)) ÷ 100) × 100`, with
  `order.FeeRate` (0.15%) charged on both buys and sells. Partial buys
  are allowed (needed for DCA); leftover cash stays as cash.
- Baselines run on every backtest: **lump sum** (exists today as the
  "benchmark" line; becomes fee/lot-aware) and **monthly DCA** (capital
  ÷ months, bought on each month's first session).
- New result fields: `timeInCashPct`, `daysToFirstFill`, `feesPaid`,
  and per-baseline return, max drawdown and final capital.
- `/backtest` page: three equity lines (rule, lump sum, DCA) and a
  "Thời gian giữ tiền mặt" (time in cash) column. If the rule sat in
  cash ≥ 50% of the time while lump sum gained, show a callout.
- Optional realism (open question 3): 0.1% personal income tax on sell
  value, and T+2 settlement (can't sell shares bought less than 2
  sessions ago). Neither exists in `order` today. Whatever is decided
  goes into both `order` and `backtest`, so paper trading and backtests
  stay consistent.
- Tests: lot rounding, fees on both sides, DCA buying in each month,
  time in cash on a rule that never fires (= 100%), and existing EMA/RSI
  tests updated for fees (returns get slightly lower; that's correct).

### Slice 2 — point-in-time fundamentals (`claude/fundamentals-history-port`)

The data foundation for slices 3 and 4.

- New port in `symbol`: `FinancialsProvider.History(sym) []Period`.
  Each `Period` is a fiscal year or quarter with: EPS, BVPS, revenue,
  net income, equity, total debt, cash DPS, shares outstanding, and
  **`PublishedAt`**, the date the report was released. A backtest only
  sees periods whose `PublishedAt` ≤ the bar date, which removes
  look-ahead bias (phase-valuation.md "Checked against the code" item 6).
- `AsOf(sym, t)` derives point-in-time inputs: trailing EPS, BVPS, 3-year
  EPS CAGR (g, capped at 25%), ROE history, D/E, cash DPS, and from
  price history × trailing EPS the 5-year mean/σ P/E and historical high
  cash yield.
- **Macro port** for the VN 10-year government yield as a dated
  series. It replaces V-1's constant 4.43%.
- Adapter: decided by open question 1. Whatever the source, the mock
  adapter must **refuse to serve** when prices come from live VCI.
  Synthetic fundamentals against real prices produce fake signals on
  real companies.
- Stock-dividend adjustment: confirm whether VCI bars are adjusted (open
  question 2). If they're not, adjust in the adapter from the
  bonus-issue dates in the same data, or disable the dip and P/E-band
  rules.

### Slice 3 — valuation entry rules (`claude/valuation-backtest-rules`)

- New backtest rule type `valuation_entry` added to the `oneof` binding.
  Params (ints, per the existing format): `method` (enum index),
  `marginOfSafety` (percent), `exit` (0 = hold to end, 1 = sell at fair
  value, 2 = sell when the quality gate fails).
- Each session: `AsOf` → `valuation.Evaluate` → `FloorToTick` → skip if
  below that session's floor (an LO order outside the band can't be
  placed) → fill if the bar's low ≤ target, at the target price.
- Plus `dip_entry` (buy at 52-week high × (1 − d)) as the weak-evidence
  comparison phase-valuation.md lists.
- `/backtest` gets the method picker, and every run shows the three
  lines from slice 1.
- Detail page: Graham 1974, Lynch, Rule #1, Weiss and P/E band start
  returning numbers on the 1–3 năm tab, from `AsOf(now)`.

### Slice 4 — H2 long-horizon factors (`claude/horizon-growth-debt-factors`)

- `eps_growth` (3-year EPS CAGR, 0% → 15%), `health` (D/E 2 → 0.5;
  skipped for banks, where D/E isn't meaningful) and `consistency` (ROE σ
  over 3 years) get real values from slice 2's `AsOf(now)`. The
  "Thiếu dữ liệu" rows go away and `partial` turns false.
- Tests: weights stop re-normalising once the data exists, and banks
  skip `health`.

### Slice 5 — fair-value line (`claude/chart-fair-value-line`)

- `DetailChart` draws a dashed fair-value line and a shaded band from
  the buy price down, from the rating response's `valuation`. A
  checkbox in the indicator chips turns it on or off.
- Independent of slices 2–4; can go any time.

## Order

```
slice 1 (backtest-fees-lots-dca) ─┐
slice 2 (fundamentals-history-port) ─┴─> slice 3 (valuation-backtest-rules)
slice 2 ────────────────────────────────> slice 4 (horizon-growth-debt-factors)
slice 5 (chart-fair-value-line): any time
```

Slices 1 and 5 can start immediately. Slices 2–4 wait on open question
1.

## Verification (per slice)

- Backend: `go build/vet/test ./...`. Frontend: `typecheck`, `eslint`,
  `build`.
- Slice 1: a hand-computed backtest (known prices, 100-share lots,
  0.15% fees) matches to the đồng.
- Slice 2: a test that a period with `PublishedAt` after the bar date is
  invisible to `AsOf`.
- Slice 3: on a series built so the target is never reached, time in
  cash = 100% and the result equals the starting capital.
- UI slices: headless-Chromium screenshots at 1440px and 390px, as in
  stage 1.

## Open questions (for the user)

1. **Fundamentals source** (blocks slices 2–4):
   - (a) VCI's company-financials endpoints, the same unofficial API as
     prices. Needs checking outside this sandbox, which can't reach VCI.
   - (b) A licensed vendor (ties into Phase M's data-vendor question).
   - (c) Mock fundamentals, usable **only** with mock prices; valuation
     backtests disabled on live data.
   Recommendation: try (a) first, keep (c) for offline demos.
2. **Adjusted prices:** are VCI `gap-chart` bars adjusted for stock
   dividends and bonus shares? Needs a check outside the sandbox.
3. **Sell tax and T+2:** add the 0.1% sell tax and T+2 settlement to
   both paper trading and backtests (more realistic), or keep the
   current 0.15%-only, instant-settlement model?
4. **The old branch:** once `claude/stage1-rating-horizon-valuation` is
   confirmed, can `claude/trusting-turing-k8v9v6` be deleted, or should
   it stay?
