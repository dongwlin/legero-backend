package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/dongwlin/legero-backend/internal/app"
	"github.com/dongwlin/legero-backend/internal/infra/config"
)

func run(ctx context.Context, cfg *config.Config, appLogger zerolog.Logger, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("server does not accept arguments")
	}

	return runHTTPServer(ctx, cfg, appLogger)
}

func runHTTPServer(ctx context.Context, cfg *config.Config, appLogger zerolog.Logger) error {
	application, err := app.New(ctx, cfg, appLogger)
	if err != nil {
		return fmt.Errorf("bootstrap app: %w", err)
	}
	defer func() {
		_ = application.Close()
	}()

	go func() {
		<-ctx.Done()
		if err := application.Server.Shutdown(context.Background()); err != nil {
			appLogger.Error().Err(err).Msg("shutdown http server")
		}
	}()

	appLogger.Info().Str("addr", cfg.HTTPAddr).Msg("listening")
	if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
