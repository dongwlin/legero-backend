package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/infra/logger"
)

// Persistent flags (available to all subcommands)
const (
	flagConfig = "config"
)

var rootCmd = &cobra.Command{
	Use:   "legero",
	Short: "Legero restaurant order management backend",
	Long:  "A backend service for managing restaurant orders, workspaces, and real-time updates.",
	// Default to running the server when no subcommand is specified
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHTTPServer()
	},
}

func init() {
	rootCmd.PersistentFlags().StringP(flagConfig, "c", "", "config file path (default: config/config.yaml)")
}

// Execute runs the root command.
//
// The global logger is initialized first so that every command — config
// loading, database migrations, and the create-user bootstrap — logs through
// the configured ConsoleWriter instead of zerolog's default logger.
func Execute() error {
	logger.New()
	return rootCmd.Execute()
}
