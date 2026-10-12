package stress

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

// RegisterRoutes mounts GET /portfolios/:id/stress-test, authenticated
// like the other per-portfolio reads.
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	v1.GET("/portfolios/:id/stress-test", middleware.RequireAuth(tokens), func(c *gin.Context) {
		res, err := svc.Run(c.GetString(middleware.ContextUserIDKey), c.Param("id"))
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.Error(c, http.StatusNotFound, "not_found", "portfolio not found")
		case errors.Is(err, ErrNotStocks):
			httpx.Error(c, http.StatusUnprocessableEntity, "unsupported_market", "stress test covers stock portfolios only")
		case err != nil:
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not run stress test")
		default:
			httpx.Item(c, http.StatusOK, res)
		}
	})
}
