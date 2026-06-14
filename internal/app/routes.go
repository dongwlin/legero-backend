package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/middleware"
	"github.com/dongwlin/legero-backend/internal/service"
)

func newRouter(
	appLogger zerolog.Logger,
	authService *service.Auth,
	authHandler *handler.Auth,
	orderHandler *handler.Order,
	statsHandler *handler.Stats,
	realtimeHandler *handler.Realtime,
) *gin.Engine {
	router := gin.New()
	router.Use(
		middleware.CORS(),
		middleware.Logger(appLogger),
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
	protected.Use(middleware.Auth(authService))
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
		authCtx, ok := handler.AuthContext(c)
		if !ok {
			httpx.AbortError(c, httpx.UnauthorizedError("missing auth context"))
			return
		}
		statsHandler.Daily(c, authCtx.WorkspaceID)
	})

	return router
}
