package v2

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/middleware"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

// RegisterRoutes registers all v2 routes on the given router. The v2 group
// applies the HTTP cache middleware which owns the response writer lifecycle:
// it wraps the writer to capture the response body, then after the handler
// completes generates an ETag, checks If-None-Match, and sets cache headers.
func RegisterRoutes(
	r gin.IRouter,
	authSvc service.Auth,
	orderSvc service.Order,
	statsSvc service.Stats,
	broker *realtime.Broker,
	sessions *realtime.SessionManager,
	location *time.Location,
	cfg *config.Config,
	now func() time.Time,
) {
	// Placeholder for v2 auth middleware — for now, reuse v1's auth
	// middleware. v2-specific auth (e.g. different token format) can be
	// introduced later without affecting v1.
	_ = authSvc

	protected := r.Group("/")
	protected.Use(middleware.Auth(authSvc))
	protected.Use(middleware.HTTPCache())

	// v2 routes will be registered here as handlers are migrated.
	// Example (not yet implemented):
	//   protected.GET("/orders/:id", orderHandler.Get)
}
