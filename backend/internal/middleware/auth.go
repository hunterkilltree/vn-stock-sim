// Package middleware holds Gin middleware shared across feature route
// groups — currently just the bearer-token auth check described in
// api-spec.md section 3 ("checked by a Gin middleware registered once on
// the v1 group ... except the handful of public routes").
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
)

const ContextUserIDKey = "userID"
const ContextEmailKey = "userEmail"

// RequireAuth validates the Authorization: Bearer <JWT> header and stores
// the resolved user id/email in the Gin context for downstream handlers.
func RequireAuth(issuer *authtoken.Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			httpx.Error(c, 401, "unauthorized", "missing or malformed Authorization header")
			return
		}
		claims, err := issuer.Verify(strings.TrimPrefix(header, prefix))
		if err != nil {
			httpx.Error(c, 401, "unauthorized", "invalid or expired token")
			return
		}
		c.Set(ContextUserIDKey, claims.Subject)
		c.Set(ContextEmailKey, claims.Email)
		c.Next()
	}
}
