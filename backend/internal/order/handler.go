package order

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/httpx"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, svc *Service, tokens *authtoken.Issuer) {
	group := v1.Group("/orders", middleware.RequireAuth(tokens))
	group.GET("", listHandler(svc))
	group.POST("", createHandler(svc))
	group.POST("/check", checkHandler(svc))
	v1.GET("/trade-plans", middleware.RequireAuth(tokens), plansHandler(svc))
	group.GET("/:id", getHandler(svc))
	group.POST("/:id/cancel", cancelHandler(svc))
}

func listHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		c.JSON(http.StatusOK, gin.H{"data": svc.List(userID)})
	}
}

func createHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid order request", nil)
			return
		}
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Create(userID, req)
		if err != nil {
			writeOrderError(c, err)
			return
		}
		httpx.Item(c, http.StatusCreated, o)
	}
}

// checkHandler is the ticket's dry run (phase-risk-rules.md design item
// 1): the findings Create would apply, plus the risk-sized quantity.
func checkHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.ValidationError(c, "invalid order request", nil)
			return
		}
		a, err := svc.Check(c.GetString(middleware.ContextUserIDKey), req)
		if err != nil {
			writeOrderError(c, err)
			return
		}
		httpx.Item(c, http.StatusOK, a)
	}
}

func plansHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": svc.Plans(c.GetString(middleware.ContextUserIDKey))})
	}
}

func writeOrderError(c *gin.Context, err error) {
	var rv *RuleViolation
	if errors.As(err, &rv) {
		// api-spec.md's validation shape, with 422: the request is well
		// formed, the portfolio's rules refuse it.
		fields := make([]httpx.FieldError, len(rv.Findings))
		for i, f := range rv.Findings {
			fields[i] = httpx.FieldError{Field: f.Code, Reason: f.Message}
		}
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"code": "rule_violation", "message": err.Error(), "fields": fields})
		return
	}
	switch {
	case errors.Is(err, ErrSymbolNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "symbol not found")
	case errors.Is(err, ErrPortfolioNotFound):
		httpx.Error(c, http.StatusNotFound, "portfolio_not_found", "portfolio not found")
	case errors.Is(err, ErrWrongMarket):
		httpx.Error(c, http.StatusConflict, "wrong_market", err.Error())
	case errors.Is(err, ErrInvalidQuantity), errors.Is(err, ErrOCOPrices), errors.Is(err, ErrPriceRequired), errors.Is(err, ErrOddLot):
		httpx.ValidationError(c, err.Error(), nil)
	case errors.Is(err, ErrReplayPortfolio):
		httpx.Error(c, http.StatusConflict, "replay_portfolio", err.Error())
	case errors.Is(err, ErrInsufficientFunds):
		httpx.Error(c, http.StatusConflict, "insufficient_funds", "not enough virtual cash for this order")
	case errors.Is(err, ErrInsufficientShares):
		httpx.Error(c, http.StatusConflict, "insufficient_shares", "not enough shares held for this order")
	default:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "could not place order")
	}
}

func getHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Get(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusNotFound, "not_found", "order not found")
			return
		}
		httpx.Item(c, http.StatusOK, o)
	}
}

func cancelHandler(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.ContextUserIDKey)
		o, err := svc.Cancel(userID, c.Param("id"))
		if err != nil {
			httpx.Error(c, http.StatusConflict, "cannot_cancel", err.Error())
			return
		}
		httpx.Item(c, http.StatusOK, o)
	}
}
