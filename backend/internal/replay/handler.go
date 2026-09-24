package replay

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/replay/sessions", middleware.RequireAuth(tokens))
	group.POST("", startHandler(svc))
	group.GET("/:id", getHandler(svc))
	group.POST("/:id/advance", advanceHandler(svc))
	group.POST("/:id/orders", orderHandler(svc))
	group.POST("/:id/end", endHandler(svc))
}

func startHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req StartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid replay session request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		view, err := svc.Start(userID, req)
		if err != nil {
			if errors.Is(err, ErrNoData) {
				httpx.Error(c, http.StatusNotFound, "not_found", "no historical data for that symbol/range")
				return
			}
			httpx.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		httpx.Item(c, http.StatusCreated, view)
	}
}

func getHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		view, err := svc.Get(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusNotFound, "not_found", "replay session not found")
			return
		}
		httpx.Item(c, http.StatusOK, view)
	}
}

func advanceHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		view, err := svc.Advance(userID, c.Param("id"))
		if err != nil {
			writeSessionErr(c, err)
			return
		}
		httpx.Item(c, http.StatusOK, view)
	}
}

func orderHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req OrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid replay order request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		view, err := svc.PlaceOrder(userID, c.Param("id"), req)
		if err != nil {
			writeSessionErr(c, err)
			return
		}
		httpx.Item(c, http.StatusCreated, view)
	}
}

func endHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		view, err := svc.End(userID, c.Param("id"))
		if err != nil {
			writeSessionErr(c, err)
			return
		}
		httpx.Item(c, http.StatusOK, view)
	}
}

func writeSessionErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "replay session not found")
	case errors.Is(err, ErrInvalidQuantity):
		httpx.ValidationError(c, err.Error(), nil)
	case errors.Is(err, ErrSessionDone):
		httpx.Error(c, http.StatusConflict, "session_completed", "replay session already completed")
	default:
		httpx.Error(c, http.StatusConflict, "cannot_process", err.Error())
	}
}
