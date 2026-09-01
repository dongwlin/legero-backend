package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a gin.HandlerFunc that handles Cross-Origin Resource Sharing.
// Cache-validation headers are exposed only for v2 API requests. Keeping them
// out of the shared policy preserves the frozen v1 contract and keeps
// infrastructure endpoints such as /healthz outside the cache protocol.
func CORS() gin.HandlerFunc {
	base := cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
		},
		ExposeHeaders: []string{
			"Content-Type",
		},
		MaxAge:                    24 * time.Hour,
		OptionsResponseStatusCode: http.StatusNoContent,
	})
	v2 := cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Authorization",
			"Content-Type",
			"Cache-Control",
			"Accept",
			"If-None-Match",
		},
		ExposeHeaders: []string{
			"Content-Type",
			"Cache-Control",
			"ETag",
		},
		MaxAge:                    24 * time.Hour,
		OptionsResponseStatusCode: http.StatusNoContent,
	})

	return func(c *gin.Context) {
		if isV2Path(c.Request.URL.Path) {
			v2(c)
			return
		}
		base(c)
	}
}

func isV2Path(path string) bool {
	return path == "/api/v2" || strings.HasPrefix(path, "/api/v2/")
}
