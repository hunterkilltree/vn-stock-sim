package watchlist

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/watchlist", middleware.RequireAuth(tokens))
	group.GET("", listHandler(svc))
	group.POST("", addHandler(svc))
	group.DELETE("/:symbol", removeHandler(svc))
}

func listHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, gin.H{"data": svc.List(userID)})
	}
}

func addHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req addRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "symbol is required", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		if err := svc.Add(userID, req.Symbol); errors.Is(err, ErrUnknownSymbol) {
			httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.List(userID)})
	}
}

func removeHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		svc.Remove(userID, c.Param("symbol"))
		c.Status(http.StatusNoContent)
	}
}
