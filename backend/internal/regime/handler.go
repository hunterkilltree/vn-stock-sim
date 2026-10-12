package regime

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts GET /market/regime. Public, like the other
// market read endpoints.
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	v1.GET("/market/regime", func(c *gin.Context) {
		r, ok := svc.Current()
		if !ok {
			httpx.Error(c, http.StatusServiceUnavailable, "insufficient_data", "not enough VN-Index history to judge the market regime")
			return
		}
		httpx.Item(c, http.StatusOK, r)
	})
}
