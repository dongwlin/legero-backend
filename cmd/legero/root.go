package main

import (
	"github.com/spf13/cobra"
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
