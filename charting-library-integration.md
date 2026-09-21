# Charting Integration — TradingView Charting Library

This documents how the frontend connects to TradingView's **Charting Library** (the advanced, self-hosted widget — not Lightweight Charts) for the main stock chart, and how it plugs into the Go backend already described in the tech-approach README.

---

## 1. Before writing any code: get access

The Charting Library is not on npm and not open source. It's a free-to-use but access-gated bundle: you request access through TradingView's GitHub form (`tradingview/charting_library`), and once approved you get a private repo with the library files (JS/CSS) to download and self-host inside the Next.js app (typically under `public/charting_library/`, loaded via a `<script>` tag rather than `import`). This is a one-time approval step with no cost for standard use, but it does need to happen before this integration can be built or demoed — worth kicking off in parallel with backend work since approval isn't instant.

This is the one real tradeoff against the original Lightweight Charts decision: Lightweight Charts is MIT-licensed and `npm install`-able with zero approval step. Charting Library buys a full symbol-search UI, multi-pane layouts, drawing tools, and built-in studies (Volume Profile, etc.) out of the box — at the cost of that access request and a heavier bundle.

---

## 2. Integration shape: custom Datafeed, not UDF

The Charting Library supports two ways to feed it data: **UDF** (a fixed HTTP protocol the library calls directly) or a **custom JavaScript Datafeed object** you hand to the widget constructor. Use the custom Datafeed — UDF is the older, more rigid option, and a hand-written Datafeed lets it call the Go REST/WebSocket API exactly as already designed (`MarketDataProvider`, `ws.Hub`) instead of forcing the backend to also expose a second, UDF-shaped API surface.

```
Next.js Client Component
  └─ new Datafeed(apiBaseUrl, wsUrl)   // implements IExternalDatafeed + IDatafeedChartApi
       ├─ REST calls  → Go /api/v1/...     (symbols, historical bars)
       └─ WebSocket   → Go ws.Hub          (live bars, live order-flow ticks)
  └─ new TradingView.widget({ datafeed, ...ChartingLibraryWidgetOptions })
```

The Datafeed is a thin adapter. It owns no state of its own beyond active subscriptions — it translates between the Charting Library's shapes (`SymbolInfo`, `Bar`) and the Go API's existing shapes (`Symbol`, `OHLCV`), which already line up closely with the `MarketDataProvider` interface in the backend README.

---

## 3. Method-by-method mapping

| Datafeed method | Called when | Backed by |
|---|---|---|
| `onReady(callback)` | Widget init | Static config: supported resolutions (`1, 5, 15, 60, 1D`), exchanges (`HOSE`, `HNX`, `UPCOM`), no server round-trip needed |
| `resolveSymbol(symbolName, onResolve)` | User picks/loads a symbol | `GET /api/v1/symbols/{symbol}` → map to `LibrarySymbolInfo` (timezone `Asia/Ho_Chi_Minh`, session `0900-1130,1300-1500`, `pricescale` from the symbol's tick size) |
| `searchSymbols(userInput, onResult)` | Symbol search box | `GET /api/v1/symbols?q=` (existing symbol list endpoint, paginated per README §2) |
| `getBars(symbolInfo, resolution, periodParams, onResult)` | Initial load + panning back in history | `GET /api/v1/market/bars?symbol=&resolution=&from=&to=` → directly reuses `MarketDataProvider.GetHistoricalBars` |
| `subscribeBars(symbolInfo, resolution, onTick, listenerGuid)` | Chart goes live for a symbol | Open (or reuse) a WebSocket connection to `ws.Hub`, subscribe to that symbol's channel; each incoming tick/candle update is translated to a `Bar` and passed to `onTick` |
| `unsubscribeBars(listenerGuid)` | Symbol changed or chart unmounted | Unsubscribe that symbol's channel on `ws.Hub`; close the socket only when no symbols remain subscribed |
| `getServerTime(callback)` (optional) | If `supports_time` is enabled | `GET /api/v1/time` — keeps the chart's "last updated" clock honest against server time rather than the browser's |

`getQuotes` / `subscribeQuotes` / `subscribeDepth` (watchlist, DOM data) are **not required for the MVP chart** — skip them until Version 2's watchlist and Version 4's broker/DOM features actually need them, matching the phased build sequence already in the README.

All callbacks must fire asynchronously (`setTimeout(() => cb(data), 0)`) — the library documents this as a hard requirement to avoid a synchronous call-stack blowup on symbol switch.

---

## 4. Real-time path reuses the existing broadcaster

The backend README already plans a `Market Data Worker` that fetches live prices and a `Price Broadcaster` (`ws.Hub`) that fans them out to connected clients. The Datafeed's `subscribeBars` is just another consumer of that same hub — no new backend component, just a new message type on the existing WebSocket connection (`{"type": "bar_update", "symbol": ..., "bar": {...}}`) that the frontend Datafeed listens for and re-emits into the chart via `onTick`.

---

## 5. Order flow: the one gap to design around

The Charting Library's built-in studies (Volume Profile, etc.) don't include footprint / bid-ask-at-price order-flow rendering — that's a custom-drawing problem either way. Two paths, and the choice can be made later without touching the Datafeed above:

- **Custom Study / drawing primitive** inside the Charting Library — keeps everything in one widget, but the library's custom-study drawing API is more constrained (harder to draw per-price-level bid/ask split text inside each candle).
- **A second, lower-level canvas (Lightweight Charts or raw canvas) rendered as a synchronized overlay pane** beneath or beside the main chart, sharing the same time axis — more drawing freedom for footprint-style visuals, at the cost of keeping two chart instances in sync (crosshair, zoom, pan).

Given "orderflow built into every chart" is the product's core differentiator, this is worth a short spike before committing — recommend prototyping the footprint overlay in isolation (Version 4 timing per the README, but worth validating the rendering approach much earlier since it affects whether Charting Library is even the right primary widget).

---

## 6. Frontend placement

Per the existing rule in the tech-approach README — anything live/ticking must be a Client Component — the whole chart (widget + Datafeed + WebSocket connection) lives in one `"use client"` component, constructed once on mount and updated imperatively through the widget's own API, never re-rendered by React on a tick.

---

## 7. Dependency inversion: the service depends on a port, TradingView specifics live in an adapter

This chart feature should follow the same shape already used for `order` and `ai` in the backend README: a `Service` that holds the rules, sitting behind a small interface it defines itself, with anything vendor-specific — a market-data vendor, TradingView's shape requirements — pushed out into an adapter that implements that interface. The Service never imports a vendor SDK or knows TradingView exists.

**Backend (Go) — the port is `MarketDataProvider`, already defined in the tech README:**

```go
// internal/market/service.go
package market

type Service struct {
    data MarketDataProvider // the Service depends on this interface, not a vendor client
}

func NewService(data MarketDataProvider) *Service {
    return &Service{data: data}
}

func (s *Service) GetBars(ctx context.Context, symbol string, tf Timeframe, from, to time.Time) ([]OHLCV, error) {
    // rules live here: caching, gap-filling, session-hours clipping — none of it vendor-specific
    return s.data.GetHistoricalBars(ctx, symbol, tf, from, to)
}
```

```go
// internal/market/adapter/ssi/adapter.go
package ssi

type Adapter struct{ client *ssi.Client } // implements market.MarketDataProvider

func (a *Adapter) GetHistoricalBars(ctx context.Context, symbol string, tf market.Timeframe, from, to time.Time) ([]market.OHLCV, error) {
    raw, err := a.client.FetchCandles(ctx, toSSISymbol(symbol), toSSIResolution(tf), from, to)
    if err != nil {
        return nil, err
    }
    return toDomainBars(raw), nil // translate SSI's shape into the domain's OHLCV
}
```

Swapping market-data vendors, or dropping in a mock adapter for local dev and tests, means writing a new adapter — the `Service`, the `Handler` above it, and the Charting Library integration below it never change. This mirrors the `order` and `ai` features' "fits into" arrows in the architecture diagram: draw `market` the same way, with `Service` → `MarketDataProvider` (interface) ← `ssi.Adapter` (or `mock.Adapter`).

**Frontend (TypeScript) — the direction flips, but it's the same pattern:**

The custom Datafeed from §2 is itself a textbook Adapter — just pointed the other way. Instead of your domain defining a port that a vendor adapts to, TradingView's library defines the interface it demands (`IDatafeedChartApi`), and `TVDatafeedAdapter` is the thing that makes your own API client speak that shape:

```ts
// lib/charting/tv-datafeed-adapter.ts
class TVDatafeedAdapter implements IDatafeedChartApi, IExternalDatafeed {
  constructor(private api: StockApiClient, private ws: PriceSocket) {}

  getBars(symbolInfo, resolution, params, onResult) {
    this.api.getBars(symbolInfo.name, resolution, params.from, params.to)
      .then(bars => onResult(toTVBars(bars)))     // translate our OHLCV into TradingView's Bar[]
      .catch(err => onResult([], { noData: true }));
  }
  // ...resolveSymbol, subscribeBars, etc. — same translate-and-delegate shape
}
```

Same principle both times — nothing above the adapter (the `Service` in Go, the rest of the React app in TS) should know which vendor or which charting library it's ultimately talking to. Only the adapter's file changes if either one is swapped out later.
