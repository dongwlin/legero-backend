package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
)

type Application struct {
	infra  *Infra
	server *http.Server
}

func New(infra *Infra) (*Application, error) {
	services, err := newServices(infra)
	if err != nil {
		return nil, err
	}

	realtimeHandler := handler.NewRealtime(
		infra.Broker, infra.Sessions,
		infra.Location,
		infra.Config.RealtimeHeartbeatInterval,
		infra.Config.WSWriteTimeout, infra.Config.WSReadTimeout,
		infra.Config.WSAllowedOrigins,
		time.Now,
	)

	authHandler := handler.NewAuth(services.Auth, infra.Location)
	orderHandler := handler.NewOrder(services.Order, infra.Location)
	statsHandler := handler.NewStats(services.Stats, infra.Location)

	router := newRouter(
		infra.Logger,
		services.Auth,
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
