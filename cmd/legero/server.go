package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/app"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
	"github.com/dongwlin/legero-backend/internal/infra/shutdown"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the Legero HTTP server with graceful shutdown support.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHTTPServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runHTTPServer() error {
	// Initialize logger (sets global log.Logger)
	appLogger := logger.New()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create app context
	ctx := context.Background()

	// Create shutdown handler
	handler := shutdown.New(ctx)

	// Bootstrap application
	application, err := app.New(ctx, cfg, appLogger)
	if err != nil {
		return fmt.Errorf("bootstrap app: %w", err)
	}

	// Start HTTP server in goroutine
	go func() {
		log.Info().Str("addr", cfg.HTTPAddr).Msg("listening")
		if err := application.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("http server error")
		}
	}()

	// Wait for shutdown signal
	<-handler.Done()
	log.Info().Msg("shutdown signal received")

	// Graceful shutdown with timeout
	return handler.Shutdown(30*time.Second, func(ctx context.Context) error {
		log.Info().Msg("shutting down http server")
		if err := application.Server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	}, func(ctx context.Context) error {
		log.Info().Msg("closing application resources")
		if err := application.Close(); err != nil {
			return fmt.Errorf("close application: %w", err)
		}
		return nil
	})
}
