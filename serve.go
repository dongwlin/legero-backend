package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/app"
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
	// Create app context
	ctx := context.Background()

	// Bootstrap infrastructure (config, DB, migrations, logger)
	infra, err := app.NewInfra(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap infra: %w", err)
	}

	// Bootstrap application (services, handlers, router)
	application, err := app.New(infra)
	if err != nil {
		_ = infra.Close()
		return fmt.Errorf("bootstrap app: %w", err)
	}

	// Run blocks until shutdown signal, then gracefully stops
	return application.Run(ctx)
}
