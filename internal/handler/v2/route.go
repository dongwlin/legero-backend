package v2

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/middleware"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

// RegisterRoutes registers all v2 routes on the given router.
//
// Every v2 response is wrapped in the unified httpresp.Response envelope. The
// WebSocket upgrade stays at GET /api/ws: it speaks the realtime message
// protocol rather than the v2 JSON envelope, so only its ticket endpoint is
// versioned here.
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
	authHandler := NewAuthHandler(authSvc, location)
	orderHandler := NewOrderHandler(orderSvc, location)
	statsHandler := NewStatsHandler(statsSvc, location)
	realtimeHandler := NewRealtimeHandler(sessions, location)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	protected := r.Group("/")
	protected.Use(middleware.Auth(authSvc))
	{
		protected.GET("/bootstrap", authHandler.Bootstrap)
		protected.GET("/orders", orderHandler.List)
		protected.POST("/orders", orderHandler.Create)
		protected.PUT("/orders/:id", orderHandler.Update)
		protected.POST("/orders/:id/actions/toggle-step", orderHandler.ToggleStep)
		protected.POST("/orders/:id/actions/toggle-served", orderHandler.ToggleServed)
		protected.DELETE("/orders/:id", orderHandler.Delete)
		protected.POST("/orders/actions/clear", orderHandler.Clear)
		protected.POST("/realtime/session", realtimeHandler.CreateSession)
		protected.GET("/stats/daily", statsHandler.Daily)
		protected.GET("/stats/report", statsHandler.Report)
	}
}
