package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/auth"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/stats"
)

func newRouter(
	appLogger zerolog.Logger,
	authService *auth.Service,
	authHandler *auth.Handler,
	orderHandler *order.Handler,
	statsHandler *stats.Handler,
	realtimeHandler *realtime.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		httpx.CORSMiddleware(),
		logger.Gin(appLogger),
		gin.Recovery(),
	)
	router.GET("/healthz", func(c *gin.Context) {
		httpx.JSON(c, http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.GET("/ws", realtimeHandler.ServeWS)

	protected := api.Group("/")
	protected.Use(auth.Middleware(authService))
	protected.GET("/bootstrap", authHandler.Bootstrap)
	protected.GET("/orders", orderHandler.List)
	protected.POST("/orders", orderHandler.Create)
	protected.PUT("/orders/:id", orderHandler.Update)
	protected.POST("/orders/:id/actions/toggle-step", orderHandler.ToggleStep)
	protected.POST("/orders/:id/actions/toggle-served", orderHandler.ToggleServed)
	protected.DELETE("/orders/:id", orderHandler.Delete)
	protected.POST("/orders/actions/clear", orderHandler.Clear)
	protected.POST("/realtime/session", realtimeHandler.CreateSession)
	protected.GET("/stats/daily", func(c *gin.Context) {
		authCtx, ok := auth.ContextFromGin(c)
		if !ok {
			httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
			return
		}
		statsHandler.Daily(c, authCtx.WorkspaceID)
	})

	return router
}
