package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
)

// Application is the assembled HTTP application.
type Application struct {
	server *http.Server
}

// NewApplication creates an Application from the wired HTTP server.
func NewApplication(server *http.Server) *Application {
	return &Application{server: server}
}

// Run starts the HTTP server and blocks until a shutdown signal is received,
// then gracefully shuts down with a 30-second timeout.
func (a *Application) Run(ctx context.Context) error {
	graceful := shutdown.New(ctx)

	go func() {
		log.Info().Str("addr", a.server.Addr).Msg("listening")
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
	)
}
