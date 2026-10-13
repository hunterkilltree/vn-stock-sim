package luk

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

// RegisterRoutes mounts the checklist and the Gemini review, both public
// like rating: anyone viewing a stock can ask for the review. The service's
// 15-minute per-symbol cache bounds how often it spends the operator's
// Gemini quota (phase-luk-review.md decision 6).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service) {
	v1.GET("/symbols/:symbol/luk-review", func(c *gin.Context) {
		res, err := svc.Review(c.Param("symbol"))
		if err != nil {
			writeErr(c, err)
			return
		}
		httpx.Item(c, http.StatusOK, res)
	})
	v1.POST("/symbols/:symbol/luk-review/ai", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
		defer cancel()
		res, err := svc.AIReview(ctx, c.Param("symbol"))
		if err != nil {
			writeErr(c, err)
			return
		}
		httpx.Item(c, http.StatusOK, res)
	})
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSymbolNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
	case errors.Is(err, ErrAIDisabled):
		httpx.Error(c, http.StatusServiceUnavailable, "ai_disabled", "GEMINI_API_KEY is not set on the server")
	case errors.Is(err, ErrAIFailed):
		httpx.Error(c, http.StatusBadGateway, "ai_failed", "the AI review failed; try again later")
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not review symbol")
	}
}
