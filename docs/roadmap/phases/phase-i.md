# Phase I — Crypto market, real prices, and the full heatmap dashboard

Plan written against `design/screens/Crypto-Main.dc.html`,
`Crypto-Detail.dc.html`, `Crypto-Replay.dc.html`, and (for the heatmap)
the heatmap panel of `Main.dc.html`, per FULL-APP-PLAN.md section 11.

User's request, verbatim: "go start phase I and with real stock price,
also add full heatmap dashboard follow desgin prototype". Read as: real
market prices rather than generated ones (for crypto, which was the open
question, and for the stocks the heatmap shows), plus a full-screen
heatmap. design/SCREENS.md lists the full-screen heatmap as **not
designed** ("only the dashboard panel exists"), so it is built in the
panel's exact visual language (tile colour formula, legend, sector
blocks) scaled up to a full screen — flagged as a design extrapolation.

## Environment finding that shapes verification

This sandbox's network policy refuses both `trading.vietcap.com.vn` and
`data-api.binance.vision` (the proxy answers 403 to CONNECT). So:
stock prices seen in this session have been the mock fallback all along,
and neither real adapter can be exercised live here. Both are written
against the providers' documented response formats, unit-tested against
local fake servers speaking those formats, and fall back per call to the
deterministic generators — the same "LiveProvider" decorator pattern the
VCI adapter already uses. Seeing live prices needs the two hosts allowed
in the environment's network settings (or a normal deployment).

## Design decisions (made before code)

1. **Crypto data from Binance's public market-data API**
   (`data-api.binance.vision`, no key: `ticker/24hr`, `klines`,
   `depth`) behind a `CryptoDataProvider` port, with a deterministic mock
   fallback and a short cache — exact mirror of the stock side. Pairs
   Binance doesn't list (the design includes OKB and VNDC) are served by
   the mock and marked `source: "mock"` per pair, never mixed silently.
2. **One `internal/crypto` package** (pairs + market data + heatmap/
   movers/overview) instead of the plan's two (`pair`, `cryptomarket`):
   the pair list and its market data are one small feature, and splitting
   them bought nothing but an extra import. Still fully separate from the
   stock `symbol`/`market` packages, as FULL-APP-PLAN.md 2.3 requires.
3. **The pair universe is the design's own 24 coins in its 4 categories**
   (Layer 1, DeFi, Sàn & hạ tầng, Dự án Việt Nam). Static facts Binance
   doesn't provide (circulating/max supply, all-time high, name) are
   hand-seeded, like the stock fundamentals already are, and labelled as
   such. Market cap = live price × seeded circulating supply.
4. **"Vốn hoá toàn cầu" / "Thống trị BTC" cards are computed over the
   app's 24 pairs** and labelled that way ("trong 24 coin của ứng
   dụng") — a real global figure needs a market-cap aggregator this app
   doesn't have.
5. **Positions and orders become fractional.** Crypto trades 0,025 BTC,
   so `Quantity` becomes `float64` in `order`, `portfolio` and `replay`.
   Stock orders are still validated as whole shares; crypto orders as
   positive amounts up to 8 decimals.
6. **Crypto symbols use the exchange form `BTCUSDT`** in orders and
   positions (displayed "BTC/USDT"), so they can never collide with a
   VN ticker in a portfolio's position map.
7. **One quote source for orders and valuation.** `order` and
   `portfolio` already read prices through a `Detail(sym)` port; a small
   `crypto.QuoteRouter` answers for `…USDT` pairs from the crypto service
   and delegates everything else to the stock symbol service. So the
   existing order/portfolio stacks are reused, not duplicated.
8. **Market rules enforced in `order`**: a crypto portfolio only accepts
   crypto pairs and a stock portfolio only VN tickers (409
   `wrong_market`); fee 0,10 % for crypto (Crypto-Detail's "Phí mô phỏng
   (0,10%)") vs 0,15 % for stocks; order types Thị trường / Giới hạn /
   Dừng / OCO (OCO stores a limit price and a stop price; like every
   non-market order in this app, queued, not matched).
9. **A crypto wallet per user, opt-in.** The stock "default portfolio"
   stays stock-only. A user gets a crypto portfolio by creating one
   ("Ví crypto", 10.000 USDT — the design's amount) from the Crypto
   overview, the account menu (now with a Cổ phiếu / Crypto choice), or
   Signup (the 10.000 USDT option is now enabled and opens a crypto
   wallet next to a default stock portfolio). The active crypto wallet
   has its own cookie, so switching one never changes the other.
10. **Portfolio page stays stock-only**; the crypto wallet and its
    holdings live on the Crypto overview, exactly where Crypto-Main.dc.html
    draws them (SCREENS.md: crypto paper trading "borrows the stock
    screens" — there is no crypto portfolio screen). Choosing a crypto
    wallet in the account menu goes to `/crypto`.
11. **Crypto Replay reuses the Phase F engine**: `market: "crypto"`, 1-hour
    candles from 2021-05-01 00:00 UTC (the design's May-2021 crash),
    120 candles, a 10.000 USDT replay wallet, fractional quantities,
    timestamps not dates. The design's "stricter risk scoring" isn't
    specified anywhere; the same documented heuristic is used and the
    difference is flagged.
12. **Charts get a price scale.** `DetailChart`/`ReplayChart` currently
    divide by 1000 (VND in thousands); they take a `scale`/decimals
    setting instead, 1000 for stocks and 1 for USDT pairs.
13. **Mode switch** exactly as drawn: "Cổ phiếu | Crypto" segmented
    control under the sidebar logo, "CP / CRY" at the top of the rail.
14. **Stock universe for the heatmap**: the design's own heatmap tickers
    (Ngân hàng, Chứng khoán, Bất động sản, Công nghệ & Viễn thông, plus
    the movers it lists) and the existing five, grouped under Vietnamese
    sector names as the design writes them. Prices come from the VCI
    adapter (real when reachable), fundamentals are seeded approximations
    as before. Sector names switch to Vietnamese everywhere (Main,
    Portfolio allocation) — the design never shows English sectors.
15. **Full heatmap dashboard (`/heatmap`)**: a two-level treemap (sectors,
    then tickers) sized by market cap, coloured with the panel's own
    formula; filters for exchange (Tất cả/HOSE/HNX/UPCOM) and period
    (1 ngày / 1 tuần / 1 tháng / 3 tháng, computed from real daily bars);
    "Kích thước: vốn hoá / bằng nhau"; a Cổ phiếu / Crypto switch (crypto
    uses categories and the design's ±12 % colour range); a side panel
    with per-sector average and market breadth (tăng/giảm/đứng); hover
    details; click → Detail. The nav's "Bản đồ nhiệt" item becomes built
    in both modes.

## Scope left out, flagged

- Crypto Quant (the design's "Hỏi Quant: coin nào có RSI 4h dưới 30")
  — Quant's screen executor is stock-only; the card links to Quant.
- A crypto Portfolio page (not designed).
- Live WebSocket prices — still pull-based like the rest of the app.

## Verification plan

- Go unit tests: Binance adapter against a fake server in Binance's real
  wire format (ticker/klines/depth + a 400 for an unlisted symbol →
  fallback), fallback/caching, heatmap/overview maths, fractional fills,
  wrong-market rejection, crypto fee.
- curl: pairs, detail, bars (1h: continuous, no weekend gaps), order
  book, crypto order on a crypto wallet, 409 for a stock ticker on it,
  crypto replay start/advance/order/end, stock heatmap with period and
  exchange filters.
- Newman (new Crypto folder), tsc/eslint/build.
- Real browser (production build): STK/CRY switch both ways, Crypto
  overview, open a crypto wallet, BTC/USDT detail with the 8-level book,
  place a real (paper) order with a fractional amount, see it on the
  overview, crypto Replay session, and the full heatmap in both markets
  with every filter.

## Verification (done)

Everything below ran on mock data: this sandbox's network policy refuses
both `data-api.binance.vision` and `trading.vietcap.com.vn` (403 on
CONNECT), so the live Binance and VCI paths were exercised only against
fake servers in Binance's real wire format. To see real prices, allow
those two domains (environment settings → Network access) or run it
anywhere with normal internet; `MARKET_DATA_SOURCE=vci` switches both
markets to live data, and each falls back to the mock per call.

- **Go** (`go test ./...`, all pass): crypto (Binance ticker/klines/depth
  formats, 400 → fallback with `source: "mock"`, 10 s cache, 30 s circuit
  breaker, 24/7 mock continuity, order book cumulative totals, heatmap and
  overview maths, QuoteRouter); order (crypto wallet defaults, markets
  can't mix, whole-share stocks vs 8-decimal crypto, 0.10 % crypto fee,
  fractional full sell, OCO needs both prices); screener (sector grouping
  and order, breadth, 1D/1W/1M/3M period maths, exchange filter).
- **curl**: 24 pairs; BTC detail with ATH and 30-day volatility; 72
  continuous hourly bars across a weekend; 8-level book; overview
  dominance; crypto heatmap in 4 categories; stock heatmap 1M/HOSE in 9
  sectors; bad period → 400; unknown pair → 404; a 10,000 USDT wallet
  with the stock default untouched; 0.025 BTC buy with a 1.41 USDT fee;
  FPT into the wallet → 409 `wrong_market`; OCO queued with both prices;
  crypto Replay (resolution 60, step 3600 s, 10,000 USDT); fractional
  stock Replay quantity → 400.
- **Newman**: new Crypto folder (14 requests with assertions) plus heatmap
  period/exchange examples — 62 requests, 31 assertions, 0 failures.
- **tsc / eslint / `npm run build`**: clean.
- **Browser** (Playwright on the production build, no console or page
  errors):
  - `/heatmap` in all 4 exchanges and 4 periods, both size modes, and
    crypto; a pairwise check of tile rectangles found 0 overlaps in every
    view.
  - Signup with 10.000 USDT lands on `/crypto` with a 10,000 USDT wallet.
  - Cổ phiếu ⇄ Crypto switch works both ways.
  - BTC/USDT detail: 8-level book, 4 giờ default timeframe, 1 giờ switch.
  - A 0,025 BTC market buy fills, and the position shows on the overview.
  - The account menu lists the wallet in USDT and a stock portfolio in ₫,
    with one active row per market. Creating a stock portfolio from it
    goes to `/portfolio`; choosing the wallet goes to `/crypto`.
  - Crypto Replay on ETHUSDT: hourly candles, a 0,05 ETH buy, stepping
    forward.
- **Fixed after reviewing the screenshots:**
  - The chart's last-price tag clipped five-digit USDT prices; it now
    widens to fit.
  - The BTC stat card truncated its price.
  - Small treemap tiles clipped their labels; they now show only the
    ticker.
  - The Replay fills table's Giá and Ghi chú columns touched.

### Known gaps carried forward

- **Trading fees are still not deducted from cash.** This is the
  pre-existing gap from phase-e.md, not new, but crypto makes it easier to
  see: after a 0.025 BTC buy the wallet shows 9.999,99 USDT rather than
  about 9.998,6. The fee is computed and shown on the order. Fixing it
  touches stock accounting too, so it gets its own change.
- The live Binance/VCI round trip is unverified here (network policy, see
  above).
- Crypto Quant, a crypto Portfolio page, and live WebSocket prices stay
  out of scope, as listed above.

