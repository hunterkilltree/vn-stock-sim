# Phase: Real fundamentals data for valuation and the rating card

Status: **steps 2 and 3 built offline (2026-10-21, branch
`claude/fundamentals-history-port`); steps 0, 1 and the wiring wait on
VCI access.** Expands slice 2 of
phase-valuation-backtest.md ("point-in-time fundamentals") into a
concrete fetch plan, and answers its open question 1 with option (a),
VCI's company-financials endpoints.

## Why the valuation card is mostly empty

The valuation formulas are built (`internal/valuation`). The data isn't.
`rating/service.go` `fundamentals()` fills only Price, EPS, BVPS, ROE
and CashDPS, all from the hand-seeded `symbol.MockProvider` snapshot.
Every other input is zero, so:

| Shown today | Missing input | Data needed |
|---|---|---|
| Graham Number: works, but on seeded EPS/BVPS | real EPS, BVPS | latest statements |
| Graham 1974, Lynch: "Thiếu dữ liệu tăng trưởng EPS" | `GrowthPct` | 4+ years of annual EPS |
| Rule #1: same | `GrowthPct`, `AvgHistPE` | annual EPS + P/E history |
| Weiss: "Cần lịch sử tỷ suất cổ tức" | `HighYield` | 5+ years of cash dividends + daily closes |
| P/E band: "Cần lịch sử P/E 5–7 năm" | `AvgHistPE`, `StdHistPE` | quarterly TTM EPS history + daily closes |
| Quality gate: checks only today's EPS | "EPS > 0 for several years" | annual EPS history |
| H2 factors (eps_growth, health, consistency): "Thiếu dữ liệu" | EPS CAGR, D/E, ROE σ | annual statements |
| Bond yield: constant 4.43% (2026-09-09) | a dated series | VN 10-year yield, monthly |
| Valuation backtests (phase-valuation-backtest.md slice 3) | all of the above, point-in-time | plus publication dates |

Daily closes already come from VCI (`market/vciprovider.go`), so the
missing piece is **per-company financial history and dividend events**,
plus one macro series.

## Sources

The same unofficial Vietcap API the price feed already uses (same
caveats: no licence, no SLA; phase-vci-market-data.md). Endpoints taken
from the open-source vnstock client (`vnstock/explorer/vci/financial.py`,
`company.py`, `const.py`, read 2026-10-21). **None of these have been
called from this repo yet.** This sandbox can't reach `*.vietcap.com.vn`
(checked again 2026-10-21: no connection), so the response shapes
below are from vnstock's parsing code, not seen directly.

Base: `https://iq.vietcap.com.vn/api/iq-insight-service`

| Endpoint | Gives | Used for |
|---|---|---|
| `GET /v1/company/{SYM}/financial-statement?section=INCOME_STATEMENT` | `data.years[]` / `data.quarters[]`, each with `yearReport`, `lengthReport` and item columns | net income to parent shareholders, revenue |
| `…?section=BALANCE_SHEET` | same shape | owners' equity, total debt |
| `GET /v1/company/{SYM}/financial-statement/metrics` | the dictionary mapping item column ids to names | mapping items; layouts differ for banks, brokers, insurers |
| `GET /v1/company/{SYM}/statistics-financial` | per-period ratios: `pe`, `pb`, `roe`, `debtToEquity`, `dividendYield`, `marketCap`, `numberOfSharesMktCap`, … | cross-check our derived numbers; shares outstanding |
| `GET /v1/events?ticker={SYM}&eventCode=DIV,ISS&fromDate=…&toDate=…&page=0&size=…` | dividends and share issues with dates | cash DPS history; stock-dividend/bonus dates for price adjustment |
| `GET /v1/company/details?ticker={SYM}` | company profile | sector, name (replaces the seeded values) |

vnstock first GETs `https://trading.vietcap.com.vn/priceboard` and
reuses its cookies. Our chart adapter doesn't need that; whether the IQ
endpoints do is one of the things the capture step checks.

**Bond yield:** no VCI endpoint known. Use a hand-maintained, dated
monthly series checked into the repo (`backend/internal/valuation/
bondyield.csv`, with the source of each row), extended by hand each
quarter. Replaces the `BondYieldPct` constant; the latest row feeds the
detail page, the as-of row feeds backtests.

## Publication dates

The financial statements carry a period (`yearReport`, `lengthReport`),
not a release date. Without one, a backtest would see Q4 numbers on
31 December, weeks before anyone could. Use the legal filing deadlines
(Circular 96/2020/TT-BTC, my reading; please confirm): quarterly reports
within 30 days of quarter end (45 for consolidated parents), audited
annual reports within 90 days. So:

- `PublishedAt = quarter end + 45 days` for quarterly figures
- `PublishedAt = year end + 90 days` for audited annual figures

This is late rather than early on purpose. Early would be look-ahead
bias; late only delays a signal by a few weeks. If VCI's news/events
feed turns out to carry report-release events, use those instead.

## Design

Follows the port → adapter shape CLAUDE.md asks for, and the port
phase-valuation-backtest.md slice 2 already names.

1. **Port in `symbol`:** `FinancialsProvider` with
   `History(sym) ([]Period, []DividendEvent, error)`. `Period`: fiscal
   year or quarter, net income to parent, revenue, equity, total debt,
   shares outstanding, `PublishedAt`, `IsBank`.
2. **Adapter `symbol/vcifinancials.go`:** same headers and HTTP client
   as `market/vciprovider.go`. Maps items through the metrics
   dictionary, with separate mappings for banks, brokers and insurers
   (vnstock's `_get_company_type` does the same). Unknown layout → error,
   never a guessed field.
3. **Cache:** fundamentals change four times a year. One fetch per
   symbol per 24 hours, through `market/swrcache.go`'s stale-while-
   revalidate cache, and with `DATABASE_URL` set, a `fundamental_periods`
   table so a restart doesn't refetch. Fetch lazily on first request;
   at startup, warm the seeded tickers at most 2 requests a second so we
   don't hammer an API that owes us nothing.
4. **Pure derivation `valuation.AsOf(periods, dividends, bars, t)`:**
   only periods with `PublishedAt ≤ t`. Computes:
   - TTM EPS = last 4 quarters' net income to parent ÷ shares as of t
   - BVPS = equity ÷ shares; ROE = TTM net income ÷ average equity
   - g = 3-year CAGR of annual EPS, capped at 25%; undefined if the
     start year's EPS ≤ 0
   - ROE σ over 3 years; D/E (skipped for banks)
   - CashDPS = cash dividends with ex-date in the last 12 months
   - P/E series = each daily close ÷ TTM EPS as of that day (EPS ≤ 0
     days skipped) → 5-year mean and σ
   - HighYield = 5-year max of TTM CashDPS ÷ close
   - quality gate: EPS > 0 in each of the last 3 years
5. **Wiring:** `rating/service.go` `fundamentals()` calls `AsOf(now)`.
   `symbol.Detail`'s EPS, P/E, P/B, ROE, dividend yield and market cap
   come from it when it succeeds. **Fallback:** if VCI fails, today's
   card uses the seeded snapshot (labelled as such); backtests refuse to
   run on mock fundamentals with live prices (phase-valuation-backtest.md
   slice 2).
6. **Share-count changes:** stock dividends and bonus issues (ISS
   events) change shares outstanding. Per-share figures always use
   shares as of the same date as the price they're compared with. If
   VCI's bars turn out to be **unadjusted**, adjust past closes by the
   ISS ratios in the adapter. Otherwise every ex-date looks like a crash
   to the dip and P/E-band rules.

## Steps

0. **Capture real responses (needs a machine that can reach VCI).**
   `backend/scripts/capture-vci-fundamentals.sh` saves every endpoint
   above for VCB (bank), FPT and HPG, plus daily bars across FPT's past
   stock-dividend ex-dates, into `backend/internal/symbol/testdata/vci/`.
   Commit the output. That turns "shape from vnstock's code" into
   fixtures the adapter is tested against, and answers the
   adjusted-prices question (phase-valuation.md open question 2).
1. **Adapter + port + cache**, tested against the fixtures. Result: the
   detail page's EPS, P/E, P/B, ROE and dividend yield are real. Fixes
   RESUME.md's "fundamentals are hand-seeded" gap.
2. **`AsOf` derivations**, with table tests: a period with
   `PublishedAt` after t is invisible; g is undefined after a loss year;
   P/E σ on a hand-built series. Result: Graham 1974, Lynch, Rule #1,
   Weiss and P/E band return numbers, and the H2 factors turn on.
   Covers phase-valuation-backtest.md slice 4.
3. **Bond-yield series** replacing the constant.
4. **Backtests** use `AsOf(bar date)`: phase-valuation-backtest.md
   slice 3, unchanged.

Step 1 depends on step 0. Step 3 is independent.

## Built so far (branch `claude/fundamentals-history-port`)

Everything that doesn't need a live response:

- `valuation/asof.go`: `Period`, `Dividend`, `Close`, `FilingDeadline`
  and `AsOf(periods, dividends, closes, t, opts) Snapshot`, implementing
  design item 4 and the share-basis rule in item 6. **Changed from the
  design:** the data types live in `valuation`, not `symbol`, so the
  pure package stays dependency-free and the port in `symbol` returns
  `valuation` types. P/E and yield history are sampled at month-end
  closes, not daily (enough for a 5-year band). The quality gate's
  "EPS > 0 for 3 years" became `Fundamentals.RecentLoss` (a loss in the
  last 3 fiscal years), so a short history doesn't fail the gate.
- `symbol/financials.go`: the `FinancialsProvider` port and
  `Financials` type. Nothing implements it yet.
- `valuation/bondyield.csv` + `BondYieldOn(t)` (step 3). It holds only
  the one row we have (4.43% on 2026-09-09, unverified), so behaviour
  today is unchanged. Before its first row, `BondYieldOn` reports the
  yield as unknown rather than borrowing a later one. **Needs more
  rows**, with sources, before any backtest before 2026-09 can use
  Graham 1974.
- Tests (`asof_test.go`): an unpublished quarter is invisible;
  quarterly-to-annual fallback; growth restated across a bonus issue;
  growth undefined after a loss year; ROE σ; the P/E band needs 3 years
  of points; trailing cash yield and its 5-year high; flat P/E across a
  1:1 bonus issue on both adjusted and unadjusted closes.

Not wired yet: `rating/service.go` still feeds `valuation` from the
seeded snapshot. Wiring comes with the adapter (step 1), so the card
doesn't change until there is real data behind it.

## Verification

- `go build ./... && go vet ./... && go test ./...`.
- Fixture tests: VCB maps through the bank layout and skips D/E; FPT's
  derived TTM EPS and P/E match `statistics-financial`'s figures within
  1%.
- Live check on a machine that reaches VCI: `/stocks/FPT`'s 1–3 năm
  tab shows numbers, not "Thiếu dữ liệu", for every method that applies.

## Open questions (for the user)

1. Can you run step 0's script on your machine (or anywhere outside
   this sandbox) and commit the output? Everything after it depends on
   seeing real responses.
2. Are you OK relying on VCI's unofficial API for fundamentals too, as
   for prices? The alternative is a licensed vendor (Phase M's
   data-vendor question).
3. Bond yield: is a hand-maintained monthly CSV acceptable, or do you
   have a source with an API?
