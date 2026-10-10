# Phase: Market risk regime, crash scenarios and stress tests

Status: **slice 1 built (2026-10-10, branch `claude/market-regime-indicator`); slice 2 partly built (exposure warning, branch `claude/regime-aware-risk-limits`); slices 3–6 planned.** Asked for on 2026-10-10: "2008 had
many warning signs. If 2026 follows the same pattern, what can the app
do so users see it coming and avoid losing money?"

## The honest framing (this shapes every feature below)

- **No indicator reliably predicts a crash.** The signals before 2008
  were clear *in hindsight*. At the time, the same signals also fired in
  years that didn't crash. Leaving the market on false alarms also costs
  money (missed rebounds).
- What works better, and what this phase builds, is **measuring
  conditions and shrinking risk when they deteriorate**. Investors who
  survived 2008 mostly didn't predict it. They had rules that cut
  exposure as the trend broke, and they weren't on margin.
- The app must **not** say "a crash is coming". It says: "conditions
  look like this; historically, conditions like this came with bigger
  drawdowns; here is what that would do to *your* portfolio; here is how
  to reduce it." Every number is labelled as historical, not a forecast.
- I can't judge whether 2026 resembles 2008. That depends on current
  macro data the app doesn't have yet (see slice 4).

### What happened, for context (approximate; check against data before
showing in the app)

| Episode | VN-Index move | Warning signs at the time |
|---|---|---|
| 2007–2009 | ~1,170 (Mar 2007) → ~235 (Feb 2009), about −80% | Credit growth ~50% in 2007; inflation peaking near 28% y/y (Aug 2008); State Bank refinancing rate raised to 15% (Jun 2008); market P/E very high in early 2007; real-estate and margin-lending boom; then the global crisis (Sep 2008) |
| 2018 | about −25% from the April peak | Fast run-up, foreign net selling, global rate rises |
| 2020 | about −33% in Jan–Mar | Covid shock: fast, then an equally fast recovery |
| 2022 | ~1,530 (Jan) → ~870 (Nov), about −43% | Corporate-bond crisis, margin calls, rising interest rates, tighter credit |

The common thread the app *can* measure: the index lost its long-term
trend early, market breadth narrowed, volatility jumped, and rates and
credit tightened. Margin made it worse: forced selling at the floor
price, when there are no buyers.

## Features

### 1. "Nhiệt kế thị trường" — market risk regime card

On the market overview (Main) page and the Portfolio page: a traffic
light, **Bình thường / Thận trọng / Rủi ro cao** (normal / caution /
high risk), computed daily from signals the app already has or can get.
Each signal is shown with its value and one line on why it matters, the
same pattern as the Khuyến nghị card.

| Signal | Measure | Data |
|---|---|---|
| Long-term trend | VN-Index vs its 200-day average, and the 200-day average's slope | have (`GetBars("VN-Index")`) |
| Drawdown | VN-Index % below its 52-week high | have |
| Volatility regime | 20-day volatility vs its 1-year median | have |
| Breadth | % of the stock universe above its own 200-day average; new highs vs new lows | have, but the universe is only ~40 seeded tickers (open question 3) |
| Market valuation | Universe median P/E vs its own history | partial: P/E history needs phase-valuation-backtest.md slice 2 |
| Rates and credit | Policy/interbank rate trend, CPI, credit growth, margin debt | **don't have**: slice 4 |

Regime = how many signals are in their risk zone, with knock-outs: for
example, VN-Index below a *falling* 200-day average is at least
"Thận trọng".

### 2. Regime-aware risk rules (connects to phase-risk-rules.md)

This is what actually prevents losses. Each regime tightens the
portfolio's limits automatically:

| Regime | Risk per trade | Max total exposure | New-trade lock (rule 4) |
|---|---|---|---|
| Bình thường | setting (default 1%) | 100% | 6% |
| Thận trọng | half | 70% | 4% |
| Rủi ro cao | quarter | 40% | 3% |

Coach mode warns ("Thị trường đang rủi ro cao: mức rủi ro mỗi lệnh đã
giảm còn 0,25%"); strict mode enforces. All numbers are per-portfolio
settings, like the rest of phase-risk-rules.md.

### 3. "Kịch bản khủng hoảng" — crash replays

Replay already accepts a `startDate`. Add curated scenarios: start
Replay a few months **before** each episode above, with the warning
signals visible in the sidebar as they were at the time. The user
practises noticing the regime change, cutting exposure and honouring
stops, and the process score from phase-risk-rules.md grades it.
Afterwards a summary shows "you vs buy-and-hold vs the 200-day-average
rule".

### 4. Stress test for the user's own portfolio

On the Portfolio page: "Nếu thị trường lặp lại 2008 / 2022 / 2020…"
(if the market repeats…). For each holding, estimate its sensitivity
(beta) to VN-Index from its own history, apply the episode's index
path, and show:

- the estimated portfolio loss in VND and %
- worst holdings first
- **days that would have been locked at the floor price**, meaning stops
  couldn't fill (phase-risk-rules.md rule 8), with the T+2 effect added
  (rule 7)

Labelled as a rough estimate. Beta is unstable in crises, and
correlations go to 1.

### 5. Evidence: backtest the regime filter

Add an option to the backtest engine: "chỉ nắm giữ khi VN-Index trên
SMA 200" (only hold while VN-Index is above its 200-day average). Run it
against buy-and-hold with the same costs (phase-valuation-backtest.md
slice 1). This shows the user the trade-off honestly: in the US data
it's best known from, a trend filter like this cut the deepest
drawdowns but gave up some return and suffered whipsaws. The app should
show the VN result, whatever it is, rather than assert it.

### 6. Alerts when the regime changes

When the light changes colour, or a held stock falls through its
200-day average, send an in-app alert. This ties into the planned V2
alerts (RESUME.md: in-app price alerts on the matcher loop).

## Data checks before building

1. **How far back does VCI history go?** Today's adapter fetches at most
   2000 bars per request (≈8 years, back to ~2018). Episodes before that
   need paged requests, and **it's unverified whether VCI has 2007–2009
   daily bars at all** (the sandbox can't reach VCI). If it doesn't,
   2008 can only appear as a described scenario (index path from a
   cited source), not a Replay.
2. **Macro series** (refinancing/interbank rate, CPI, credit growth,
   margin debt) need a source: hand-entered monthly figures with their
   source and date, or a data provider. Hand-entered is fine for monthly
   macro data if every point cites its source.
3. **Breadth on ~40 tickers is thin.** A real breadth signal needs the
   full HOSE list (~400 tickers) of daily bars, which is heavier on VCI.

## Slices — one branch each

| # | Branch | Contents | Depends on |
|---|---|---|---|
| 1 | `claude/market-regime-indicator` | `internal/regime`: pure signal functions on VN-Index bars + universe breadth; `GET /market/regime`; Nhiệt kế thị trường card on Main and Portfolio | nothing; can start now |
| 2 | `claude/regime-aware-risk-limits` | Regime-based tightening of risk per trade, exposure and the lock (feature 2) | phase-risk-rules.md slices 1 and 3 |
| 3 | `claude/portfolio-stress-test` | Beta per holding, episode paths, floor-lock days (feature 4) | episode index paths (data check 1) |
| 4 | `claude/macro-risk-signals` | Macro series port + hand-entered cited data, added to the regime (feature 1, last row) | data check 2 |
| 5 | `claude/crisis-replay-scenarios` | Curated Replay starts with time-stamped signals (feature 3) | data check 1, slice 1 |
| 6 | `claude/backtest-regime-filter` | SMA-200 regime filter option in backtests (feature 5) | phase-valuation-backtest.md slice 1 (costs) |
| — | (with alerts, later) | Regime-change alerts (feature 6) | V2 alerts |

Slice 1 delivers the most value on its own: a daily, explained read of
market conditions from data the app already has.

## Verification (per slice)

- Unit tests on synthetic VN-Index series: a steady uptrend → Bình
  thường; a series that breaks below a falling 200-day average with
  rising volatility → Rủi ro cao; regime knock-outs; breadth on a
  hand-built universe.
- Slice 3: a portfolio with beta 1 under a −40% path loses ~40% before
  floor-lock effects, and floor-lock days are counted correctly on a
  limit-down sequence.
- Slice 6: backtest of the filter on a synthetic crash series cuts the
  drawdown compared with buy-and-hold, and costs are charged on every
  switch.
- UI slices: headless-Chromium screenshots at 1440px and 390px.

## Built (slice 1) — what changed from the plan

- `backend/internal/regime`: pure `Compute` over VN-Index daily bars and
  the stock universe's bars; `GET /api/v1/market/regime` (public);
  service caches the result for 10 minutes (one bars fetch per universe
  symbol otherwise).
- Signals and thresholds:
  - Trend: above SMA 200 with SMA 200 rising = ok; one of the two =
    caution; below a falling SMA 200 = risk.
  - Drawdown from the 52-week high: −10% caution, −20% risk.
  - Volatility: 20-day volatility ÷ its 1-year median, 1.25× caution,
    1.75× risk.
  - Breadth: % of the universe above its own SMA 200, < 60% caution,
    < 40% risk, plus 52-week highs and lows.

  Caution = 1 point, risk = 2; level ≥ 2 Thận trọng, ≥ 4 Rủi ro cao.
  Knock-out: below a falling SMA 200 is never "Bình thường".
- Valuation and macro rows are shown as "Chưa có dữ liệu" and not
  scored (data checks 2 and 4 still open).
- **Mock fix:** `MockProvider.GetBars` for an index name now uses the
  same `indexValueFor` series `GetIndex` shows on the index cards.
  Before this, mock "VN-Index" bars came from the stock-price generator,
  so the regime card and the index card disagreed.
- Frontend: `MarketRegimeCard.tsx`, full card on `/stocks` (top of the
  right column on desktop, after the VN-Index hero on mobile) and a
  compact one-line banner on `/portfolio` for stock portfolios. Each
  level carries the feature-2 tightening as advice ("Nên làm: …");
  enforcement waits for slice 2.
- Open questions taken at their defaults: placement on Main + Portfolio;
  breadth over the ~40 seeded tickers (the card says so).

## Verification (slice 1)

- `go test ./internal/regime`: steady uptrend → Bình thường with 100%
  breadth; synthetic crash (falling 0.4%/day with violent swings) → all
  four signals risk → Rủi ro cao; slow bleed under a falling SMA 200 →
  knock-out to Thận trọng; no-data signals excluded from the max score;
  short history → not ok; annualised-volatility arithmetic.
- `go build/vet/test ./...`, frontend `typecheck`/`eslint`/`build` clean.
- Headless Chromium on mock data: `/stocks` at 1440px and 390px and
  `/portfolio` (logged-in test user) at 1440px. Card and banner render,
  "Vì sao quan trọng?" expands, no console errors. The mock market reads
  Thận trọng (2/8: volatility 1.3× normal, breadth 52%).
- **Not verified:** live VCI data (the sandbox can't reach VCI).

## Built (slice 2, partial) — exposure warning

The user chose to build only the part that works without the risk-rules
engine (phase-risk-rules.md slices 1 and 3 don't exist yet):

- `regime.LimitsFor(level)`: exposure cap 100/70/40%, risk per trade
  1/0.5/0.25%, monthly loss lock 6/4/3%. Returned as `limits` on
  `GET /market/regime`; the card's "Nên làm" text is built from it
  (`frontend/src/lib/regime.ts`), so the numbers live in one place.
- Order ticket (`OrderTicket.tsx`, `lib/exposure.ts`): on the stock page,
  "Tỷ trọng cổ phiếu 57% → 97% / trần 70%" (stock exposure before → after,
  against the cap); past the cap, a warning with a one-click "Giảm còn N
  cổ phiếu" (largest whole-lot buy that stays under the cap, fee
  included). **Warning only, the order still goes through** (coach mode
  in phase-risk-rules.md terms).
- Portfolio banner: stock exposure meter with the cap marked.

**Still open in slice 2:** risk-per-trade and the loss lock need
stop-losses and the monthly loss tally (phase-risk-rules.md slices 1 and
3); strict-mode blocking needs its per-portfolio settings. When those
land, the exposure check moves into `internal/riskrules` as one rule.

Verification: `go test ./internal/regime` (limits tighten
monotonically; a regime's limits match its level). Headless Chromium
with a test user holding 57% VCB under a Thận trọng regime: "Tối đa"
(700 shares) → 97% and the warning; "Giảm còn 200 cổ phiếu" sets 200
and the warning clears (room 13.0M VND ÷ (56,930 × 1.0015) → 228 →
200 in lots); banner shows 57% / trần 70%; no console errors.

## Open questions (for the user)

1. **Where the light lives:** Main page, Portfolio page, the stock page
   header, or all three?
2. **Regime tightening:** on by default in coach mode (warnings only),
   or opt-in?
3. **Breadth:** expand the stock universe to the full HOSE list (more
   VCI calls), or accept breadth over the ~40 seeded tickers for now?
4. **Macro data:** hand-enter monthly figures with cited sources, or
   wait for a data provider?
5. **2008 data:** if VCI doesn't serve 2007–2009 bars, is a described
   scenario (cited index path, no Replay) acceptable for 2008?
