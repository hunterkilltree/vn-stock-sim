# Phase: Buy/Sell rating on the stock detail page

A rule-based **Mua mạnh / Mua / Trung lập / Bán / Bán mạnh** (Strong Buy ..
Strong Sell) rating, shown as the "Khuyến nghị" card at the top of the
right column on `/stocks/[symbol]`.

- Backend: `backend/internal/rating` -- `rules.go` (pure `Compute`, every
  threshold as a named constant), `service.go`, `handler.go`.
- API: `GET /api/v1/symbols/:symbol/rating`, public like `/insight`.
- Frontend: `frontend/src/components/RatingCard.tsx`, `getRating` in
  `lib/api.ts`.

## Decisions

1. **Five steps, TradingView-style roll-up.** Each rule votes +1 (buy),
   0 (neutral) or -1 (sell); a group's score is the mean vote in [-1, 1].
   Score ≥ 0.5 → Strong Buy, ≥ 0.15 → Buy, > -0.15 → Neutral,
   > -0.5 → Sell, else Strong Sell -- the same shape as TradingView's
   Technical Ratings, so the scale reads as users expect. Neutral is kept
   (not just the four buy/sell steps) because forcing a call on a flat
   chart is exactly what good practice says not to do.
2. **Overall = 60% technical + 40% fundamental.** For a simulator focused
   on trading, price action leads; valuation/quality moderates it. A
   symbol with no fundamentals (crypto, unseeded tickers) is rated on
   technicals alone; a zero fundamental field is treated as missing and
   doesn't vote.
3. **Technical rules** (daily bars, ~420 calendar days fetched so SMA 200
   exists; rules needing more history than available are skipped):
   - Price vs SMA 20 / 50 / 200 -- above by > 1% buys, below by > 1% sells
     (short/medium/long trend; the ±1% band ignores noise around the line).
   - SMA 50 vs SMA 200 -- golden cross buys, death cross sells.
   - RSI 14 -- < 30 oversold buys, > 70 overbought sells (contrarian at
     the extremes only).
   - MACD (12, 26, 9) vs its signal line -- above buys, below sells.
     Shown as a % of price so stocks (VND) and crypto read the same.
   - 3-month momentum (63 sessions) -- > +5% buys, < -5% sells.
   - Volume confirmation -- a session at ≥ 1.5x its 20-day average volume
     confirms that day's direction; otherwise neutral.
4. **Fundamental rules** (from `symbol.Detail`):
   - P/E -- negative EPS (loss-making) sells; P/E < 10 buys; > 25 sells.
   - P/B -- < 1 (below book) buys; > 3 sells.
   - ROE -- ≥ 15% buys; < 8% sells.
   - Dividend yield -- ≥ 4% (≈ a Vietnamese 12-month deposit rate) buys;
     lower is neutral, never a sell (growth stocks often pay none).
5. **Transparent, not advice.** Every rule's input value and vote is
   listed in the card behind "Xem tín hiệu", and the card says it's for
   simulation reference only. No LLM, same reasoning as `internal/insight`.

## Verification

- `go test ./internal/rating` -- uptrend + cheap fundamentals → Strong
  Buy; downtrend + loss-making → Strong Sell; flat → Neutral; no
  fundamentals → technical only; short history skips SMA 200/MACD;
  volume spike confirms direction; cutoffs.
- `go build/vet/test ./...`, frontend `typecheck`, `eslint`, `build` clean.
- Rendered `/stocks/FPT` (mock data) at 1440px and 390px in headless
  Chromium: card shows the rating, the 5-step scale, technical/fundamental
  split and the expanded signal list; the mobile Mua/Bán buttons still
  jump to the order ticket (`#dat-lenh` moved onto the ticket's wrapper).

## Still open

- Fundamentals are hand-seeded (see RESUME.md), so the fundamental half
  is only as good as that seed data.
- Thresholds are fixed rules of thumb, not sector-relative (a bank's
  normal P/B differs from a tech stock's).
