package cmd

import (
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	permissionWorkspace string
	permissionDomain    string
)

var permissionCmd = &cobra.Command{
	Use:   "permission",
	Short: "Manage permission declarations from DSL",
	Long: `Generate domain permission declaration files from defs/permissions/*.yaml.

The DSL is the source of truth. Generated files under
domains/<domain>/permissions/declared_permissions.go must not be edited by hand.

Examples:
  ygctl permission gen --domain rbac
  ygctl permission sync`,
}

var permissionGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate one domain permission declaration from DSL",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, err := resolvePermissionWorkspace(permissionWorkspace)
		if err != nil {
			return err
		}
		if err := generator.SyncPermissionDeclaration(workspace, permissionDomain); err != nil {
			return err
		}
		color.Green("✅ Permission declaration generated: %s", permissionDomain)
		return nil
	},
}

var permissionSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate all domain permission declarations from DSL",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, err := resolvePermissionWorkspace(permissionWorkspace)
		if err != nil {
			return err
		}
		if err := generator.SyncPermissionDeclarations(workspace); err != nil {
			return err
		}
		color.Green("✅ Permission declarations synced from %s", filepath.Join(workspace, "defs", "permissions"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(permissionCmd)
	permissionCmd.AddCommand(permissionGenCmd)
	permissionCmd.AddCommand(permissionSyncCmd)
	permissionCmd.PersistentFlags().StringVar(&permissionWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	permissionGenCmd.Flags().StringVar(&permissionDomain, "domain", "", "Target domain key (required)")
	_ = permissionGenCmd.MarkFlagRequired("domain")
}

func resolvePermissionWorkspace(workspace string) (string, error) {
	if workspace == "" {
		return generator.FindWorkspaceRoot("")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	return abs, nil
}
