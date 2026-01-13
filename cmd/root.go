package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var version = "0.6.0"

var rootCmd = &cobra.Command{
	Use:   "go-ygctl",
	Short: "Yogan Framework CLI Tool",
	Long: `go-ygctl is a CLI tool for generating Yogan Framework applications.

Commands:
  new        Create a new application
  component  Manage framework components

Supported application types:
  - HTTP (Gin-based web application)
  - CLI  (Command-line application) [coming soon]
  - gRPC (gRPC server/client)       [coming soon]

Examples:
  go-ygctl new http              # Create a new HTTP application
  go-ygctl component list        # List available components
  go-ygctl component add database # Generate integration guide`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("go-ygctl version %s\n", version))
}
