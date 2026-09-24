# VN Stock Sim — API Spec Blueprint

This document describes the *approach* to the REST API, not a finished contract. It follows the same mental model as Go's own ["Developing a RESTful API with Go and Gin"](https://go.dev/doc/tutorial/web-service-gin) tutorial — design the endpoints first, keep each handler small and obvious, bind JSON in and out with Gin — scaled up to fit the feature-based, Handler → Service → Adapter layout already set in the tech-approach README, instead of the tutorial's single flat `main.go`.

Scope for this pass: Version 1 (MVP) plus the market-data endpoints the TradingView Charting Library integration needs.

---

## 1. Design endpoints first, one resource at a time

The Gin tutorial's first and most important step is deciding the endpoints before writing any handler code — `GET /albums`, `POST /albums`, `GET /albums/:id`. Same discipline here, one row per resource:

| Resource | Endpoints | Feature package |
|---|---|---|
| Auth | `POST /auth/register`, `POST /auth/login`, `GET /auth/me` | `internal/auth` |
| Symbols | `GET /symbols`, `GET /symbols/:symbol` | `internal/symbol` |
| Market data | `GET /market/bars`, `GET /market/indicators`, `GET /time` | `internal/market` |
| Watchlist | `GET /watchlist`, `POST /watchlist`, `DELETE /watchlist/:symbol` | `internal/watchlist` |
| Portfolio | `GET /portfolio`, `GET /portfolio/positions` | `internal/portfolio` |
| Orders | `GET /orders`, `POST /orders`, `GET /orders/:id`, `POST /orders/:id/cancel` | `internal/order` |
| Backtest | `GET /backtests`, `POST /backtests`, `GET /backtests/:id` | `internal/backtest` |

Every path is versioned under `/api/v1`, per the tech-approach README's existing rule.

`Symbols` and `Market data` are read-heavy and unauthenticated by design (they back the stock browser and the chart before login); everything else requires the bearer token from `/auth/login`.

---

## 2. Where Gin fits, and where it stops

Gin's role is deliberately narrow: routing, path/query param extraction, and JSON binding — exactly what the tutorial uses it for. It does **not** hold business logic. A route registration looks like the tutorial's `router.GET("/albums", getAlbums)`, just grouped per feature instead of listed flat in `main.go`:

```
cmd/api/main.go
  router := gin.Default()
  v1 := router.Group("/api/v1")
  auth.RegisterRoutes(v1, authService)
  symbol.RegisterRoutes(v1, symbolService)
  market.RegisterRoutes(v1, marketService)
  ...
```

Each feature owns a `RegisterRoutes` function and its own `gin.HandlerFunc`s. A handler's job stops at: parse the request, call the Service, write the response — the same three steps the tutorial's `getAlbumByID` does, just with the "get the data" step delegated to a `Service` behind an interface instead of a package-level slice. That Service → interface → Adapter shape is exactly what's already documented for `order`, `ai`, and `market` — this doc doesn't introduce a new pattern, it just says where Gin sits on top of it: strictly at the Handler layer, never below it.

---

## 3. Conventions every endpoint follows

- **Versioning:** every route lives under `/api/v1`.
- **Response envelope:** a list endpoint returns `{ "data": [...], "meta": { page, pageSize, totalItems, totalPages } }`; a single-resource endpoint returns the resource directly.
- **Errors:** a consistent shape everywhere — `{ "code": "insufficient_funds", "message": "..." }`, or `{ "code": "validation_error", "message": "...", "fields": [{ "field": "quantity", "reason": "must be greater than 0" }] }` for input validation failures. Gin's `c.ShouldBindJSON` error is translated into this shape at the handler boundary, never passed through raw.
- **Pagination:** `page` (default 1) and `pageSize` (default 20, max 100) query params on every list endpoint.
- **Auth:** `Authorization: Bearer <JWT>`, checked by a Gin middleware registered once on the `v1` group rather than per-route, except on the handful of routes above that are explicitly public.
- **Input validation:** happens at the handler boundary before the Service is ever called, per the README's existing "never trust client input" rule.

---

## 4. Endpoint-by-endpoint shape

### Auth

| Method & path | Request body | Response |
|---|---|---|
| `POST /auth/register` | `email, password, displayName` | `201` → `{ accessToken, expiresIn, user }` |
| `POST /auth/login` | `email, password` | `200` → `{ accessToken, expiresIn, user }` |
| `GET /auth/me` | — | `200` → `User` |

### Symbols

| Method & path | Query params | Response |
|---|---|---|
| `GET /symbols` | `q`, `exchange`, `page`, `pageSize` | paged list of `Symbol` |
| `GET /symbols/:symbol` | — | `SymbolDetail` (adds price, market cap, P/E, P/B, EPS, dividend yield) |

`GET /symbols` backs both the stock-database browser and the Charting Library Datafeed's `searchSymbols`; `GET /symbols/:symbol` backs the stock detail page and `resolveSymbol`. One endpoint, two consumers — no TradingView-specific route needed.

**`GET /symbols` response:**

```json
{
  "data": [
    {
      "symbol": "VNM",
      "companyName": "Vietnam Dairy Products JSC",
      "exchange": "HOSE",
      "sector": "Consumer Staples",
      "tickSize": 100
    }
  ],
  "meta": { "page": 1, "pageSize": 20, "totalItems": 1732, "totalPages": 87 }
}
```

**`GET /symbols/:symbol` response:**

```json
{
  "symbol": "VNM",
  "companyName": "Vietnam Dairy Products JSC",
  "exchange": "HOSE",
  "sector": "Consumer Staples",
  "tickSize": 100,
  "lastPrice": 68500,
  "change": 500,
  "changePercent": 0.74,
  "marketCap": 143200000000000,
  "peRatio": 16.2,
  "pbRatio": 4.1,
  "eps": 4228,
  "dividendYield": 5.8
}
```

Every field above is a plain domain fact — nothing TradingView-shaped leaks into this response. The Charting Library wants a different, richer shape (`LibrarySymbolInfo`: `pricescale`, `minmov`, `timezone`, `session`, `has_intraday`, `supported_resolutions`, `volume_precision`, `data_status`, ...), and that translation happens entirely inside `TVDatafeedAdapter.resolveSymbol` on the frontend, not here:

| This response field | → | `LibrarySymbolInfo` field | How it's derived |
|---|---|---|---|
| `symbol` | → | `name`, `ticker`, `full_name` | passthrough |
| `exchange` | → | `exchange`, `listed_exchange` | passthrough |
| `tickSize` (VND) | → | `pricescale`, `minmov` | `pricescale = 1 / tickSize` when `tickSize < 1`, otherwise `pricescale = 10 / tickSize` for VND's whole-đồng ticks; `minmov` stays `1` |
| *(not returned — static per exchange)* | → | `timezone` | hardcoded `"Asia/Ho_Chi_Minh"` in the adapter |
| *(not returned — static per exchange)* | → | `session` | hardcoded `"0900-1130,1300-1500"` in the adapter (HOSE/HNX/UPCOM trading hours) |
| *(not returned — fixed list)* | → | `supported_resolutions` | hardcoded `["1","5","15","60","1D"]` in the adapter, matching `/market/bars`' `resolution` enum |

This keeps the backend vendor-neutral (per the `MarketDataProvider` dependency-inversion design) — it never needs to know TradingView exists — while still giving the adapter every raw fact it needs to build a compliant `LibrarySymbolInfo` without a second round-trip.

### Market data

| Method & path | Query params | Response |
|---|---|---|
| `GET /market/bars` | `symbol`, `resolution` (`1,5,15,60,1D`), `from`, `to` (unix seconds) | ascending array of `{ time, open, high, low, close, volume }` |
| `GET /market/indicators` | `symbol`, `resolution`, `indicator` (`sma,ema,rsi,macd,bollinger,vwap`), `period`, `from`, `to` | array of `{ time, value }` or `{ time, values: {...} }` for multi-line indicators |
| `GET /time` | — | `{ timestamp }` — server clock, for the Datafeed's optional `getServerTime` |

`GET /market/bars` is the same endpoint whether it's serving the initial chart paint or the Charting Library Datafeed's `getBars` — see `charting-library-integration.md` for how the frontend's `TVDatafeedAdapter` calls it.

**`GET /market/bars` response:**

```json
{
  "data": [
    { "time": 1758067200, "open": 68000, "high": 68800, "low": 67900, "close": 68500, "volume": 1245300 },
    { "time": 1758153600, "open": 68500, "high": 69200, "low": 68300, "close": 69000, "volume": 987600 }
  ]
}
```

The field names (`time, open, high, low, close, volume`) were chosen to match the Charting Library's own `Bar` type exactly, so `TVDatafeedAdapter.getBars` is a near-passthrough — deserialize and hand to `onResult(data)`, no field renaming. Two contract details the adapter depends on, so they're pinned here explicitly:

- **Ascending order, oldest first.** The Datafeed API requires this; the backend must guarantee it rather than leaving sort order to the adapter.
- **Empty range means "no data," not an error.** When `from`/`to` covers a period with no trading data (e.g. before the symbol's listing date), the endpoint returns `200` with `"data": []`, never a `404`. The adapter is what translates that into the Datafeed's specific no-data shape — `onResult([], { noData: true })` — the backend just stays silent and honest about there being nothing there.

**`GET /market/indicators` response** (single-line indicator, e.g. `sma`):

```json
{ "data": [ { "time": 1758067200, "value": 68120.5 }, { "time": 1758153600, "value": 68340.2 } ] }
```

**`GET /market/indicators` response** (multi-line indicator, e.g. `macd`):

```json
{ "data": [ { "time": 1758067200, "values": { "macd": 120.4, "signal": 98.1, "histogram": 22.3 } } ] }
```

**`GET /time` response:**

```json
{ "timestamp": 1758153612 }
```

### Watchlist, Portfolio, Orders, Backtest

These four are standard authenticated CRUD/read shapes and don't need TradingView-specific handling — nothing here is consumed by the Datafeed adapter — but concrete shapes are still worth pinning down:

**`GET /watchlist` response:**

```json
{ "data": [ { "symbol": "VNM", "addedAt": "2026-08-01T03:12:00Z", "lastPrice": 68500, "change": 500, "changePercent": 0.74, "volume": 1245300 } ] }
```

**`GET /portfolio` response:**

```json
{ "cashBalance": 85000000, "marketValue": 15000000, "totalEquity": 100000000, "unrealizedPnl": 320000, "unrealizedPnlPercent": 2.18 }
```

**`POST /orders` request → response** (`market` or `limit`, `buy` or `sell`; `409` on insufficient virtual cash/shares):

```json
// request
{ "symbol": "VNM", "side": "buy", "type": "market", "quantity": 100 }

// 201 response
{
  "id": "b3f1c2e4-...", "symbol": "VNM", "side": "buy", "type": "market", "quantity": 100,
  "status": "filled", "filledPrice": 68500, "filledAt": "2026-09-18T02:30:00Z",
  "createdAt": "2026-09-18T02:30:00Z"
}
```

**`POST /backtests` request → response** (queues onto the worker pool; poll `GET /backtests/:id` for the result):

```json
// request
{ "symbol": "VNM", "from": "2024-01-01", "to": "2026-01-01", "startingCapital": 100000000,
  "rule": { "type": "ema_crossover", "params": { "fast": 20, "slow": 50 } } }

// 202 response
{ "id": "9a2e...", "symbol": "VNM", "status": "queued", "createdAt": "2026-09-18T02:31:00Z" }

// later: GET /backtests/9a2e... → 200
{ "id": "9a2e...", "symbol": "VNM", "status": "completed", "createdAt": "2026-09-18T02:31:00Z",
  "finalCapital": 134500000, "returnPercent": 34.5, "totalTrades": 42, "winRate": 0.57 }
```

---

## 5. What's deliberately out of scope here

- **Real-time data.** Live prices, live bars, and (later) order-flow ticks travel over the WebSocket `ws.Hub` connection, not REST — there's no way to model a persistent push channel as a Gin route, and trying to force one (long-polling, SSE-over-REST) would fight the architecture already chosen. That contract gets its own short doc when it's built, not a REST endpoint list.
- **Version 2+ endpoints** (screener, heatmap, strategy builder, replay mode, journal, alerts) — left out until Version 1 actually ships, per the phased build sequence in `version-highlights.md`.
- **A machine-readable contract (OpenAPI/Swagger).** This doc is the human-readable blueprint; generating and maintaining an OpenAPI file from it (or from `swaggo` annotations on the Gin handlers) is a separate, later decision, not assumed here.
