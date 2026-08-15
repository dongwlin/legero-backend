package cmd

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
	ctx := context.Background()

	// Wire the application (config, DB, services, handlers, router).
	application, cleanup, err := app.InitializeApplication()
	if err != nil {
		return fmt.Errorf("bootstrap app: %w", err)
	}
	defer cleanup()

	// Run blocks until shutdown signal, then gracefully stops
	return application.Run(ctx)
}
