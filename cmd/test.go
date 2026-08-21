package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	testWorkspace string
	testApp       string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Generate e2e interface test skeletons",
	Long: `Generate e2e interface test skeletons under apps/<app>/test/<module>/:

  setup_test.go    TestMain + shared test server shutdown
  <module>_test.go normal-flow test skeleton (fill requests/assertions)

Example:
  ygctl test gen auth --app hrise-admin-api`,
}

var testGenCmd = &cobra.Command{
	Use:   "gen <module>",
	Short: "Generate e2e test skeleton for a module",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.TestGenConfig{
			WorkspacePath: testWorkspace,
			AppName:       testApp,
			ModuleName:    args[0],
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ Test skeleton generated: %s", result.TestDir)
		fmt.Println()
		color.Yellow("Fill the normal-flow requests and assertions, then run: go test ./apps/<app>/test/<module>/...")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.AddCommand(testGenCmd)
	testCmd.PersistentFlags().StringVar(&testWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	testCmd.PersistentFlags().StringVar(&testApp, "app", "", "Target app (default: auto-detect)")
}
