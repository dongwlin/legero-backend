package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/apperr"
	"github.com/dongwlin/legero-backend/internal/handler/httpresp"
	"github.com/dongwlin/legero-backend/internal/handler/middleware"
	"github.com/dongwlin/legero-backend/internal/handler/v1"
	"github.com/dongwlin/legero-backend/internal/handler/v2"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

const v2RoutePrefix = "/api/v2"

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
		middleware.PerformanceTiming(),
		gin.CustomRecovery(func(c *gin.Context, _ any) {
			if !isV2Request(c) {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			// Once headers or a body have been sent, appending an error would
			// corrupt the response and cannot change its HTTP status.
			if c.Writer.Written() {
				c.Abort()
				return
			}
			httpresp.AbortError(c, apperr.InternalError(http.StatusText(http.StatusInternalServerError), nil))
		}),
	)
	router.NoRoute(func(c *gin.Context) {
		if isV2Request(c) {
			httpresp.AbortError(c, apperr.NotFoundError("route_not_found", http.StatusText(http.StatusNotFound)))
		}
	})
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

	v2.RegisterRoutes(
		router.Group(v2RoutePrefix),
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

func isV2Request(c *gin.Context) bool {
	path := c.Request.URL.Path
	return path == v2RoutePrefix || strings.HasPrefix(path, v2RoutePrefix+"/")
}
