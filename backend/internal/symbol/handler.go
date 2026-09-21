package symbol

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts the public (unauthenticated) symbol endpoints.
// Public by design: they back the stock browser and chart before login
// (api-spec.md section 1).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	group := v1.Group("/symbols")
	group.GET("", searchHandler(svc))
	group.GET("/:symbol", detailHandler(svc))
}

func searchHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, pageSize := httpx.Pagination(c)
		items, total := svc.Search(c.Query("q"), c.Query("exchange"), page, pageSize)
		httpx.List(c, items, httpx.BuildMeta(page, pageSize, total))
	}
}

func detailHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		detail, ok := svc.Detail(c.Param("symbol"))
		if !ok {
			httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
			return
		}
		httpx.Item(c, http.StatusOK, detail)
	}
}
