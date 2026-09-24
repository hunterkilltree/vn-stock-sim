# Phase F — Replay Mode

Plan written against `design/screens/Replay.dc.html` as the literal spec,
per FULL-APP-PLAN.md section 8 and CLAUDE.md's instruction to write a
planning/analysis file before each lettered phase. This is the plan's own
"single feature with no existing backend surface at all" — a new
`internal/replay` package plus a new `/replay` frontend page.

## Design decisions (made here, before code, per FULL-APP-PLAN.md section 2.5)

1. **Historical range is a fixed, deterministic anchor.** Since
   `market.Service.GetBars` is already a pure function of
   `(symbol, resolution, from, to)` (real for the VCI-backed live
   provider, deterministic-synthetic for the mock fallback either way),
   "replaying 2021" just means requesting that date range — no new data
   source needed. Default anchor `2021-01-04` (matches the design's own
   caption exactly, "Phiên mô phỏng bắt đầu 04/01/2021", so a visual
   check against the mockup is a fair comparison), default 120 daily
   bars, both overridable in the start request.
2. **The full bar series is fetched once at session start and stored
   entirety in the session** (not re-fetched per `/advance`), even though
   only a prefix is ever returned to the client. This is what makes
   "cannot preview the future" a real backend guarantee rather than a
   client-side mask: the server holds candles 45-120 from bar 1, the
   client's request for candle 45 simply isn't in the response body until
   `/advance` says so, and the skill score at the end can honestly look at
   the full series (a real, already-known-server-side range) without the
   client ever having had access to it.
3. **A fill only ever happens at the currently-revealed bar's close.**
   The design's order ticket has no price field (only quantity + stop-
   loss) — Mua/Bán always execute at "now" in replay-time. This is a
   *stronger* guarantee than validating a client-supplied price against
   what's revealed (the mechanism FULL-APP-PLAN.md section 8 asks for):
   there is no price field for a client to spoof in the first place.
4. **Each Replay session gets its own dedicated portfolio**, created via
   the existing `portfolio.Service.CreatePortfolio` (Phase B/E), not the
   user's regular paper-trading portfolio. Considered and rejected:
   booking replay fills against the user's real default portfolio at
   *historical* prices while live fills book at *current* prices — this
   would blend a symbol's average cost across two unrelated time periods
   in one FIFO ledger (Phase E's `reconstructTrades`), producing a
   genuinely confusing, not just cosmetically odd, position. A dedicated
   portfolio per session sidesteps this entirely and is free: Phase E
   already built multi-portfolio support, `Service.Stats`/
   `Service.Positions`/`Service.ApplyFill` all just work unchanged against
   whatever portfolio ID the session hands them. Naming:
   `"Replay {symbol} {anchor date}"`.
5. **Replay fills are real `order.Order` records**, appended directly via
   the existing `*order.MemoryStore.Append` (a small new port,
   `replay.OrderLog`, satisfied by the store's existing public method —
   no new method needed on `order`), with `FilledAt`/`CreatedAt` set to
   the *simulated* historical bar date, not real wall-clock time. This
   matters for two reasons: (a) it's what makes the design's "Tự động ghi
   vào Sổ giao dịch" (auto-recorded to the trade journal) copy literally
   true — the session's dedicated portfolio's own `GET .../stats` (Phase
   E) sees these fills and computes real win rate/profit factor/drawdown
   over them; (b) using real wall-clock time instead would make every
   trade in a fast replay session look like a ~0-day hold, which is
   wrong — the simulated date is the actually meaningful one for holding-
   period math.
   **Correctness fix this required:** Phase E's `reconstructTrades`
   assumed a portfolio's orders arrive pre-sorted by fill time (true for
   real trading, since orders are appended in real-time order). Replay
   fills carry *historical* dates but get appended in real (later) wall-
   clock order, so that assumption breaks. Fixed by sorting fills by
   parsed `FilledAt` before FIFO-matching in `reconstructTrades` — a
   small, defensive, backward-compatible fix (real trading fills are
   already sorted, so this changes nothing for Phase E's existing
   behavior).
6. **Stop-loss is real, not decorative.** `PlaceOrder` accepts an optional
   `stopLoss` on a buy, recorded as the session's single active stop for
   that symbol (V1 Replay supports exactly one symbol per session, same
   as the design's one-chart-at-a-time scope). On each `/advance`, after
   revealing the new bar, if its low breaches the active stop, the
   session auto-sells the full held position at the stop price and
   records the fill with a note -- this is what gives the skill score's
   "stop-loss discipline" sub-score real data to grade (was a stop set at
   all, and was it the thing that actually closed the position, vs. never
   set or silently ignored).
7. **Skill score formula** (heuristic, explicitly labeled as such in the
   API response and the UI, same honesty convention as the existing rule-
   based AI Insight feature):
   - **Entry quality:** for each buy fill, the best (lowest) close in a
     ±5-bar window around its bar index (server has the full series, even
     the parts not yet revealed at fill time -- fair to use only *after*
     the session ends, which is when this is computed). Score per fill:
     `100 * (1 - |fillPrice - bestPrice| / bestPrice)`, clamped to
     [0,100]; averaged across all buy fills. No buy fills -> 100 (nothing
     to penalize).
   - **Exit quality:** same idea for sell fills against the best (highest)
     close in the same window.
   - **Stop-loss discipline:** `70% * (buys with a stop set / total buys)
     + 30% * (stops that actually triggered and closed the position /
     stops set)`. No buys -> 100.
   - **Position sizing:** 100 minus the coefficient of variation (stdev/
     mean, as a percentage, clamped) of each fill's notional value
     relative to the session's starting capital -- consistent sizing
     scores higher than wildly varying bet sizes. Fewer than 2 fills ->
     100 (nothing to compare).
   - **Overall** is the plain average of the four. This is a deliberate,
     documented simplification (the design shows an overall number
     alongside four sub-scores but never states its own formula) -- not
     claimed as a validated skill assessment anywhere in the API or UI
     copy.

## Backend (`backend/internal/replay`)

- `types.go` — `Session`, `Fill`, `StartRequest`, `OrderRequest`,
  `SessionView` (the one response shape reused by start/advance/order/get
  -- revealed bars + SMA20 aligned to them + fills + current stop + a
  cash/position snapshot for the order ticket's "Đang giữ" line),
  `EndResult` (KPIs via the same `portfolio.Stats` shape Phase E already
  returns, plus `SkillScore`).
- `store.go` — `MemoryStore`, same per-user-indexed pattern as every
  other V1 feature (`order`, `backtest`).
- `service.go` — `Start`/`Advance`/`PlaceOrder`/`End`/`Get`, plus the
  `BarsPort`/`PortfolioPort`/`OrderLog` interfaces `market.Service`/
  `portfolio.Service`/`*order.MemoryStore` already satisfy without any
  changes to those packages (`PortfolioPort` needs `CreatePortfolio`,
  `ApplyFill`, `Positions`, `Stats` -- all four already exist on
  `portfolio.Service` since Phase B/E).
- `score.go` — the four sub-score + overall formula above, kept in its
  own file since it's the one piece of real "made this number up, here's
  exactly how" logic worth reading end to end in isolation.
- `handler.go` — `POST /api/v1/replay/sessions`,
  `POST /api/v1/replay/sessions/:id/advance`,
  `POST /api/v1/replay/sessions/:id/orders`,
  `POST /api/v1/replay/sessions/:id/end`,
  `GET /api/v1/replay/sessions/:id`.
- `main.go`: `order.NewMemoryStore()` gets pulled into a named
  `orderStore` variable (was inlined) so both `order.NewService` and the
  new `replay.NewService` can share it.

## Frontend

- `frontend/src/app/replay/page.tsx` — a thin Server Component that
  reads the session user (redirects/prompts guests, same pattern as
  Portfolio) and renders a Client Component doing the actual work, since
  the whole screen is one continuous interactive session (advance/orders/
  keyboard shortcuts) rather than server-fetched-then-static.
- `frontend/src/components/ReplaySession.tsx` — the Client Component:
  candlestick chart (real inline SVG, following `DetailChart.tsx`'s
  established pixel-geometry approach rather than pulling in
  `lightweight-charts` for a use case that also needs a "future hidden"
  overlay and buy/sell markers baked into the same canvas), playback
  controls (step/auto-play/speed), order ticket (qty + stop-loss +
  Mua/Bán), session log table, and the results/skill-score panel that
  only appears once `/end` has been called. A start form (symbol picker +
  "Bắt đầu phiên") renders before any session exists.
- `frontend/src/lib/api.ts` gains the typed client functions for the 5
  endpoints above.

## Documented simplifications/deviations

- **One symbol per session**, matching the design's one-chart scope --
  no portfolio-wide replay across several tickers at once.
- **No short selling.** "Bán" only reduces/closes an existing long
  position (same constraint `portfolio.MemoryStore.ApplyFill` already
  enforces via `ErrInsufficientShares`), matching every other paper-
  trading order in this app.
- **The replay portfolio doesn't show up in the main Portfolio page's
  multi-portfolio switcher yet** -- that switcher UI is Phase G
  (Account-Menu.dc.html), not built yet. A finished Replay session's
  numbers are visible on the Replay results panel itself (which is what
  the design actually draws) and, once Phase G exists, through the
  regular Portfolio page like any other portfolio.
  **Revised in Phase G** (phase-g.md decision 4): Replay portfolios are
  now `kind: "replay"` and deliberately excluded from the switcher, since
  an active Replay portfolio would receive live Detail-ticket orders and
  mix price eras -- the backend rejects such orders with 409.
- **Skill score is a heuristic**, explicitly labeled as such in both the
  API response (`"source": "heuristic"` alongside the score) and the UI,
  never presented as a validated skill assessment.

## Bug found and fixed during browser verification

A real, not cosmetic, data-integrity bug: `SessionView.Result.NAV` was
initially computed by calling `portfolio.Service.Stats(portfolioID, ...)`
directly and reusing its `TotalEquity`/`TotalPnlPercent`. That method's
mark-to-market for an *open* position values it at **today's real, live
market quote** (via `portfolio.Service`'s `QuotePort`) -- exactly correct
for real paper trading (Phase E), but wrong here: it silently leaked
today's live 2026 HPG price into a session labeled "Phiên mô phỏng bắt
đầu 04/01/2021". Caught by actually watching a real browser session
(bought 500 HPG, and the KPI panel immediately showed a nonzero,
unexplained P&L with nothing else having happened) -- not visible from
reading the code, since `portfolio.Stats` is "correct" by its own
contract, just wrong for this caller. `TotalTrades`/`Wins`/`Losses`/
`ProfitFactor` were and remain safe to reuse from `Stats` (they come from
FIFO-matched *closed* trades using each fill's own historical price,
never a live quote) -- only the open-position mark-to-market was
contaminated. Fixed with a dedicated `replayAccounting` (score.go): walks
the session's own fills against its own revealed bar closes bar-by-bar,
the same accounting style `backtest/rule.go` already uses for a
simulated run over historical data, computing NAV/PnlPercent and a real
max-drawdown series with zero dependency on live market data. Re-verified
in the browser: buying 500 HPG at the exact price shown as "current" now
leaves NAV and P&L untouched (100,00 tr / 0,00%) until the price actually
moves within the replay's own revealed history.

## Verification

Per FULL-APP-PLAN.md section 8's own callout, this needed real browser
verification, not just curl -- done:

- `go build ./... && go vet ./...` clean; `gofmt -l` shows no new
  non-compliant files. `npx tsc --noEmit`, `npx eslint .`, `npm run build`
  all clean (route table lists `/replay` as dynamic, as expected -- it
  reads the session cookie).
- Backend, via curl: a full session (start -> place a buy with a
  stop-loss -> advance until the stop triggers -> end) returned correct
  fills, a triggered auto-exit with the right note, and a final score
  with `final: true`; separately confirmed error paths -- selling with no
  position returns 409 `insufficient shares`, acting on a completed
  session returns 409 `session_completed`, and `GET` after `End` still
  returns the same result.
- Full Postman collection (`npx newman run ...`, 5 new Replay requests
  added): 41 requests, 0 failures, both before and after the NAV fix
  below.
- Real browser (Playwright against this sandbox's pre-installed headless
  Chromium, driving the actual `next dev` + `go run` servers): registered
  a real account, started a real HPG session through the real start
  form, watched the real chart render (candles, future-hidden mask,
  volume, progress bar) matching the design closely, placed a real buy
  order with a stop-loss and watched it auto-trigger on the next real
  candle (confirmed via the session log showing "Tự động cắt lỗ", not
  just trusting the request succeeded), ran real auto-play for several
  ticks, and ended the session -- confirmed the skill panel's title and
  copy actually switch from "tạm tính" (provisional) to final, and that
  every mutating control (Nến tiếp, Phát tự động, Mua/Bán) becomes really
  disabled (checked the DOM `disabled` attribute directly, not just that
  the buttons look greyed out) once the session is completed. Caught and
  fixed one real bug this way (the NAV live-quote leak above) that curl
  alone had not surfaced, since the leak only shows up as an unexplained
  number next to an action that should have been a no-op.
