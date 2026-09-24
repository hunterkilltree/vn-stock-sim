package screener

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts the public (unauthenticated) heatmap/movers
// endpoints under the existing /market group -- same reasoning as
// symbol/market's own public routes (they back the Main screen before
// login).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	group := v1.Group("/market")
	group.GET("/heatmap", heatmapHandler(svc))
	group.GET("/movers", moversHandler(svc))
}

func heatmapHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "1D")
		if _, ok := PeriodSessions[period]; !ok {
			httpx.ValidationError(c, "period must be one of 1D, 1W, 1M, 3M", nil)
			return
		}
		exchange := c.DefaultQuery("exchange", "")
		switch exchange {
		case "", "ALL":
			exchange = ""
		case "HOSE", "HNX", "UPCOM":
		default:
			httpx.ValidationError(c, "exchange must be HOSE, HNX, UPCOM or ALL", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.GetSectorHeatmap(period, exchange)})
	}
}

func moversHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		direction := c.DefaultQuery("direction", "up")
		if direction != "up" && direction != "down" {
			httpx.ValidationError(c, "direction must be 'up' or 'down'", nil)
			return
		}
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "5"))
		if err != nil || limit <= 0 {
			limit = 5
		}
		c.JSON(http.StatusOK, gin.H{"data": svc.GetTopMovers(direction, limit)})
	}
}
