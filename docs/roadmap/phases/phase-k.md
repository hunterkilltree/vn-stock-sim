# Phase K — Finish V1: order matching, fees, watchlist, backtest page

Branch `phase-k-finish-v1`, from master `f9b077d` (Phases A–J merged).

FULL-APP-PLAN.md's Phase K is a list of stretch items (screener,
journal, strategy builder, settings sections, English). Checking master
first showed that four **V1** features from
`vn-stock-sim-version-highlights.md` are still missing or wrong. CLAUDE.md's
build rule is V1 before later versions, so this phase finishes those
first; the stretch list moves to Phase L.

| Gap on master | Evidence |
|---|---|
| Limit / stop / ATC / OCO orders never fill | `order.Service.Create` stores them as `queued`; nothing reads them again (RESUME.md "Future work": "stored as queued forever") |
| Trading fees are never taken out of cash | `Order.Fee` is computed, but `portfolio.MemoryStore.ApplyFill` moves only `price × quantity` (open since phase-e.md) |
| No watchlist on screen | `/api/v1/watchlist` exists; only Quant's "add to watchlist" button calls it, and nothing lists it |
| No backtest page | `/api/v1/backtests` exists; only Quant's strategy card uses it ("Kiểm thử lịch sử" in the nav is WILL) |

Out of scope, unchanged: persistence (in-memory stores; separate
Postgres work), the WebSocket price feed, the TradingView Charting
Library.

## Decisions

### Fees
1. **The ledger charges the fee.** `ApplyFill(portfolioID, sym, side,
   qty, price, fee)`:
   - A buy needs `price·qty + fee` in cash and deducts it.
   - A sell credits `price·qty − fee`.
   - Average cost stays the pure price, as the designs show it ("vốn
     44,80"); the fee lives in cash.
   - So equity, `Stats.TotalPnl`, the equity curve and NAV all include
     fees. Per-trade journal % stays price-based.
2. **Rates are unchanged**: 0.15 % stocks, 0.10 % crypto, charged on
   every fill, including matched orders and Replay.
3. **Replay pays fees too.** Its session NAV/P&L (`replayAccounting`)
   subtracts each fill's fee, so the session score and the portfolio
   agree.

### Order matching
4. **Match against intraday bars, not a polled price.** A matcher scans
   queued orders against the **5-minute bars** since the order was
   placed: stocks from `market.Service`, pairs from `crypto.Service`.
   These are the same bars the Detail chart's "1 ngày" / "5 phút" view
   shows. A touch between checks is not missed, and a fill can always be
   pointed to on the chart. The first bar considered is the first that
   starts at or after the order's creation time.
5. **Fill rules per bar** (standard backtest conventions; a gap fills at
   the bar's open):

   | Type | Buy | Sell |
   |---|---|---|
   | limit (LO) | low ≤ P → min(P, open) | high ≥ P → max(P, open) |
   | stop | high ≥ P → max(P, open) | low ≤ P → min(P, open) |
   | OCO | limit leg low ≤ P, stop leg high ≥ S | limit leg high ≥ P, stop leg low ≤ S |

   If both OCO legs trigger in the same bar, the stop leg wins (the
   conservative assumption). The order records which leg filled in
   `triggeredBy: "limit" | "stop"`.
6. **Marketable on arrival fills at once**, at the current price shown on
   the ticket (a buy limit at or above it, a sell limit at or below it,
   or a stop already crossed), exactly like a market order. A user who
   types a limit above the price expects a fill, not a wait.
7. **ATC** (stocks only) fills at the day's closing price once the
   14:45 (Asia/Ho_Chi_Minh) close has passed on a weekday, at the
   symbol's last price then. An ATC placed after 14:45 waits for the
   next weekday's close.
8. **A fill that can't be booked** (not enough cash or shares at that
   moment) becomes `status: "rejected"` with `rejectReason`. Queued buys
   don't reserve cash (a documented simplification). A queued sell is
   checked against the position when it triggers.
9. **The matcher runs in the background** every 20 s
   (`ORDER_MATCH_INTERVAL`, default `20s`). The loop is
   `Service.MatchPending(now)`; tests call it directly with fixed bars.
   An order placed mid-bar needs no immediate check: nothing new can have
   touched it yet, and decision 6 covers orders already marketable.
10. **No trading-hours gate**: the data source decides. VCI's real
    intraday bars only exist during sessions; the mock's continuous price
    and crypto trade around the clock.

### Watchlist
11. **The API returns what a list needs:**
    - Items gain `companyName` and `exchange`.
    - Adding an unknown symbol returns 404 (today anything is stored).
    - The watchlist takes crypto pairs as well as stocks, through the same
      quote router the orders use.
12. **UI** (no design exists, so this extends existing panels):
    - A bookmark toggle on Detail (stock and pair), the button
      Mobile-Detail.dc.html draws, placed beside the header on desktop.
    - A "Danh sách theo dõi" card on Main, in the right column under the
      positions and on phones under the gainers: symbol, name, price,
      change %, remove.
    - Rows link to Detail. The empty state explains the bookmark.

### Backtest page
13. **The API adds what a results page needs** (fields only added, so
    Quant's use is unchanged):
    - `equity` points per bar and `trades` (entry/exit date and price,
      % change);
    - `benchmarkReturnPercent` (buy-and-hold over the same bars);
    - `from`, `to`, `startingCapital`, `params`, `id`.
    - Fees stay out of the backtest engine for now, as before; the page
      says so.
14. **`/backtest`** ("Kiểm thử lịch sử" becomes built, stock mode only;
    nothing is designed, so it follows Quant's strategy card and
    Portfolio's panels):
    - **Form:** symbol, rule (EMA cắt nhau: fast/slow; RSI hồi phục:
      period/entry/exit/stop-loss/SMA trend), date range presets (1/3/5
      years) and starting capital.
    - **Results:** KPI row (return vs buy-and-hold, trades, win rate, max
      drawdown, profit factor), equity curve with the buy-and-hold line,
      trades table, and a list of this session's previous runs.
    - Quant's strategy card gets an "Mở trong Kiểm thử" link carrying the
      rule, and Detail links "Kiểm thử mã này" to it.
    - Responsive from the start (Phase J's phone shell).

### Frontend around matching
15. **Pending orders** show what each order waits for (limit/trigger
    price, OCO's two prices, ATC's 14:45). Under them, "Vừa xử lý" lists
    the latest five limit/stop/ATC/OCO orders that were filled (at what
    price, which OCO leg), rejected (with the reason in Vietnamese) or
    cancelled. A matched order also shows in positions and the journal
    like any fill. The crypto overview gets the same card for the wallet.

## Verification plan

- **Go unit tests:**
  - fee accounting (buy cost + fee, sell proceeds − fee, insufficient
    cash counts the fee);
  - every matching rule in the table (limit/stop both sides, gap fills
    at the open, OCO legs with a same-bar tie → stop);
  - marketable-on-arrival;
  - ATC before and after 14:45 and on weekends;
  - rejected on insufficient cash;
  - Replay NAV net of fees;
  - backtest trades/equity/benchmark;
  - watchlist unknown symbol → 404.
- **curl:**
  - place a limit near the price, watch it fill within one matcher tick;
  - cash reduced by value + fee;
  - OCO fills one leg;
  - watchlist add/list/remove;
  - a backtest with trades.
- **Newman:** new requests for matching, watchlist and backtest.
- **Browser (production build), desktop and phone:**
  - place a limit order, see it pending, then filled on Portfolio;
  - bookmark from Detail, see it on Main, remove it;
  - run a backtest from the page and from Quant's link.
- The Phase I desktop suite and the Phase J phone suite stay green.

## Changed while building

- **Unknown watchlist symbols**: pairs must be given by their full name
  (`BTCUSDT`). The quote router is strict on purpose, so a coin name can
  never shadow a stock ticker. Adding just `BTC` returns 404.
- **The backtest page checks the ticker exists** before running. The
  mock market generates bars for any string, so before this an unknown
  ticker got a result.
- **Detail's "Replay mã này"** was a disabled button on desktop although
  Replay has accepted `?symbol=` since Phase H. It is now a link, next to
  the new "Kiểm thử mã này". Between 1024 and 1280 px the header buttons
  show icons only (labels kept as `aria-label`) and the indicator chips
  are hidden, so nothing wraps.
- **Crypto pairs are not in the backtest page** (the crypto nav's
  "Kiểm thử lịch sử" stays WILL): the rules run on daily bars and crypto
  Replay/Quant were scoped as stock-first.

## Verification (done)

All against the production build and a backend on mock data (the sandbox
can't reach VCI/Binance), with `ORDER_MATCH_INTERVAL=5s`.

- **Go tests**: 41 in total, all passing.
  - Fees: order wallet cash net of fees on buy and sell; Replay NAV net
    of fees for stock and crypto.
  - Matcher (`order/matcher_test.go`):
    - the full rule table, including gaps filling at the open and a
      same-bar OCO tie going to the stop leg;
    - a limit that ignores a bar before the order and fills on the
      touching bar with the fee and that bar's time;
    - marketable-on-arrival (limit and an OCO with its stop crossed);
    - an OCO filling one leg only;
    - rejected when cash ran out;
    - cancelled never fills;
    - ATC timing (same day, after the close, Friday evening, weekend).
  - Watchlist: canonical symbols, names, 404, pairs.
  - Backtest: equity per bar, trade list, benchmark, open trade at the
    end.
- **curl**:
  - A market buy leaves cash = 100,000,000 − 5,208,205 − 7,812.3 (fee).
  - A buy limit above the price fills at once at the price; a limit
    without a price → 400.
  - A limit at 52,050 queued at 01:41:40 filled on the 01:45 bar at
    52,050 with fee 7,807.5.
  - An ATC placed in the morning waits for 14:45; an OCO waits.
  - Watchlist: unknown → 404; FPT and BTCUSDT listed with names;
    remove → 204.
  - Backtest: 3-year RSI run with 1,096 equity points and 13 trades; the
    list omits equity.
- **Newman**: 67 requests, 39 assertions, 0 failures. New requests cover
  the limit without a price, a marketable limit, an unknown watchlist
  symbol, a crypto pair in the watchlist, and the backtest detail.
- **Browser** (`phase-k-verify.js`, `phase-k-match.js`), no page errors:
  - Watchlist: bookmark FPT and ETH/USDT from their Detail pages; both
    listed on Main with names and prices; removing ETH leaves 1; the
    bookmark persists.
  - Two limit orders show as pending with their limit prices; cancelling
    one moves it to "Vừa xử lý · Đã huỷ".
  - A buy stop at 52,18 (price 52,08) placed from the ticket filled in
    the background 269 s later. It filled at 52,54, the bar's open, since
    the price gapped past the trigger, and appeared under "Vừa xử lý ·
    Đã khớp" and in positions.
  - A limit 0.15 % below the price correctly did *not* fill within 7
    minutes: the lowest bar since was 52,066.
  - `/backtest` from Detail's link with the symbol prefilled: the RSI run
    shows 5 KPIs, the equity curve against buy-and-hold and 12 trades;
    fast ≥ slow EMA is refused; an EMA run adds to history; `NOPE` →
    "Không tìm thấy mã NOPE."
  - Quant's strategy card "Xem kết quả đầy đủ" opens the run on
    `/backtest`.
  - Phone (390): backtest layout 390 wide with results above the form;
    Main watchlist visible; the Detail bookmark is pressed.
- **Regressions**: the Phase I desktop suite and Phase J phone flows pass.
  The phone audit has 0 overflowing views at 390 × 844 and 1024 × 768
  across 17 routes, guest and signed in.

### Still open

- Queued buys don't reserve cash; a queued sell isn't checked against the
  position until it triggers (decision 8).
- Backtests ignore fees (the page says so).
- Everything is still in memory (Postgres is separate work).

