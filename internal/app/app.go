package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
	"github.com/dongwlin/legero-backend/migrations"
)

type Application struct {
	config   *config.Config
	location *time.Location
	db       *bun.DB
	router   *gin.Engine
	server   *http.Server
}

func New(ctx context.Context, cfg *config.Config, appLogger zerolog.Logger) (*Application, error) {
	location, err := time.LoadLocation(cfg.BizTimezone)
	if err != nil {
		return nil, fmt.Errorf("load biz timezone: %w", err)
	}

	if err := migrations.Migrate(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	db, err := database.New(ctx, database.Options{DSN: cfg.DatabaseURL})
	if err != nil {
		return nil, err
	}

	realtimeBroker := realtime.NewBroker()
	realtimeSessions := realtime.NewSessionManager(cfg.RealtimeSessionTTL, time.Now)
	realtimeHandler := handler.NewRealtime(
		realtimeBroker,
		realtimeSessions,
		location,
		cfg.RealtimeHeartbeatInterval,
		cfg.WSWriteTimeout,
		cfg.WSReadTimeout,
		cfg.WSAllowedOrigins,
		time.Now,
	)

	orderService := service.NewOrder(
		db,
		location,
		realtimeBroker,
	)

	authService, err := service.NewAuth(
		db,
		orderService,
		crypto.NewPasswordHasher(cfg.Argon2),
		location,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.PasetoSymmetricKey,
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	statsService := service.NewStats(db, cfg.BizTimezone)

	authHandler := handler.NewAuth(authService, location)
	orderHandler := handler.NewOrder(orderService, location)
	statsHandler := handler.NewStats(statsService, location)

	router := newRouter(
		appLogger,
		authService,
		authHandler,
		orderHandler,
		statsHandler,
		realtimeHandler,
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Application{
		config:   cfg,
		location: location,
		db:       db,
		router:   router,
		server:   server,
	}, nil
}

// Run starts the HTTP server and blocks until a shutdown signal is received,
// then gracefully shuts down with a 30-second timeout.
func (a *Application) Run(ctx context.Context) error {
	handler := shutdown.New(ctx)

	go func() {
		log.Info().Str("addr", a.config.HTTPAddr).Msg("listening")
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("http server error")
		}
	}()

	<-handler.Done()
	log.Info().Msg("shutdown signal received")

	return handler.Shutdown(30*time.Second,
		func(ctx context.Context) error {
			log.Info().Msg("shutting down http server")
			if err := a.server.Shutdown(ctx); err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}
			return nil
		},
		func(ctx context.Context) error {
			log.Info().Msg("closing application resources")
			if err := a.db.Close(); err != nil {
				return fmt.Errorf("close application: %w", err)
			}
			return nil
		},
	)
}
