package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/handler/v1"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/httpx"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/middleware"
	"github.com/dongwlin/legero-backend/internal/service"
)

// NewRouter builds the gin engine with global middleware and registers all routes.
func NewRouter(
	authSvc service.Auth,
	orderSvc service.Order,
	statsSvc service.Stats,
	broker *realtime.Broker,
	sessions *realtime.SessionManager,
	location *time.Location,
	cfg *config.Config,
	appLogger zerolog.Logger,
	now func() time.Time,
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

	v1.RegisterRoutes(
		router.Group("/api"),
		authSvc,
		orderSvc,
		statsSvc,
		broker,
		sessions,
		location,
		cfg,
		now,
	)

	return router
}
