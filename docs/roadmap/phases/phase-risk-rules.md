# Phase: Risk-management rules (12 trading rules, coach / strict modes)

Status: **planned, not built.** Spec supplied by the user (2026-10-09):
12 widely used risk-management practices. Each has a default the user
can change per portfolio. This doc maps them onto the existing code and
splits the work into slices, one clearly named branch each (the
convention from phase-valuation-backtest.md).

These are risk-management heuristics, not guarantees of profit, and the
app labels them that way.

## The rules (summary of the spec)

| # | Rule | Default |
|---|---|---|
| 1 | Plan the exit before the entry: reason (setup), entry, stop, target on every trade; the **initial stop** is saved and never overwritten | stop required in every mode |
| 2 | Size from risk: qty = (equity × risk%) ÷ (entry − stop), rounded down to 100 shares; show "if stopped, you lose X VND (Y%)" | 1%, cap 2% (Elder 2% rule) |
| 3 | Reward-to-risk ≥ 2 | warn below 2 |
| 4 | Open risk + this month's losses ≤ 6% of equity, else no new trades this month (Elder 6% rule); "open risk" meter on Portfolio | 6% |
| 5 | Concentration and liquidity: max weight per stock and per sector; position ≤ % of 20-day average traded value | 20% / 40% / 5% |
| 6 | Stops only move up (or trail); lowering one is a logged violation; stop/target exits fire automatically | — |
| 7 | T+2.5 settlement: new shares are "pending settlement" and unsellable; show the worst-case gap loss (HOSE 2 × −7% ≈ 13.5%) and cap it | cap 3% of equity; settlement is a setting (T+0 is arriving on KRX) |
| 8 | Floor/ceiling: a stop can't fill when the stock is locked at the floor; warn when buying at or near the ceiling | — |
| 9 | Count costs: 0.1% sell tax plus 0.15% fee each side (~0.4% per round trip); show fees, taxes and yearly turnover; optional cap on new positions per week | tax on; weekly cap off |
| 10 | Journal in R: auto entry per trade (setup, entry, initial stop, target, exit, R); note and emotion tag; weekly review (expectancy, by setup, violations) | — |
| 11 | Circuit breaker: after 3 losses in a row, or a −2% day, lock new orders until the next session | 3 / 2% |
| 12 | Overfitting guard in backtests: hold out the most recent 30%, count setting variations tried, always plot VN-Index, flag < 30 trades | 30% / 30 trades |

**Enforcement modes, per portfolio:** *coach* warns and logs the
violation; *strict* blocks the order. Thresholds live in Settings →
"Tài khoản giấy" (already a placeholder in `settingsSections.ts`).

## Checked against the code (2026-10-09)

1. **Orders have no trade plan.** `order.createRequest` has
   `type ∈ market|limit|atc|stop|oco`, with `StopPrice` only as an OCO
   leg. Nothing records a protective stop, a target or a setup reason for
   a buy. Rules 1, 2, 3, 6 and 10 all need a new **trade plan** attached
   to the position, not just to an order.
2. **Lots are only enforced in the UI.** `OrderTicket.tsx` has
   `LOT_SIZE = 100`, but the backend accepts any whole number of shares.
3. **No sell tax, no settlement.** `order.FeeRate = 0.0015` per fill is
   the only cost. Shares are sellable immediately after a buy.
4. **Portfolios have no settings.** `portfolio.Portfolio` is id, name,
   market, kind and capital. Modes and thresholds need new per-portfolio
   storage: memory store plus a Postgres migration
   (`db/migrations/002_*.sql`), with the usual store contract test.
5. **The journal exists only as a stats grid.** `portfolio.Stats` builds
   the Portfolio page's KPI row and "journal" grid from closed trades.
   There are no per-trade entries, notes, emotions or R values.
6. **The Replay skill score is the one the spec criticises**
   (`replay/score.go`): entry/exit quality = distance from the best
   nearby price, and position sizing = low variation in position size.
   `Fill.StopSet` records whether a stop existed at each buy, but not
   whether it was set *before* entry or later lowered.
7. **The backtest benchmark is the stock's own buy-and-hold, not
   VN-Index.** There's no holdout, no count of variations tried, and no
   small-sample warning.
8. **The matcher** (`order/matcher.go`) already fills queued
   stop/limit/OCO orders against 5-minute bars in the background. That's
   the hook for automatic stop/target exits (rule 6) and for the
   floor-lock no-fill rule (rule 8).

## Design

1. **One rules engine: `internal/riskrules`.** Pure functions:
   `Check(OrderIntent, PortfolioState, Settings) []Finding`, where each
   `Finding{Rule, Severity, Message, Detail}`. `order.Service` calls it
   before booking. In coach mode findings are logged as violations and
   the order goes through. In strict mode an order with a blocking
   finding is rejected with `422 { code: "rule_violation", fields: [...] }`,
   which matches api-spec.md's validation-error shape. The ticket calls
   the same check through a dry-run endpoint, so warnings show *before*
   the user clicks Buy.
2. **Trade plan = new entity, owned by `order`.** `TradePlan{ID,
   PortfolioID, Symbol, Setup, PlannedEntry, InitialStop (immutable),
   CurrentStop, Trailing, Target, OpenedAt, ClosedAt, Exit, RMultiple}`.
   It's created with the first buy and closed when the position returns
   to zero. Adds and partial sells keep R based on the initial entry and
   initial stop (rule 10). Stop changes are appended to a
   `StopEvent{at, from, to}` log, and a lowered stop is a violation
   (rule 6).
3. **Automatic exits:** the matcher treats `CurrentStop` and `Target`
   like an OCO pair on the plan's open quantity. It fills only sellable
   (settled) shares (rule 7). A stop does **not** fill on a bar locked at
   the floor: open = close = floor, or the bar's low can't trade below the
   floor (rule 8).
4. **Quantity = the smallest of all limits.** The ticket shows which one
   binds: risk-based size (rule 2), cash, max stock weight (rule 5),
   liquidity cap (rule 5), and the settlement gap-loss cap (rule 7:
   position value × (1 − (1 − band)ⁿ) ≤ 3% equity, where n = settlement
   sessions). On HOSE at T+2 that's at most ~22% of equity. Then round
   down to 100 shares.
5. **Costs everywhere.** A shared cost model (fee 0.15% each side, sell
   tax 0.1%, settlement days) is used by `order`, `backtest` and `replay`,
   so paper trading, backtests and Replay can't disagree. This answers
   phase-valuation-backtest.md's open question 3: yes to tax and
   settlement. Crypto portfolios skip the tax, settlement and price-band
   rules.
6. **Positions without a stop** (anything opened before this ships):
   open risk counts them as "chưa có điểm cắt lỗ" (no stop yet), shown
   separately on the meter. Strict mode asks for a stop before allowing
   new buys.
7. **Monthly lock (rule 4):** month-to-date net realised loss (0 if
   positive) + open risk ≥ 6% of equity at month start → strict blocks
   new buys until the 1st; coach shows the red warning. The daily
   circuit breaker (rule 11) works the same way per session.
8. **Replay process score (replaces score.go's formula):** four
   components, equal weights, each 0–100:
   - **Stop discipline:** the stop was set before entry and never
     lowered.
   - **Sizing:** risk per trade ≤ limit, and the risk *percentage* was
     consistent (low variation in risk %, not in position size).
   - **Entry:** planned R:R ≥ 2 and the fill was close to the planned
     price.
   - **Exit:** the exit followed the plan (stop, target or trailing) and
     wasn't an override.

   "Distance from best price" stays as a separate statistic next to the
   score. Scores are computed from fills on every view, so **old
   sessions' scores will change**. The page notes the new formula.

## Slices — one branch each

| # | Branch | Rules | Contents |
|---|---|---|---|
| 1 | `claude/risk-trade-plan-sizing` | 1, 2, 3 + modes | `internal/riskrules`; per-portfolio `RiskSettings` (store + migration + contract test); `TradePlan` with immutable initial stop; backend 100-share lots; dry-run check endpoint; ticket gets setup, stop, target, risk-sized qty, "if stopped you lose X VND (Y%)", R:R; Settings → Tài khoản giấy page |
| 2 | `claude/risk-costs-settlement` | 7, 8, 9 | Shared cost model (tax, fees, settlement days); pending-settlement shares; gap-loss check and cap; floor-lock no-fill in matcher/backtest/replay; ceiling-chase warning; fees/taxes/turnover on Portfolio; optional weekly new-position cap |
| 3 | `claude/risk-portfolio-limits` | 4, 5, 11 | Open-risk meter on Portfolio; 6% monthly lock; stock/sector weight and liquidity checks on the ticket; daily circuit breaker |
| 4 | `claude/risk-stop-management` | 6 | Raise-only stops, trailing stops, stop-event log, automatic stop/target exits via the matcher |
| 5 | `claude/trade-journal-r-multiples` | 10 | Journal entries from trade plans; notes and emotion tags (bình tĩnh / FOMO / gỡ lỗ / sợ hãi); weekly review page with expectancy, results by setup, and violations |
| 6 | `claude/replay-process-score` | Replay score | Design item 8; plan capture in Replay (stop before entry, stop events) |
| 7 | `claude/backtest-holdout-overfitting` | 12 | 30% holdout locked until settings are frozen; count of variations tried; VN-Index on every backtest chart; < 30 trades warning |

**Order:** 1 → 2 → (3, 4 in either order) → 5 → 6. Slice 7 can go any
time, but slice 2's cost model should land before it **and** before
stage-2 slice 1 (`claude/backtest-fees-lots-dca`), so backtests get
the tax once. Slices 1–2 are where most of the work is; the later ones
build on the trade plan.

## Verification (per slice)

- `go build/vet/test ./...`. Unit tests in `riskrules` for every rule,
  in both modes, including the spec's worked numbers:
  - 2% risk → three losses in a row trip the 6% lock.
  - A 0.4% round trip.
  - HOSE T+2 worst case 1 − 0.93² = 13.51%.
  - Qty = equity × 1% ÷ (entry − stop), rounded down to 100.
- Store contract tests for `RiskSettings`, `TradePlan`, violations and
  journal entries, against memory and (with `TEST_DATABASE_URL`)
  Postgres.
- Matcher tests: a stop on a floor-locked bar doesn't fill; unsettled
  shares aren't sold.
- Frontend `typecheck`/`eslint`/`build`; headless-Chromium screenshots
  of the ticket in coach and strict modes, the open-risk meter and the
  journal at 1440px and 390px.

## Open questions (for the user)

1. **Default mode for existing portfolios:** coach (nothing that
   worked before starts failing), with strict offered when creating a
   new portfolio? (Recommended.)
2. **Rule 4 in coach mode:** red warning only, or also require a
   confirmation click?
3. **Settlement default:** T+2 (today's rule), with T+0 as an option
   for when KRX rolls it out?
4. **Replay:** apply the rules there too (coach mode, feeding the new
   score), or score only?
5. **Crypto portfolios:** apply rules 1–6 and 10–12 (sizing, stops,
   journal) but skip 7–9? (Recommended.)
