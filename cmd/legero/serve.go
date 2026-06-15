package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/app"
	"github.com/dongwlin/legero-backend/internal/infra/config"
	"github.com/dongwlin/legero-backend/internal/infra/logger"
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

	// Bootstrap application
	application, err := app.New(ctx, cfg, appLogger)
	if err != nil {
		return fmt.Errorf("bootstrap app: %w", err)
	}

	// Run blocks until shutdown signal, then gracefully stops
	return application.Run(ctx)
}
