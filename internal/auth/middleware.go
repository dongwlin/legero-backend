package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/identity"
)

func Middleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.AbortError(c, httpx.UnauthorizedError("missing authorization header"))
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			httpx.AbortError(c, httpx.UnauthorizedError("invalid authorization header"))
			return
		}

		authCtx, err := service.RequireAccessToken(c.Request.Context(), parts[1])
		if err != nil {
			httpx.AbortError(c, err)
			return
		}

		c.Set(identity.GinContextKey, authCtx)
		c.Next()
	}
}

func ContextFromGin(c *gin.Context) (*identity.Context, bool) {
	value, ok := c.Get(identity.GinContextKey)
	if !ok {
		return nil, false
	}
	authCtx, ok := value.(*identity.Context)
	return authCtx, ok
}
