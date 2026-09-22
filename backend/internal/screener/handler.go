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
		c.JSON(http.StatusOK, gin.H{"data": svc.GetSectorHeatmap()})
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
