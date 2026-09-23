package market

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts the public market-data endpoints (unauthenticated,
// same reasoning as symbol.RegisterRoutes).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	group := v1.Group("/market")
	group.GET("/bars", barsHandler(svc))
	group.GET("/indicators", indicatorsHandler(svc))
	group.GET("/indices", indicesHandler(svc))
	group.GET("/orderbook", orderBookHandler(svc))
	group.GET("/macd", macdHandler(svc))
	v1.GET("/time", timeHandler())
}

// macdHandler mirrors indicatorsHandler's query-param shape (symbol/
// resolution/from/to) plus fast/slow/signal periods, defaulting to the
// conventional MACD(12,26,9) Detail.dc.html's chart uses.
func macdHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sym := c.Query("symbol")
		resolution := c.DefaultQuery("resolution", "1D")
		from, to, ok := parseRange(c)
		if sym == "" || !ok {
			httpx.ValidationError(c, "symbol, from, and to are required", nil)
			return
		}
		fast, _ := strconv.Atoi(c.DefaultQuery("fast", "12"))
		slow, _ := strconv.Atoi(c.DefaultQuery("slow", "26"))
		signal, _ := strconv.Atoi(c.DefaultQuery("signal", "9"))
		points := svc.GetMACD(sym, resolution, fast, slow, signal, from, to)
		if points == nil {
			points = []IndicatorMultiPoint{}
		}
		c.JSON(http.StatusOK, gin.H{"data": points})
	}
}

// indicesHandler returns all four tracked indices (Main.dc.html's index
// cards) in one call -- the frontend always needs all of them together,
// so there's no per-index query param.
func indicesHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		out := make([]IndexSnapshot, 0, len(Indices))
		for _, name := range Indices {
			out = append(out, svc.GetIndex(name))
		}
		c.JSON(http.StatusOK, gin.H{"data": out})
	}
}

func orderBookHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sym := c.Query("symbol")
		if sym == "" {
			httpx.ValidationError(c, "symbol is required", nil)
			return
		}
		today, _, ok := svc.LatestClose(sym)
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "no quote data for this symbol")
			return
		}
		bids, asks := svc.GetOrderBook(sym, today)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"bids": bids, "asks": asks}})
	}
}

func barsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sym := c.Query("symbol")
		resolution := c.DefaultQuery("resolution", "1D")
		from, to, ok := parseRange(c)
		if sym == "" || !ok {
			httpx.ValidationError(c, "symbol, from, and to are required", nil)
			return
		}
		bars := svc.GetBars(sym, resolution, from, to)
		if bars == nil {
			bars = []Bar{}
		}
		c.JSON(http.StatusOK, gin.H{"data": bars})
	}
}

func indicatorsHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		sym := c.Query("symbol")
		resolution := c.DefaultQuery("resolution", "1D")
		indicator := c.Query("indicator")
		period, _ := strconv.Atoi(c.DefaultQuery("period", "14"))
		from, to, ok := parseRange(c)
		if sym == "" || indicator == "" || !ok {
			httpx.ValidationError(c, "symbol, indicator, from, and to are required", nil)
			return
		}
		points := svc.GetIndicator(sym, resolution, indicator, period, from, to)
		if points == nil {
			points = []IndicatorPoint{}
		}
		c.JSON(http.StatusOK, gin.H{"data": points})
	}
}

func timeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"timestamp": time.Now().Unix()})
	}
}

func parseRange(c *gin.Context) (from, to int64, ok bool) {
	from, err1 := strconv.ParseInt(c.Query("from"), 10, 64)
	to, err2 := strconv.ParseInt(c.Query("to"), 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return from, to, true
}
