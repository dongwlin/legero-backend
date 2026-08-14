package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dongwlin/legero-backend/internal/infra/config"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "Print the version and build information of legero.",
	Run: func(cmd *cobra.Command, args []string) {
		bi, _ := cmd.Flags().GetBool("build-info")
		fmt.Printf("%s\n", config.Version)
		if bi {
			fmt.Printf("Commit:     %s\n", config.Commit)
			fmt.Printf("Build Time: %s\n", config.BuildTime)
			fmt.Printf("Go Version: %s\n", config.GoVersion)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("build-info", "b", false, "Show detailed build information")
}
