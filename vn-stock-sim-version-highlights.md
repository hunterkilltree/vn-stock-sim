# VN Stock Sim — Version Highlights

**VN Stock Sim** — Analyze, simulate, and backtest Vietnamese stocks in one place.

This document explains what VN Stock Sim will actually contain, version by version. Each version builds on the one before it, so by the time you reach the final version, every module from the product vision has a home. The goal at every stage is the same: give Vietnamese investors, students, and quant developers a place to study the market and practice trading it without financial risk — before adding anything flashy like AI or order flow.

---

## Version 1 (MVP) — "See the market, trade it on paper"

This is the first thing real users touch. It has to work well on its own, even though it's the smallest version. Nothing here depends on data or features that come later.

**What a user can do:**

- **Create an account and log in.** Basic authentication and a user profile — the foundation everything else (watchlists, portfolios, trades) attaches to.
- **Browse a real Vietnamese stock database.** Every ticker on HOSE, HNX, and UPCOM, with company name, exchange, and sector, sourced from a licensed market-data provider (not scraped websites).
- **See historical daily prices.** Open, high, low, close, and volume (OHLCV) for each stock, going back far enough for meaningful charts and backtests.
- **Search for any stock instantly.** Type a ticker or company name and jump straight to it.
- **Open a stock detail page.** This is the centerpiece screen: current price and change, a candlestick chart, and key fundamentals (market cap, P/E, P/B, EPS, dividend yield).
- **Read a candlestick chart with 10 basic indicators.** Trend tools like SMA, EMA, and VWAP; momentum tools like RSI and MACD; volatility tools like Bollinger Bands — the standard toolkit every trader expects, layered directly on the chart.
- **Build a watchlist.** Save the tickers you're following and see their live price, change, and volume in one table.
- **Hold a virtual portfolio.** Starting cash, current holdings, market value, and profit/loss — always visible.
- **Paper trade with real money on the line — except it's fake.** Place buy and sell orders against live or historical prices and watch your virtual portfolio update.
- **Review your trade history.** Every simulated order you've placed, in one list.
- **Run a basic backtest.** Pick a simple rule (for example, an EMA crossover), a stock, a date range, and starting capital, and see how that rule would have performed: final capital, return, number of trades, and win rate.

**Why this is the right starting point:** it proves the two hardest and most valuable things — trustworthy Vietnamese market data, and a working paper-trading loop — before a single dollar (or đồng) is spent on AI, order flow, or an 800-indicator charting engine. Everything after this version is an enhancement, not a foundation.

---

## Version 2 — "Find opportunities, practice on history, learn from your trades"

Version 2 turns VN Stock Sim from "a place to look at stocks" into "a place to actively hunt for setups and train yourself."

**What's new:**

- **Stock screener.** Filter the entire market by fundamentals (market cap, P/E, ROE, revenue growth), by activity (trading volume), and by technicals (RSI range, price above EMA50, MACD bullish crossover). This is especially valuable in Vietnam, where there's no dominant existing screener tailored to HOSE/HNX/UPCOM.
- **Market heatmap.** A single visual view of the whole market (or a sector), sized and colored by performance, so you can spot what's moving at a glance.
- **Advanced chart drawing tools.** Beyond the basics — Fibonacci retracement and extension, rectangles, trend lines, text and arrow annotations — for real technical analysis directly on the chart.
- **Strategy Builder.** A no-code, visual way to define trading rules: "WHEN EMA20 crosses above EMA50 AND RSI(14) > 50 AND Volume > its 20-day average, THEN BUY." No programming required — just dropdowns and comparisons.
- **Strategy Library.** Save your strategies, revisit them, and (eventually) browse ones others have shared.
- **Replay Mode — the killer feature.** Pick a stock and a historical period, then step through it one candle at a time, seeing only the information that would have been available at that moment. At each step you decide: buy, sell, or hold. At the end, the system scores you — return, win rate, max drawdown — compared to the index. This is what makes VN Stock Sim a *training simulator*, not just another charting site: "Trade the past before you trade the future."
- **Trade Journal.** Every simulated trade automatically becomes a journal entry: what you bought, why (your stated reason), how it turned out, and how you felt about it (following your strategy, FOMO, fear, revenge trade). Over time this rolls up into personal performance analytics — your win rate, your best setup, your worst setup — so you can see your own patterns, not just the market's.
- **Alerts.** Get notified when a stock crosses a price level or a technical condition you've set.

**Why this matters:** these are the features that make people come back daily and actually improve as traders, rather than just checking prices once and leaving.

---

## Version 3 — "Let AI do the research and the strategy-writing for you"

Version 3 introduces **Quant**, the AI assistant, but only once the manual versions of these tools already exist and work — AI here accelerates a proven workflow, it doesn't invent an unproven one.

**What's new:**

- **AI Assistant (Quant).** Ask questions in plain language — "Find Vietnamese stocks where price is above EMA200, RSI is between 50 and 70, and volume is above its 20-day average" — and get a direct list of matching tickers.
- **Natural-language screener.** The screener from Version 2, but driven by conversation instead of filter sliders.
- **AI Strategy Builder.** Describe a strategy in a sentence ("buy when EMA20 crosses EMA50 and RSI is above 50") and Quant generates the entry/exit rules in the visual Strategy Builder format, ready to inspect or edit.
- **AI stock analysis.** A written, data-backed summary of a stock's technical and fundamental picture on demand.
- **Automatic backtesting.** Quant can generate a strategy and immediately run it through the backtesting engine, so you go from idea to results in one step.
- **Strategy optimization.** Automatically test variations of a strategy's parameters (different EMA lengths, different RSI thresholds) to find which combination performed best historically.
- **Strategy marketplace.** A place for users to publish and browse each other's strategies — turning the platform into a small community, not just a tool.

**Why this comes third, not first:** AI is most valuable once there's a strong manual foundation to accelerate. Building the AI layer before the screener, strategy builder, and backtester already exist would mean generating outputs into a vacuum with nothing solid to plug them into.

---

## Version 4 — "Professional-grade depth and platform growth"

Version 4 is where VN Stock Sim stops being just a strong simulator and starts approaching what a professional trading desk or a broader community platform would expect.

**What's new:**

- **Order flow.** See the real buying and selling pressure behind a price move, not just the price itself — the feature referenced in the project's own tagline.
- **Volume Profile and Market Profile.** Visualize *where* volume traded at each price level, revealing support/resistance zones that a plain candlestick chart hides.
- **Advanced portfolio analytics.** Deeper risk and performance metrics across your whole simulated portfolio, not just per-trade.
- **Broker integration.** An optional bridge from paper trading to a real brokerage account, for users who are ready to go live — while keeping simulation as the default, safe starting point.
- **Webhook automation.** Let a validated strategy trigger real actions automatically (for example, sending alerts or orders to external systems).
- **Mobile app.** Take watchlists, alerts, and portfolio tracking with you.
- **Social and community features.** Leaderboards and strategy sharing, turning individual practice into a competitive and social experience.

**Why this comes last:** these features assume a large, engaged user base and a mature core product. Building them earlier would mean investing heavily in infrastructure (real-time order flow data, broker APIs, mobile apps) before knowing whether the core simulation product has found its audience.

---

## How the versions connect

| Version | Core theme | Unlocks |
|---|---|---|
| 1 — MVP | Market data + paper trading | A usable Vietnam stock analysis and paper-trading platform |
| 2 | Discovery + training | Screener, Replay Mode, Trade Journal — the features that build skill |
| 3 | AI acceleration | Quant assistant on top of the manual tools from Version 2 |
| 4 | Professional depth | Order flow, broker integration, mobile, community |

Each version is fully usable on its own — a user gets real value from Version 1 alone. Nothing in a later version is required to make an earlier version work, which means the product can launch, get real feedback, and adjust before the more expensive and complex features (AI, order flow, broker integration) are built.

## A note on positioning

VN Stock Sim is **simulation-first**, not a brokerage. The one-line identity to keep in mind at every version: *"Trade the past before you trade the future."* LuxAlgo is a useful reference for UX and product structure (charting, indicators, backtesting, AI layered across pricing tiers), but VN Stock Sim's differentiation is being Vietnam-specific and simulation-first — its own branding, its own proprietary indicators (VN Trend, VN Smart Money, VN Breakout, etc.), and its own market-data licensing, never copied assets or algorithms.
