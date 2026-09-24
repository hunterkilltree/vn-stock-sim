package quant

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/quant", middleware.RequireAuth(tokens))
	group.POST("/test", testHandler(svc))
	group.POST("/chat", chatHandler(svc))
}

func testHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid quant test request", nil)
			return
		}
		resp, err := svc.Test(c.Request.Context(), req)
		if err != nil {
			writeQuantError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func chatHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid quant chat request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		resp, err := svc.Chat(c.Request.Context(), userID, req)
		if err != nil {
			writeQuantError(c, err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// writeQuantError maps service errors to responses. A provider's own
// error (bad key, unknown model, rate limit) is a 502 carrying the
// provider's status and message, so the UI can show "401: invalid x-api-key"
// instead of a generic failure.
func writeQuantError(c *gin.Context, err error) {
	var pe *ProviderError
	switch {
	case errors.As(err, &pe):
		c.JSON(http.StatusBadGateway, gin.H{"code": "provider_error", "message": pe.Message, "providerStatus": pe.Status})
	case errors.Is(err, ErrMissingKey), errors.Is(err, ErrMissingBaseURL):
		httpx.Error(c, http.StatusBadRequest, "missing_connection", err.Error())
	case errors.Is(err, ErrEndpointNotAllowed):
		httpx.Error(c, http.StatusBadRequest, "endpoint_not_allowed", err.Error())
	case errors.Is(err, ErrTimeout):
		httpx.Error(c, http.StatusGatewayTimeout, "model_timeout", err.Error())
	case errors.Is(err, ErrRefused):
		httpx.Error(c, http.StatusUnprocessableEntity, "model_refused", err.Error())
	case errors.Is(err, ErrInvalidOutput):
		httpx.Error(c, http.StatusBadGateway, "invalid_model_output", err.Error())
	default:
		httpx.Error(c, http.StatusBadGateway, "model_unreachable", err.Error())
	}
}
