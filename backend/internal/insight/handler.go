package insight

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts GET /symbols/:symbol/insight directly on the v1
// group (not a sub-group of symbol.RegisterRoutes' own group) so this
// package stays independent of the symbol package's routing -- it only
// depends on symbol's Service through the SymbolPort interface. Public,
// same reasoning as symbol/market: it backs a page section before login.
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	v1.GET("/symbols/:symbol/insight", generateHandler(svc))
}

func generateHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := svc.Generate(c.Param("symbol"))
		if err != nil {
			if errors.Is(err, ErrSymbolNotFound) {
				httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
				return
			}
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not generate insight")
			return
		}
		httpx.Item(c, http.StatusOK, result)
	}
}
