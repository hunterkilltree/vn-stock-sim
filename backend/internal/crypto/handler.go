package crypto

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts the public crypto market endpoints under
// /api/v1/crypto (public like the stock symbol/market reads).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	g := v1.Group("/crypto")
	g.GET("/pairs", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": svc.Pairs()}) })
	g.GET("/pairs/:symbol", pairHandler(svc))
	g.GET("/bars", barsHandler(svc))
	g.GET("/orderbook", orderBookHandler(svc))
	g.GET("/heatmap", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": svc.Heatmap()}) })
	g.GET("/movers", moversHandler(svc))
	g.GET("/overview", func(c *gin.Context) { c.JSON(http.StatusOK, svc.Overview()) })
}

func pairHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		d, ok := svc.Detail(c.Param("symbol"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "pair not found")
			return
		}
		httpx.Item(c, http.StatusOK, d)
	}
}

func barsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now().Unix()
		from, err1 := strconv.ParseInt(c.DefaultQuery("from", strconv.FormatInt(now-7*86400, 10)), 10, 64)
		to, err2 := strconv.ParseInt(c.DefaultQuery("to", strconv.FormatInt(now, 10)), 10, 64)
		if err1 != nil || err2 != nil || from >= to {
			httpx.ValidationError(c, "from/to must be unix seconds with from < to", nil)
			return
		}
		bars, source, ok := svc.Bars(c.Query("pair"), c.DefaultQuery("interval", "1h"), from, to)
		if !ok {
			httpx.ValidationError(c, "unknown pair or interval (5m, 15m, 1h, 4h, 1d, 1w)", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": bars, "source": source})
	}
}

func orderBookHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		levels, err := strconv.Atoi(c.DefaultQuery("levels", "4"))
		if err != nil || levels < 1 || levels > 20 {
			levels = 4
		}
		ob, ok := svc.OrderBook(c.Query("pair"), levels)
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "pair not found")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": ob})
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
		c.JSON(http.StatusOK, gin.H{"data": svc.Movers(direction, limit)})
	}
}
