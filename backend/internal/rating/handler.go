package rating

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts GET /symbols/:symbol/rating on the v1 group.
// Public, like insight: it backs a section of the detail page before login.
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	v1.GET("/symbols/:symbol/rating", func(c *gin.Context) {
		res, err := svc.Rate(c.Param("symbol"))
		if err != nil {
			if errors.Is(err, ErrSymbolNotFound) {
				httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
				return
			}
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not rate symbol")
			return
		}
		httpx.Item(c, http.StatusOK, res)
	})
}
