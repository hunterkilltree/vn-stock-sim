# Phase: Holding horizon — "should I buy to hold 3–6 months or 1–3 years?"

Status: **planned, not built.** Builds on phase-rating.md (the "Khuyến
nghị" card and `internal/rating`).

## Goal

The existing rating answers "what do the signals say *right now*?",
which is a short-term (days to weeks) read. Users also want to know
whether a stock is worth **holding**, and the answer depends on the
time frame:

| Horizon | Vietnamese label | What usually decides the outcome |
|---|---|---|
| Medium, 3–6 months | **Trung hạn (3–6 tháng)** | Trend, relative strength vs. the market, risk/volatility, liquidity. Valuation only as a sanity check. |
| Long, 1–3 years | **Dài hạn (1–3 năm)** | Business quality (ROE, growth), valuation vs. quality, balance-sheet health, dividends. Price trend matters little. |

For each horizon the user gets:

1. A verdict: **Phù hợp nắm giữ** (suitable to hold) / **Cần theo
   dõi** (watch) / **Chưa phù hợp** (not suitable now).
2. A 0–100 score and the factors behind it, each with its value, its
   sub-score and a one-line explanation of *why it matters for this
   horizon*. This is the "knowledge" part: the card teaches the rule as
   well as giving the answer.
3. **What history says:** if you had bought this stock on any day in the
   past ~8 years and held it for 6 months (or 1 year / 3 years), how
   often did you make money, and what were the median, best and worst
   outcomes, next to the same numbers for VN-Index. This uses only data
   the backend already has, and it's the most honest evidence the app
   can show.
4. **A risk plan** for the medium horizon: typical swing (volatility), the
   worst drawdown in the period, and a suggested stop-loss level. Long
   horizon: the worst drawdown over 3 years, so the user knows what
   "holding through it" has meant before.

Not financial advice: the same disclaimer as the rating card, plus
"past returns don't predict future returns" on the history section.

## Where it shows up

The "Khuyến nghị" card on `/stocks/[symbol]` gets three tabs:

- **Ngắn hạn**: today's rating (unchanged, phase-rating.md).
- **3–6 tháng**: the medium-horizon verdict.
- **1–3 năm**: the long-horizon verdict.

The tab row is a small Client Component; the data stays server-fetched
in one request (decision 2). One card with tabs, not two new cards, so
the right column doesn't grow by two screens on mobile.

A "Thử nắm giữ trong Replay" link on the medium/long tabs opens Replay
at a random past date for this symbol, so the user can practise the
holding decision. That fits the product tagline "Trade the past before
you trade the future".

## Decisions

1. **Factors scored 0–100 by linear interpolation, not ±1 votes.** A
   horizon verdict needs finer grading than the short-term rating's
   votes. Each factor has a "bad" and a "good" threshold, and a value
   between them scores proportionally. Horizon score = weighted mean of
   the factors that have data, with weights re-normalised over those
   factors. Bands: **≥ 65 Phù hợp, 45–64 Cần theo dõi, < 45 Chưa phù
   hợp.**
2. **Extend `GET /symbols/:symbol/rating` with `horizons: { medium, long }`**
   rather than adding a new endpoint. It's one page section, one request
   and one bars fetch. The lookback grows from 420 days to ~8 years of
   daily bars (VCI's `countBack` cap is 2000 bars ≈ 8 years; the SWR cache
   from #43 absorbs the extra cost). The short-term rules keep using only
   the tail they need.
3. **Knock-out rules override the score** (common practice in screening):
   - Long: EPS < 0 (loss-making) → at most *Cần theo dõi*; average
     traded value < 1 bn VND/day → *Chưa phù hợp* (you can't exit).
   - Medium: price < SMA 200 **and** SMA 50 falling → at most *Cần theo
     dõi* ("don't fight a downtrend"); liquidity < 1 bn VND/day →
     *Chưa phù hợp*.
   The card names the knock-out when one fires.
4. **Phase it by data availability** (fundamentals are hand-seeded,
   RESUME.md):
   - **H1, data we have now:** everything that comes from bars, plus
     the current P/E, P/B, ROE, EPS and dividend yield, plus sector
     medians computed from the seeded universe.
   - **H2, new seeded fields:** EPS growth (3-year CAGR), revenue growth,
     debt/equity (non-banks), net margin and ROE history (3 years). These
     are added as a `Financials` struct on `symbol.Detail`, hand-seeded
     like today's fields for the ~40 tickers, behind the same
     provider interface so a real vendor can replace it later. Until H2
     ships, those factors show as "Thiếu dữ liệu" (no data) and don't
     count.
5. **Sector-relative valuation, not fixed cut-offs.** This fixes the
   rating's open issue (phase-rating.md "Still open"): a bank's normal
   P/B isn't a tech company's. P/E is compared with the median P/E of its
   sector in the universe.
6. **Justified P/B for the long horizon:** fair P/B ≈ (ROE − g) /
   (CoE − g), with cost of equity CoE = 13% (VN 10-year bond ~3% + equity
   risk premium ~10%) and long-run growth g = 5%. Actual P/B ÷ fair P/B
   < 0.8 is cheap and > 1.3 is expensive. This is the textbook way to
   judge "is this ROE worth this P/B", and it works across sectors.

## Factors

Weights are the share of the horizon score; *H2* marks factors that wait
on the new data.

### Trung hạn (3–6 tháng)

| Factor | Measure | Scores 0 → 100 | Weight | Why (shown in card) |
|---|---|---|---|---|
| Trend structure | price vs SMA 50/200, SMA 50 slope over 20 sessions | below both and falling → above both and rising | 25% | Medium-term winners are usually already in an uptrend. |
| Relative strength | 6-month return minus VN-Index's, skipping the last month | ≤ −15 pts → ≥ +15 pts | 25% | 3–12-month momentum is one of the best-documented return factors (Jegadeesh & Titman). |
| Near 52-week high | price ÷ 52-week high | ≤ 70% → ≥ 95% | 10% | Stocks near their highs tend to keep going (52-week-high effect). |
| Volatility | 60-day annualised σ | ≥ 60% → ≤ 25% | 15% | Lower swing = easier to hold; sets position size. |
| Liquidity | 20-day average traded value | ≤ 1 bn → ≥ 20 bn VND/day | 10% | You must be able to get in and out. |
| Valuation sanity | P/E ÷ sector median P/E | ≥ 2× → ≤ 1× | 15% | Avoid paying any price for momentum. |

Risk plan: suggested stop = max(price − 2 × ATR(14), SMA 50 − 1 tick),
shown as a % below the price; max drawdown over the last 6 months.

### Dài hạn (1–3 năm)

| Factor | Measure | Scores 0 → 100 | Weight | Why |
|---|---|---|---|---|
| Profitability | ROE | ≤ 8% → ≥ 20% | 20% | High returns on capital compound over years. |
| Value vs quality | P/B ÷ justified P/B (decision 6) | ≥ 1.5 → ≤ 0.7 | 20% | A good business is only a good investment at a fair price. |
| Relative P/E | P/E ÷ sector median | ≥ 2× → ≤ 0.7× | 10% | Cheap vs peers. |
| Dividend | dividend yield | 0% → ≥ 5% | 10% | Cash returned while you wait; ≈ deposit rate. |
| Size | market cap | ≤ 1,000 bn → ≥ 20,000 bn VND | 5% | Small caps in VN carry more governance and liquidity risk over years. |
| Long-run price behaviour | 3-year max drawdown, 3-year CAGR vs VN-Index | worse → better than index | 10% | How holding this stock has felt before. |
| *H2* Earnings growth | 3-year EPS CAGR | ≤ 0% → ≥ 15% | 15% | Over years, price follows earnings. |
| *H2* Financial health | D/E (non-banks); banks: skip | ≥ 2 → ≤ 0.5 | 5% | Debt is what turns a bad year into a fatal one. |
| *H2* Consistency | ROE stdev over 3 years | high → low | 5% | Steady earners beat one-off spikes. |

In H1 the H2 weights (25%) are re-normalised away, and the card shows
"Đánh giá dựa trên dữ liệu hiện có" (based on available data).

### Holding-period history (both horizons)

From the daily bars, for every start day with a full window ahead of it:
the return after holding 126 sessions (6 months), 252 (1 year) and 756
(3 years). Report the % of windows with a positive return, and the
median, 10th-percentile and 90th-percentile returns, for the stock and
for VN-Index over the same windows. Not scored. It's evidence shown next
to the verdict, with caveats: the windows overlap, and ~8 years gives
few independent 3-year samples (the card shows the sample count).

## API shape (added to the rating response)

```json
"horizons": {
  "medium": {
    "verdict": "suitable | watch | avoid",
    "score": 72,
    "knockout": null,
    "factors": [
      { "key": "trend", "label": "Xu hướng", "value": "Trên SMA 50 & 200",
        "score": 90, "weight": 0.25, "why": "…" }
    ],
    "risk": { "volatility": 31.5, "maxDrawdown": -18.2, "stopLoss": 48100, "stopPct": -8.4 },
    "history": { "sessions": 126, "samples": 1700, "winRate": 58.0,
                 "median": 4.1, "p10": -14.0, "p90": 22.5,
                 "index": { "winRate": 61.0, "median": 3.2 } }
  },
  "long": { "…": "same shape; risk has maxDrawdown only" }
}
```

## Work breakdown

H1:
1. `market`: expose VN-Index daily bars. `VCIProvider.GetIndex` already
   fetches them (`fetchOHLC("VNINDEX", "ONE_DAY", …)`) but only returns
   a snapshot. Add `GetIndexBars(name, from, to)` to the provider
   interface, with a synthetic series in the mock. ATR is computed
   locally in `rating` from the stock's bars.
2. `rating/horizon.go`: pure `ComputeHorizons(detail, bars, indexBars,
   sectorStats)`. Factor functions, interpolation, knock-outs, history.
3. `rating/service.go`: 8-year fetch; sector medians from
   `symbol.Service` (computed once and cached, since the universe is
   static); index bars.
4. Unit tests: each factor's interpolation edges; knock-outs; weight
   re-normalisation when factors are missing; history on a synthetic
   series with known answers (e.g. a steady +10%/yr series → 100% win
   rate).
5. Frontend: `RatingCard` gets tabs (client component `HorizonTabs`);
   `HorizonPanel` renders the verdict, factor list, risk plan, history
   strip and the Replay link.
6. Docs: verification section here, RESUME.md row.

H2:
7. `symbol.Financials` (EPS growth, revenue growth, D/E, net margin,
   ROE history), hand-seeded for the universe, exposed on `Detail`.
8. Turn on the H2 factors; update tests and the card's "available data"
   note.

## Verification plan

- `go test ./internal/rating` covering the cases in step 4, plus golden
  cases: a steady uptrend + high ROE + fair P/B → both horizons *Phù
  hợp*; a downtrend + loss-making → medium knock-out and long *Chưa phù
  hợp*.
- Build, vet, typecheck, eslint and build all clean.
- Render `/stocks/FPT` and a bank (VCB) at 1440px and 390px; check the
  tabs, that the mobile Mua/Bán anchor still lands on the order ticket,
  and that the history strip reads correctly.

## Open questions (for the user)

1. **Verdict labels:** three steps (Phù hợp / Cần theo dõi / Chưa phù
   hợp) as planned, or the same five steps as the rating (Mua mạnh …
   Bán mạnh) for each horizon?
2. **H2 data:** hand-seed growth/debt figures for the ~40 tickers (fast,
   approximate, same as today's fundamentals), or wait for a real
   fundamentals source (Phase M lists the data-vendor question)?
3. **Stop-loss suggestion:** some users read a suggested stop as advice.
   Keep it, or show only volatility and drawdown?
