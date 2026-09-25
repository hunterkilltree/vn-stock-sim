// Command api is the VN Stock Sim V1 MVP backend entry point. It wires
// every feature's Service (backed by in-memory stores/mock adapters — see
// RESUME.md for what's deferred) into Gin route groups under /api/v1, per
// the layout documented in api-spec.md section 2.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/config"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/auth"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/backtest"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/crypto"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
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

	st := openStores(cfg.DatabaseURL)

	authSvc := auth.NewService(st.auth, tokens)
	// quotes (not symbolSvc) so a watchlist can hold crypto pairs too
	// (phase-k.md decision 11).
	watchlistSvc := watchlist.NewService(st.watchlist, quotes)
	portfolioSvc := portfolio.NewService(st.portfolio, quotes)
	// orderStore is a named variable (not inlined) because Phase F's
	// replaySvc below also needs it, to log Replay fills as real
	// order.Order records -- see phase-f.md decision 5.
	orderStore := st.order
	// Ledger is portfolioSvc, not the bare store -- portfolioSvc.ApplyFill
	// wraps the store's ledger op with a real equity-history snapshot on
	// every fill (see phase-e.md item 1 and portfolio/service.go).
	orderSvc := order.NewService(orderStore, portfolioSvc, quotes, portfolioSvc)
	// SetOrdersPort closes the reverse dependency (portfolioSvc.Stats
	// needs order history) after both services exist, avoiding an import
	// cycle -- see portfolio/types.go's OrdersPort comment.
	portfolioSvc.SetOrdersPort(orderSvc)
	// Queued limit/stop/ATC/OCO orders are matched against 5-minute bars
	// in the background (phase-k.md decisions 4-10).
	orderSvc.EnableMatching(crypto.BarsRouter{Crypto: cryptoSvc, Stock: marketSvc})
	go orderSvc.RunMatcher(context.Background(), cfg.OrderMatchInterval)
	backtestSvc := backtest.NewService(st.backtest, marketSvc)
	insightSvc := insight.NewService(symbolSvc, marketSvc)
	screenerSvc := screener.NewService(symbolSvc, marketSvc)
	// replaySvc reuses marketSvc (historical bars are just GetBars with a
	// date range in the past), portfolioSvc (each session gets its own
	// dedicated portfolio, see phase-f.md decision 4), and orderStore
	// (Replay fills are logged as real orders, decision 5) -- no new
	// dependency surface on any existing package.
	replaySvc := replay.NewService(st.replay, marketSvc, cryptoSvc, portfolioSvc, orderStore)
	quantSvc := quant.NewService(quant.DefaultClients(cfg.QuantAllowPrivateEndpoints), symbolSvc, marketSvc, watchlistSvc, portfolioSvc)

	router := gin.Default()
	// For a host's health check: the process is up, and whether Postgres
	// answers ("off" when running in memory) -- phase-persistence.md
	// decision 9.
	router.GET("/healthz", func(c *gin.Context) {
		status := "off"
		if st.pool != nil {
			status = "ok"
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			defer cancel()
			if err := st.pool.Ping(ctx); err != nil {
				status = "down"
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": status})
	})
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

// stores holds every user-data store: Postgres when DATABASE_URL is set,
// in-memory otherwise (phase-persistence.md decisions 2 and 3).
type stores struct {
	pool      *pgxpool.Pool
	auth      auth.Store
	watchlist watchlist.Store
	portfolio portfolio.Store
	order     order.Store
	backtest  backtest.Store
	replay    replay.Store
}

func openStores(databaseURL string) stores {
	if databaseURL == "" {
		log.Printf("storage: in-memory (DATABASE_URL unset) -- data resets on restart")
		return stores{
			auth:      auth.NewMemoryStore(),
			watchlist: watchlist.NewMemoryStore(),
			portfolio: portfolio.NewMemoryStore(),
			order:     order.NewMemoryStore(),
			backtest:  backtest.NewMemoryStore(),
			replay:    replay.NewMemoryStore(),
		}
	}
	pool, err := db.Open(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage: postgres (schema migrated)")
	return stores{
		pool:      pool,
		auth:      auth.NewPGStore(pool),
		watchlist: watchlist.NewPGStore(pool),
		portfolio: portfolio.NewPGStore(pool),
		order:     order.NewPGStore(pool),
		backtest:  backtest.NewPGStore(pool),
		replay:    replay.NewPGStore(pool),
	}
}
