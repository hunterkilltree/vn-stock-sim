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
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/order"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/watchlist"
)

func main() {
	cfg := config.Load()
	tokens := authtoken.NewIssuer(cfg.JWTSecret, 24*time.Hour)

	symbolProvider := symbol.NewMockProvider()
	symbolSvc := symbol.NewService(symbolProvider)

	marketProvider := market.NewMockProvider()
	marketSvc := market.NewService(marketProvider)

	authSvc := auth.NewService(auth.NewMemoryStore(), tokens)
	watchlistSvc := watchlist.NewService(watchlist.NewMemoryStore(), symbolSvc)
	portfolioStore := portfolio.NewMemoryStore()
	portfolioSvc := portfolio.NewService(portfolioStore, symbolSvc)
	orderSvc := order.NewService(order.NewMemoryStore(), portfolioStore, symbolSvc)
	backtestSvc := backtest.NewService(backtest.NewMemoryStore(), marketSvc)

	router := gin.Default()
	v1 := router.Group("/api/v1")

	auth.RegisterRoutes(v1, authSvc, tokens)
	symbol.RegisterRoutes(v1, symbolSvc)
	market.RegisterRoutes(v1, marketSvc)
	watchlist.RegisterRoutes(v1, watchlistSvc, tokens)
	portfolio.RegisterRoutes(v1, portfolioSvc, tokens)
	order.RegisterRoutes(v1, orderSvc, tokens)
	backtest.RegisterRoutes(v1, backtestSvc, tokens)

	log.Printf("VN Stock Sim API listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
