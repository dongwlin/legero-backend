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
			PerformanceRequestIDHeader,
		},
		ExposeHeaders: []string{
			"Content-Type",
			"Cache-Control",
			PerformanceRequestIDHeader,
			"Server-Timing",
		},
		MaxAge:                    24 * time.Hour,
		OptionsResponseStatusCode: http.StatusNoContent,
	})
}
