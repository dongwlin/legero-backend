package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dongwlin/legero-backend/internal/handler/middleware"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

// RegisterRoutes registers all v1 routes on the given router.
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
	realtimeHandler := NewRealtimeHandler(broker, sessions, location, cfg, now)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
	}

	r.GET("/ws", realtimeHandler.ServeWS)

	protected := r.Group("/")
	protected.Use(middleware.Auth(authSvc))
	{
		protected.GET("/bootstrap", authHandler.Bootstrap)
		protected.GET("/orders", orderHandler.List)
		protected.HEAD("/orders", orderHandler.List)
		protected.POST("/orders", orderHandler.Create)
		protected.PUT("/orders/:id", orderHandler.Update)
		protected.POST("/orders/:id/actions/toggle-step", orderHandler.ToggleStep)
		protected.POST("/orders/:id/actions/toggle-served", orderHandler.ToggleServed)
		protected.DELETE("/orders/:id", orderHandler.Delete)
		protected.POST("/orders/actions/clear", orderHandler.Clear)
		protected.POST("/realtime/session", realtimeHandler.CreateSession)
		protected.GET("/stats/daily", statsHandler.Daily)
		protected.HEAD("/stats/daily", statsHandler.Daily)
		protected.GET("/stats/report", statsHandler.Report)
		protected.HEAD("/stats/report", statsHandler.Report)
	}
}
