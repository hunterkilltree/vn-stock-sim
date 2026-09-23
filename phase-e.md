# Phase E — Rebuild Portfolio

Plan written against `design/screens/Portfolio.dc.html` as the literal spec,
per FULL-APP-PLAN.md section 7 and CLAUDE.md's instruction to write a
planning/analysis file before each lettered phase. Backend + frontend, no
database work (still in-memory, see RESUME.md future work).

## Scope

Portfolio.dc.html's 4-tab screen (Tổng quan/Vị thế/Lệnh chờ/Sổ giao dịch):
KPI row, equity curve chart, positions table, sector allocation, pending
orders (with a working cancel button), and a journal grid — all backed by
real backend computation over actual order/fill history, not the design's
hardcoded sample numbers (kpiRaw/holdings/allocRaw/pendRaw/jRaw).

## Backend additions (`backend/internal/portfolio`, `backend/internal/order`)

1. **Equity history.** `portfolio.MemoryStore` gained an `equity` index
   (`map[portfolioID][]EquityPoint`), seeded with one point at portfolio
   creation and appended to on every fill. This required moving the
   `Ledger` dependency `order.Service` calls from the bare
   `*portfolio.MemoryStore` to `*portfolio.Service`
   (`portfolioSvc.ApplyFill` now wraps `store.ApplyFill` with a real
   mark-to-market `Summary().TotalEquity` snapshot) — main.go wires
   `portfolioSvc`, not `portfolioStore`, as `order.NewService`'s ledger arg.
   `GET /api/v1/portfolios/:id/equity-history`.
2. **Sector allocation.** `Service.Allocation` groups `valuedPositions`'
   market value by `symbol.Detail.Sector` (already available via the
   existing `QuotePort`, no new dependency) plus a trailing "Tiền mặt"
   cash bucket. `GET /api/v1/portfolios/:id/allocation`.
3. **Stats (KPIs + journal).** `Service.Stats` FIFO-matches a portfolio's
   real filled buy/sell orders (oldest lot first) into closed trades,
   computing win rate, profit factor (0 when there are no losing trades to
   divide by — an honestly-undefined ratio, not a fabricated number), avg
   win/loss %, avg holding days, best/worst trade, and max drawdown (the
   largest peak-to-trough decline across the equity-history series).
   Needs order history, which `portfolio` doesn't otherwise depend on —
   solved with a `portfolio.OrdersPort` interface
   (`FilledOrders(userID, portfolioID) []OrderRecord`) that `order.Service`
   satisfies (`order` already imports `portfolio` for its Ledger error
   types, so this is not a new dependency direction), wired via
   `Service.SetOrdersPort` after both services exist in main.go (avoids an
   import cycle, since `order.NewService` itself needs a `*portfolio.Service`
   for the Ledger). `GET /api/v1/portfolios/:id/stats`.
4. **Positions gained `openSince`.** The same FIFO reconstruction (factored
   into `reconstructTrades`, shared by Stats and `valuedPositions`) also
   returns each symbol's still-open lots; the oldest one's fill time backs
   the design's "Nắm giữ" (days held) column.

## Found bug, fixed as part of this phase (not itself Phase E scope)

`OrderTicket.tsx` (Phase D) always collected a limit/stop trigger price
from the user, but `orderActions.ts`'s `placeOrderAction` never included it
in the POST body, and `order.Order`/`createRequest` had no `Price` field to
receive it even if it had — a queued order's real target price was silently
discarded end to end. The Pending Orders panel this phase needed to build
can't show a price that was never stored, so this is fixed now: `Order`
gained `Price float64 json:"price,omitempty"`, `createRequest` gained
`Price float64`, `Service.Create` stores it for non-market orders, and both
`OrderTicket.tsx` (hidden `price` field) and `orderActions.ts` (body) now
actually send it. Also updated the Postman collection's
"Create Order (Limit, queued)" example to include `price`.

## Frontend

- `frontend/src/app/portfolio/page.tsx` — Server Component, auth-gated
  (guest sees a sign-in prompt inside the same `SidebarNav` shell, per the
  guest-first rule already used elsewhere), parallel-fetches summary/
  positions/equity-history/allocation/stats/orders for the user's default
  portfolio and passes them to a Client Component for tab switching (fetch
  once, no re-fetch per tab — phase-e.md's own item 4 requirement).
- `PortfolioTabs.tsx` — the 4 tabs; KPI row always visible, tab body
  switches between (equity chart + holdings + allocation + pending),
  holdings-only, pending-only, and (journal + allocation).
- `KpiRow.tsx`, `EquityCurveChart.tsx`, `SectorAllocationCard.tsx`,
  `HoldingsTable.tsx`, `PendingOrdersCard.tsx`, `JournalCard.tsx` — one
  component per design panel, all consuming real API data.
- `PendingOrdersCard.tsx`'s cancel button is the first real caller of
  `POST /orders/:id/cancel` from the UI — wired via a new
  `cancelOrderAction` Server Action (`orderActions.ts`), called directly
  from a Client Component's `onClick` (via `startTransition`, not a
  `<form>`), with `revalidatePath("/portfolio")` on success.
- `navItems.ts`: `/portfolio` moved from `kind: "soon"` to `"built"`.

## Documented simplifications/deviations

- **No VN-Index benchmark line on the equity chart.** The design compares
  the account curve against VN-Index over the same period; V1 has no
  historical VN-Index series aligned to the account's own (sparse, real)
  fill timestamps, so faking one was rejected in favor of dropping the
  comparison and labeling the chart honestly ("dữ liệu thực từ lịch sử khớp
  lệnh").
- **No range-appropriate historical daily snapshots.** Equity history is
  real but sparse (one point per fill, not one per trading day) since V1
  has no scheduled snapshot job (deferred, see phase-e.md item 1's own
  note) — the 7-day/30-day/all filter still works correctly on real
  timestamps, it just may show few points for a young account.
- **Holdings table doesn't fetch company names.** Same simplification
  class as `MoversTable`'s symbol-only rows — avoids N extra per-symbol
  requests; the column shows the ticker only.
- **"Nắm giữ" is calendar days, not trading sessions.** Labeled "N ngày"
  rather than "N phiên" — this app has no real trading-session calendar,
  and overclaiming session semantics would be dishonest, same reasoning
  as the AI Insight feature's "rule-based preview" labeling.
- **Journal's "Từ Replay" cell dropped.** Replay Mode doesn't exist yet
  (Phase F) — replaced with a real closed-trade count instead of a
  feature-that-doesn't-exist-yet number.
- **Trading fee still not deducted from cash (pre-existing, found, not
  fixed here).** `order.Service.Create` computes `Fee` for display but
  never subtracts it from the portfolio ledger (`ApplyFill` only applies
  `price*quantity`) — true since Phase B/D, not introduced or worsened by
  this phase, and out of scope to fix here since it touches the paper-
  trading fill model itself, not the Portfolio screen. Flagged for a
  future session.
- **"Nạp lại tài khoản ảo" (reset virtual account) stays a disabled
  button** — no backend endpoint exists for it; same treatment as other
  not-yet-built actions elsewhere in this app (disabled, "Sắp ra mắt").

## Verification

- `go build ./... && go vet ./...` clean (real local Go 1.24, not Docker —
  this session's sandbox has no Docker daemon, unlike prior sessions;
  Docker verification deferred to whichever environment has it next).
  `gofmt -l` shows no new non-compliant files.
- `npx tsc --noEmit`, `npx eslint .`, `npm run build` all clean; hit and
  fixed one `react-hooks/purity` violation (`Date.now()` in
  `EquityCurveChart.tsx`'s `useMemo`) the same way this repo fixed the same
  class of issue before — pulled into a plain helper function.
- Backend, via curl against a real `go run ./cmd/api` (mock data source):
  registered a user, bought 100 VNM then sold 40 (market orders), placed a
  queued limit HPG order, and confirmed `equity-history`/`allocation`/
  `stats`/`positions` all returned correct, hand-checked numbers (cash
  96,312,009.4 + market value 3,687,990.6 = 100,000,000 exactly, since VNM's
  quote didn't move between the two same-second fills; `openSince` on the
  remaining VNM position matched the original buy's `filledAt`).
- Full Postman collection (`npx newman run backend/postman/...`): 36
  requests (33 defined + Newman's iteration bookkeeping), 0 failures —
  including the 3 new Portfolio endpoints and the updated Limit order body.
- Real browser (Playwright against a headless Chromium at
  `/opt/pw-browsers/chromium`, this sandbox's pre-installed browser, driving
  the actual `next dev` + `go run` servers, not a static reading of the
  code): registered a real account through the real `/register` form,
  placed real market buy/sell orders and a real queued limit order through
  the real `OrderTicket.tsx`, then screenshotted all 4 Portfolio tabs.
  Confirmed: guest state shows the sign-in prompt inside the sidebar shell;
  authenticated state shows real KPIs, a real (flat, correctly reflecting a
  net-zero round trip) equity curve, real sector allocation (bought VNM +
  FPT, held open — allocation showed "Consumer Staples 18,4%",
  "Information Technology 15,5%", "Tiền mặt 66,1%", matching hand-computed
  position weights), a real positions table with correct avg cost/current
  price/%NAV/days-held, and a real pending HPG limit order.
- **Actually exercised the cancel button**, not just read the code:
  clicked the real cancel (×) button on a real queued order in the browser,
  confirmed via the page's own text content that "Không có lệnh nào đang
  chờ khớp" appeared and the "Lệnh chờ" tab's badge count dropped from 1 to
  0 — the mutation, the server round-trip, and the `revalidatePath`-driven
  UI refresh all actually happened, not assumed from the diff.
