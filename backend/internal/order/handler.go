package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/orders", middleware.RequireAuth(tokens))
	group.GET("", listHandler(svc))
	group.POST("", createHandler(svc))
	group.GET("/:id", getHandler(svc))
	group.POST("/:id/cancel", cancelHandler(svc))
}

func listHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, gin.H{"data": svc.List(userID)})
	}
}

func createHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid order request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Create(userID, req)
		if err != nil {
			switch {
			case errors.Is(err, ErrSymbolNotFound):
				httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
			case errors.Is(err, ErrPortfolioNotFound):
				httpx.Error(c, http.StatusNotFound, "portfolio_not_found", "portfolio not found")
			case errors.Is(err, ErrWrongMarket):
				httpx.Error(c, http.StatusConflict, "wrong_market", err.Error())
			case errors.Is(err, ErrInvalidQuantity), errors.Is(err, ErrOCOPrices):
				httpx.ValidationError(c, err.Error(), nil)
			case errors.Is(err, ErrReplayPortfolio):
				httpx.Error(c, http.StatusConflict, "replay_portfolio", err.Error())
			case errors.Is(err, ErrInsufficientFunds):
				httpx.Error(c, http.StatusConflict, "insufficient_funds", "not enough virtual cash for this order")
			case errors.Is(err, ErrInsufficientShares):
				httpx.Error(c, http.StatusConflict, "insufficient_shares", "not enough shares held for this order")
			default:
				httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not place order")
			}
			return
		}
		httpx.Item(c, http.StatusCreated, o)
	}
}

func getHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Get(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusNotFound, "not_found", "order not found")
			return
		}
		httpx.Item(c, http.StatusOK, o)
	}
}

func cancelHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Cancel(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusConflict, "cannot_cancel", err.Error())
			return
		}
		httpx.Item(c, http.StatusOK, o)
	}
}
