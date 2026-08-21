package cmd

import (
	"github.com/spf13/cobra"
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Manage framework components",
	Long: `Manage Yogan Framework components.

Commands:
  list    List all available components
  add     Generate integration guide for a component

Example:
  go-ygctl component list
  go-ygctl component add database`,
}

func init() {
	rootCmd.AddCommand(componentCmd)
}
