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
	return AuthWithError(svc, httpresp.AbortError)
}

// AuthWithError returns an authentication middleware that uses abortError to
// render authentication failures. This keeps the shared token validation
// logic reusable while allowing versioned handlers to retain their response
// contracts.
func AuthWithError(svc service.Auth, abortError func(*gin.Context, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortError(c, apperr.UnauthorizedError("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abortError(c, apperr.UnauthorizedError("invalid authorization header"))
			return
		}

		authCtx, err := svc.RequireAccessToken(c.Request.Context(), parts[1])
		if err != nil {
			abortError(c, err)
			return
		}

		c.Set(identity.GinContextKey, authCtx)
		c.Next()
	}
}
