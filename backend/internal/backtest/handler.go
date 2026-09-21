package backtest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/backtests", middleware.RequireAuth(tokens))
	group.GET("", listHandler(svc))
	group.POST("", createHandler(svc))
	group.GET("/:id", getHandler(svc))
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
			httpx.ValidationError(c, "invalid backtest request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		bt, err := svc.Create(userID, req)
		if err != nil {
			if errors.Is(err, ErrUnsupportedRule) {
				httpx.ValidationError(c, "unsupported rule type", []httpx.FieldError{{Field: "rule.type", Reason: "only ema_crossover is supported in V1"}})
				return
			}
			httpx.Error(c, http.StatusBadRequest, "invalid_range", "invalid from/to date range")
			return
		}
		httpx.Item(c, http.StatusAccepted, bt)
	}
}

func getHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		bt, err := svc.Get(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusNotFound, "not_found", "backtest not found")
			return
		}
		httpx.Item(c, http.StatusOK, bt)
	}
}
