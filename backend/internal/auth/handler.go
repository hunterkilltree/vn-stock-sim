package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

// RegisterRoutes wires this feature's routes onto the v1 group, following
// the same RegisterRoutes(group, service) shape used by every feature
// package (see api-spec.md section 2).
func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/auth")
	group.POST("/register", registerHandler(svc))
	group.POST("/login", loginHandler(svc))
	group.GET("/me", middleware.RequireAuth(tokens), meHandler(svc))
}

func registerHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid request body", nil)
			return
		}
		resp, err := svc.Register(req.Email, req.DisplayName, req.Password, req.MarketInterest)
		if err != nil {
			if err == ErrEmailTaken {
				httpx.Error(c, http.StatusConflict, "email_taken", "an account with this email already exists")
				return
			}
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not register user")
			return
		}
		httpx.Item(c, http.StatusCreated, resp)
	}
}

func loginHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid request body", nil)
			return
		}
		resp, err := svc.Login(req.Email, req.Password)
		if err != nil {
			httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
			return
		}
		httpx.Item(c, http.StatusOK, resp)
	}
}

func meHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		user, err := svc.Me(userID)
		if err != nil {
			httpx.Error(c, http.StatusNotFound, "not_found", "user not found")
			return
		}
		httpx.Item(c, http.StatusOK, user)
	}
}
