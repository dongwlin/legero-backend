package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
	"github.com/dongwlin/legero-backend/internal/realtime"
	"github.com/dongwlin/legero-backend/internal/service"
)

type Application struct {
	infra  *Infra
	router *gin.Engine
	server *http.Server
}

func New(infra *Infra) (*Application, error) {
	realtimeBroker := realtime.NewBroker()
	realtimeSessions := realtime.NewSessionManager(infra.Config.RealtimeSessionTTL, time.Now)
	realtimeHandler := handler.NewRealtime(
		realtimeBroker,
		realtimeSessions,
		infra.Location,
		infra.Config.RealtimeHeartbeatInterval,
		infra.Config.WSWriteTimeout,
		infra.Config.WSReadTimeout,
		infra.Config.WSAllowedOrigins,
		time.Now,
	)

	orderService := service.NewOrder(
		infra.DB,
		infra.Location,
		realtimeBroker,
	)

	authService, err := service.NewAuth(
		infra.DB,
		orderService,
		crypto.NewPasswordHasher(infra.Config.Argon2),
		infra.Location,
		infra.Config.AccessTokenTTL,
		infra.Config.RefreshTokenTTL,
		infra.Config.PasetoSymmetricKey,
	)
	if err != nil {
		return nil, err
	}

	statsService := service.NewStats(infra.DB, infra.Config.BizTimezone)

	authHandler := handler.NewAuth(authService, infra.Location)
	orderHandler := handler.NewOrder(orderService, infra.Location)
	statsHandler := handler.NewStats(statsService, infra.Location)

	router := newRouter(
		infra.Logger,
		authService,
		authHandler,
		orderHandler,
		statsHandler,
		realtimeHandler,
	)

	server := &http.Server{
		Addr:              infra.Config.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Application{
		infra:  infra,
		router: router,
		server: server,
	}, nil
}

// Run starts the HTTP server and blocks until a shutdown signal is received,
// then gracefully shuts down with a 30-second timeout.
func (a *Application) Run(ctx context.Context) error {
	handler := shutdown.New(ctx)

	go func() {
		log.Info().Str("addr", a.infra.Config.HTTPAddr).Msg("listening")
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
			if err := a.infra.Close(); err != nil {
				return fmt.Errorf("close application: %w", err)
			}
			return nil
		},
	)
}
