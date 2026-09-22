# VN Stock Sim — screen inventory

40 artboards. Every screen exists in Vietnamese and English; the English file
is the same name with `-EN`. Desktop is 1440×960, phone is 390×844.

## Stock market — desktop

| File | Screen | What's on it |
|---|---|---|
| `Main.dc.html` | Market overview | Index cards with sparklines (VN-Index, VN30, HNX, UPCOM), sector heatmap, top gainers/losers, paper account summary, open positions, Replay promo, Quant prompt |
| `Detail.dc.html` | Ticker detail — FPT | Price header with ceiling/ref/floor, timeframe pills, candles + SMA 20/50, volume, RSI(14), MACD, paper order ticket, 3-level order book, fundamentals |
| `Replay.dc.html` | Replay Mode | 44/120 candles revealed, masked future, entry/exit markers, stop line, playback controls, session orders table, session result, provisional skill score |
| `Portfolio.dc.html` | Paper trading | 5 KPI tiles, equity curve vs VN-Index, 6-row holdings table, sector allocation, pending orders, trade journal stats |
| `Quant.dc.html` | Quant assistant | Chat thread with parsed-condition chips, 7 matching tickers, draft strategy with backtest preview, caution block, results panel with tabs |
| `Quant-Chart.dc.html` | Quant on the chart | 142-candle selection with handles, radial strategy wheel (6 wedges), payload panel, 5-step model pipeline, past runs on the same selection |
| `Settings-AI.dc.html` | Settings › AI model | Provider radio cards, API key, model and timeout, creativity slider, two permission switches, data-sent toggles, usage meter, test-before-save |

## Stock market — phone

| File | Screen |
|---|---|
| `Mobile-Market.dc.html` | Market: VN-Index hero, index chips, 3-column heatmap, top gainers, tab bar |
| `Mobile-Detail.dc.html` | Ticker: price, band chips, timeframes, candles + SMA, RSI strip, stats, buy/sell bar |
| `Mobile-Replay.dc.html` | Replay: progress, masked chart, large step controls, session stats, buy/sell bar |
| `Mobile-Portfolio.dc.html` | Portfolio: NAV hero with equity curve, 2×2 KPIs, holdings list |
| `Mobile-Quant.dc.html` | Quant: condition chips, inline results, strategy card, composer with voice input |
| `Mobile-Settings.dc.html` | AI model settings: provider row, API key, model, creativity, data switches, usage |

## Crypto mode

| File | Screen | Differences from stocks |
|---|---|---|
| `Crypto-Main.dc.html` | Crypto market overview | BTC/ETH/market-cap/dominance cards, category heatmap incl. Vietnamese projects, 24h movers, USDT wallet — **no ceiling/floor legend**, 24/7 status |
| `Crypto-Detail.dc.html` | BTC/USDT detail | 24h high/low/volume/ATH chips instead of price bands, intraday timeframes, Market/Limit/Stop/OCO, 8-level order book with depth bars, supply facts |
| `Crypto-Replay.dc.html` | Crypto Replay | Hourly candles, May 2021 crash, timestamps not dates, stricter risk scoring |

Crypto phone screens do not exist yet.

## Accounts

| File | Screen |
|---|---|
| `Login.dc.html` | Sign in — split layout, email + password, guest mode, link to sign up |
| `Signup.dc.html` | Create account — name/email/password with rule checklist, starting capital, market preference, simulator acknowledgement |
| `Account-Menu.dc.html` | Account menu popover (400×600 component) — identity, 3 paper portfolios with switcher, new portfolio, profile/settings/help, log out |
| `Mobile-Login.dc.html` | Sign in on phone |

---

## Not designed yet — carries the WILL badge

These appear in navigation but have no screen. Anything built for them needs
a design pass first.

- **Bộ lọc cổ phiếu / Stock screener** — filter UI as a screen (Quant covers
  the conversational path only)
- **Bản đồ nhiệt / Heatmap** — full-screen version; only the dashboard panel
  exists
- **Xây chiến lược / Strategy builder** — the visual no-code rule builder
- **Kiểm thử lịch sử / Backtesting** — results screen behind "Open in
  Backtest"
- **Sổ giao dịch / Trade journal** — full screen; only the summary panel
  exists
- **Settings sections** — Account, Market data, Paper account, Replay &
  scoring, Notifications, Appearance, Privacy (only AI model is drawn)
- **Profile & security**, **Help** — from the account menu
- **Crypto**: paper trading and Quant currently borrow the stock screens

Also undesigned, and worth deciding early:

- Forgot-password / reset flow
- The sign-in prompt a guest meets at their first paper order (a modal over
  the order ticket is the natural shape)
- Empty states — new account with no positions, no trades, no Replay history
- Error and loading states for every chart panel
