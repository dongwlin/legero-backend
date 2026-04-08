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
	"github.com/dongwlin/legero-backend/internal/infra/clock"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	dbpkg "github.com/dongwlin/legero-backend/internal/infra/db"
	"github.com/dongwlin/legero-backend/internal/infra/ids"
	"github.com/dongwlin/legero-backend/internal/order"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/stats"
	"github.com/dongwlin/legero-backend/internal/workspace"
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

	database, err := dbpkg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	systemClock := clock.SystemClock{}
	idGenerator := ids.UUIDGenerator{}
	realtimeBroker := realtime.NewBroker()
	realtimeSessions := realtime.NewSessionManager(cfg.RealtimeSessionTTL, systemClock.Now)
	realtimeHandler := realtime.NewHandler(
		realtimeBroker,
		realtimeSessions,
		location,
		cfg.RealtimeHeartbeatInterval,
		cfg.WSWriteTimeout,
		cfg.WSReadTimeout,
		cfg.WSAllowedOrigins,
		systemClock.Now,
	)

	userRepo := &auth.BunUserRepository{}
	refreshRepo := &auth.BunRefreshTokenRepository{}
	workspaceRepo := &workspace.BunRepository{}
	orderRepo := &order.BunRepository{}
	counterRepo := &order.BunCounterRepository{}
	statsRepo := &stats.BunRepository{}

	orderService := order.NewService(
		database,
		orderRepo,
		counterRepo,
		systemClock,
		idGenerator,
		location,
		realtimeBroker,
	)

	authService, err := auth.NewService(
		database,
		userRepo,
		refreshRepo,
		workspaceRepo,
		orderService,
		auth.NewPasswordHasher(cfg.Argon2),
		systemClock,
		idGenerator,
		location,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
		cfg.PasetoSymmetricKey,
	)
	if err != nil {
		_ = database.Close()
		return nil, err
	}

	statsService := stats.NewService(database, statsRepo, cfg.BizTimezone)

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
		DB:       database,
		Router:   router,
		Server:   server,
	}, nil
}

func (a *Application) Close() error {
	return a.DB.Close()
}
