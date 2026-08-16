package middleware

import (
	"strings"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
	"github.com/dongwlin/legero-backend/internal/service"
	"github.com/gin-gonic/gin"
)

// Auth returns a gin.HandlerFunc that validates the Bearer access token
// and stores the identity context in the gin context.
func Auth(svc service.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpresp.AbortError(c, apperr.UnauthorizedError("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			httpresp.AbortError(c, apperr.UnauthorizedError("invalid authorization header"))
			return
		}

		authCtx, err := svc.RequireAccessToken(c.Request.Context(), parts[1])
		if err != nil {
			httpresp.AbortError(c, err)
			return
		}

		c.Set(identity.GinContextKey, authCtx)
		c.Next()
	}
}
