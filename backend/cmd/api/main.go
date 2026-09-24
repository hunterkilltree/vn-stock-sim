// Command api is the VN Stock Sim V1 MVP backend entry point. It wires
// every feature's Service (backed by in-memory stores/mock adapters — see
// RESUME.md for what's deferred) into Gin route groups under /api/v1, per
// the layout documented in api-spec.md section 2.
package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/config"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/auth"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/backtest"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/crypto"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/insight"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/order"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/quant"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/replay"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/screener"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/watchlist"
)

func main() {
	cfg := config.Load()
	tokens := authtoken.NewIssuer(cfg.JWTSecret, 24*time.Hour)

	// MarketDataProvider adapter selection -- see phase-vci-market-data.md.
	// "vci" (default) is a real, live adapter against Vietcap Securities'
	// own trading-platform API, wrapped in LiveProvider so any failure
	// (this is an unofficial, undocumented upstream) falls back to the
	// deterministic mock generator per-call instead of breaking a page.
	mockProvider := market.NewMockProvider()
	var marketProvider market.MarketDataProvider = mockProvider
	if cfg.MarketDataSource == "vci" {
		marketProvider = market.NewLiveProvider(market.NewVCIProvider(), mockProvider)
		log.Printf("market data source: vci (live, with mock fallback)")
	} else {
		log.Printf("market data source: mock (MARKET_DATA_SOURCE=%q)", cfg.MarketDataSource)
	}
	marketSvc := market.NewService(marketProvider)

	// symbol.Service depends on marketSvc (via QuotePort) so a symbol's
	// lastPrice/change/changePercent are derived from the same bars the
	// chart renders, instead of an independently-seeded value that could
	// drift arbitrarily far from what the chart actually shows.
	symbolProvider := symbol.NewMockProvider()
	symbolSvc := symbol.NewService(symbolProvider, marketSvc)

	// Crypto market (Phase I): Binance's public market data with the same
	// per-call mock fallback as the stock side, or mock only when
	// MARKET_DATA_SOURCE=mock. QuoteRouter lets order/portfolio price both
	// markets through their existing Detail port (phase-i.md decision 7).
	var cryptoLive crypto.DataProvider
	if cfg.MarketDataSource == "vci" {
		cryptoLive = crypto.NewBinanceProvider()
	}
	cryptoSvc := crypto.NewService(crypto.NewLiveProvider(cryptoLive))
	quotes := crypto.QuoteRouter{Crypto: cryptoSvc, Stock: symbolSvc}

	authSvc := auth.NewService(auth.NewMemoryStore(), tokens)
	watchlistSvc := watchlist.NewService(watchlist.NewMemoryStore(), symbolSvc)
	portfolioStore := portfolio.NewMemoryStore()
	portfolioSvc := portfolio.NewService(portfolioStore, quotes)
	// orderStore is a named variable (not inlined) because Phase F's
	// replaySvc below also needs it, to log Replay fills as real
	// order.Order records -- see phase-f.md decision 5.
	orderStore := order.NewMemoryStore()
	// Ledger is portfolioSvc, not the bare store -- portfolioSvc.ApplyFill
	// wraps the store's ledger op with a real equity-history snapshot on
	// every fill (see phase-e.md item 1 and portfolio/service.go).
	orderSvc := order.NewService(orderStore, portfolioSvc, quotes, portfolioSvc)
	// SetOrdersPort closes the reverse dependency (portfolioSvc.Stats
	// needs order history) after both services exist, avoiding an import
	// cycle -- see portfolio/types.go's OrdersPort comment.
	portfolioSvc.SetOrdersPort(orderSvc)
	backtestSvc := backtest.NewService(backtest.NewMemoryStore(), marketSvc)
	insightSvc := insight.NewService(symbolSvc, marketSvc)
	screenerSvc := screener.NewService(symbolSvc, marketSvc)
	// replaySvc reuses marketSvc (historical bars are just GetBars with a
	// date range in the past), portfolioSvc (each session gets its own
	// dedicated portfolio, see phase-f.md decision 4), and orderStore
	// (Replay fills are logged as real orders, decision 5) -- no new
	// dependency surface on any existing package.
	replaySvc := replay.NewService(replay.NewMemoryStore(), marketSvc, cryptoSvc, portfolioSvc, orderStore)
	quantSvc := quant.NewService(quant.DefaultClients(cfg.QuantAllowPrivateEndpoints), symbolSvc, marketSvc, watchlistSvc, portfolioSvc)

	router := gin.Default()
	v1 := router.Group("/api/v1")

	auth.RegisterRoutes(v1, authSvc, tokens)
	symbol.RegisterRoutes(v1, symbolSvc)
	market.RegisterRoutes(v1, marketSvc)
	watchlist.RegisterRoutes(v1, watchlistSvc, tokens)
	portfolio.RegisterRoutes(v1, portfolioSvc, tokens)
	order.RegisterRoutes(v1, orderSvc, tokens)
	backtest.RegisterRoutes(v1, backtestSvc, tokens)
	insight.RegisterRoutes(v1, insightSvc)
	screener.RegisterRoutes(v1, screenerSvc)
	replay.RegisterRoutes(v1, replaySvc, tokens)
	quant.RegisterRoutes(v1, quantSvc, tokens)
	crypto.RegisterRoutes(v1, cryptoSvc)

	log.Printf("VN Stock Sim API listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
