package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/dongwlin/legero-backend/internal/handler"
	"github.com/dongwlin/legero-backend/internal/infra/crypto"
	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
	servicev1 "github.com/dongwlin/legero-backend/internal/service/v1"
)

type Application struct {
	infra  *Infra
	server *http.Server
}

func New(infra *Infra) (*Application, error) {
	orderSvc := servicev1.NewOrder(infra.DB, infra.Location, infra.Broker)

	authSvc, err := servicev1.NewAuth(
		infra.DB,
		orderSvc,
		crypto.NewPasswordHasher(infra.Config.Argon2),
		infra.Location,
		infra.Config.AccessTokenTTL,
		infra.Config.RefreshTokenTTL,
		infra.Config.PasetoSymmetricKey,
	)
	if err != nil {
		return nil, err
	}

	statsSvc := servicev1.NewStats(infra.DB, infra.Config.BizTimezone)

	router := handler.NewRouter(
		authSvc,
		orderSvc,
		statsSvc,
		infra.Broker,
		infra.Sessions,
		infra.Location,
		infra.Config,
		infra.Logger,
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
	graceful := shutdown.New(ctx)

	go func() {
		log.Info().Str("addr", a.infra.Config.HTTPAddr).Msg("listening")
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("http server error")
		}
	}()

	<-graceful.Done()
	log.Info().Msg("shutdown signal received")

	return graceful.Shutdown(30*time.Second,
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
