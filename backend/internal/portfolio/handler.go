package portfolio

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

type createPortfolioRequest struct {
	Name            string  `json:"name"`
	Market          string  `json:"market"`
	StartingCapital float64 `json:"startingCapital"`
	Currency        string  `json:"currency"`
}

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	// Singular /portfolio -- kept for the pre-Phase-B frontend, always
	// resolves to the caller's lazily-created default portfolio.
	group := v1.Group("/portfolio", middleware.RequireAuth(tokens))
	group.GET("", summaryHandler(svc))
	group.GET("/positions", positionsHandler(svc))

	// Plural /portfolios -- the real multi-portfolio API (phase-b.md
	// decision 1), used by the Account-Menu switcher once Phase G wires
	// the frontend to it.
	portfolios := v1.Group("/portfolios", middleware.RequireAuth(tokens))
	portfolios.GET("", listPortfoliosHandler(svc))
	portfolios.POST("", createPortfolioHandler(svc))
	portfolios.GET("/:id", getPortfolioHandler(svc))
	portfolios.GET("/:id/summary", portfolioSummaryHandler(svc))
	portfolios.GET("/:id/positions", portfolioPositionsHandler(svc))
	portfolios.GET("/:id/equity-history", portfolioEquityHistoryHandler(svc))
	portfolios.GET("/:id/allocation", portfolioAllocationHandler(svc))
	portfolios.GET("/:id/stats", portfolioStatsHandler(svc))
}

func portfolioSummaryHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		c.JSON(http.StatusOK, svc.Summary(p.ID))
	}
}

func portfolioPositionsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.Positions(p.ID)})
	}
}

func portfolioEquityHistoryHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.EquityHistory(p.ID)})
	}
}

func portfolioAllocationHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.Allocation(p.ID)})
	}
}

func portfolioStatsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		c.JSON(http.StatusOK, svc.Stats(p.ID, p.StartingCapital))
	}
}

func summaryHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, svc.Summary(svc.DefaultPortfolioID(userID)))
	}
}

func positionsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, gin.H{"data": svc.Positions(svc.DefaultPortfolioID(userID))})
	}
}

func listPortfoliosHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, gin.H{"data": svc.ListPortfolios(userID)})
	}
}

func createPortfolioHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createPortfolioRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid portfolio request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		p := svc.CreatePortfolio(userID, req.Name, req.Market, req.StartingCapital, req.Currency)
		httpx.Item(c, http.StatusCreated, p)
	}
}

func getPortfolioHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		p, ok := svc.GetPortfolio(userID, c.Param("id"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
			return
		}
		httpx.Item(c, http.StatusOK, p)
	}
}
