package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns a gin.HandlerFunc that handles Cross-Origin Resource Sharing.
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
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
}
