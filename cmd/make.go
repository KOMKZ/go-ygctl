package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	makeWorkspace string
	makeApp       string
	makeDryRun    bool
)

var makeCmd = &cobra.Command{
	Use:   "make",
	Short: "Manage Makefile and scripts of workspace apps",
	Long: `Analyze each app under apps/ and keep scripts/ and Makefile in sync:

1. Generate missing standard scripts (run/test/test-coverage/clean)
2. Rewrite inline standard targets as thin script wrappers
3. Append Makefile targets for scripts that lack one
4. Update .PHONY

Idempotent; use --dry-run to preview changes.

Example:
  ygctl make sync
  ygctl make sync --app hrise-admin-api --dry-run`,
}

var makeSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Makefile targets with scripts of all apps",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.MakeSyncConfig{
			WorkspacePath: makeWorkspace,
			AppName:       makeApp,
			DryRun:        makeDryRun,
		}
		result, err := cfg.Sync()
		if err != nil {
			return err
		}

		if makeDryRun {
			color.Yellow("Dry run — no files written.")
		}
		if len(result.Changes) == 0 {
			color.Green("✅ No changes needed; apps are in sync.")
			return nil
		}
		for _, ch := range result.Changes {
			fmt.Printf("  [%s] %-12s %s\n", ch.App, ch.Kind, ch.Detail)
		}
		if !makeDryRun {
			color.Green("✅ Synced %d app(s).", len(result.Apps))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(makeCmd)
	makeCmd.AddCommand(makeSyncCmd)
	makeCmd.PersistentFlags().StringVar(&makeWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	makeCmd.PersistentFlags().StringVar(&makeApp, "app", "", "Target app only (default: all apps with a Makefile)")
	makeCmd.PersistentFlags().BoolVar(&makeDryRun, "dry-run", false, "Preview changes without writing")
}
