package cmd

import (
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new application",
	Long:  `Create a new Yogan Framework application from template.`,
}

func init() {
	rootCmd.AddCommand(newCmd)
}
