package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"

	"github.com/dongwlin/legero-backend/internal/auth"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/database"
	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/stats"
	"github.com/dongwlin/legero-backend/migrations"
)

type Application struct {
	Config   *config.Config
	Location *time.Location
	DB       *bun.DB
	Router   *gin.Engine
	Server   *http.Server
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
	realtimeHandler := realtime.NewHandler(
		realtimeBroker,
		realtimeSessions,
		location,
		cfg.RealtimeHeartbeatInterval,
		cfg.WSWriteTimeout,
		cfg.WSReadTimeout,
		cfg.WSAllowedOrigins,
		time.Now,
	)

	statsRepo := &stats.BunRepository{}

	orderService := order.NewService(
		db,
		location,
		realtimeBroker,
	)

	authService, err := auth.NewService(
		db,
		orderService,
		auth.NewPasswordHasher(cfg.Argon2),
		location,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.PasetoSymmetricKey,
	)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	statsService := stats.NewService(db, statsRepo, cfg.BizTimezone)

	authHandler := auth.NewHandler(authService, location)
	orderHandler := order.NewHandler(orderService, location)
	statsHandler := stats.NewHandler(statsService, location)

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
		Config:   cfg,
		Location: location,
		DB:       db,
		Router:   router,
		Server:   server,
	}, nil
}

func (a *Application) Close() error {
	return a.DB.Close()
}
