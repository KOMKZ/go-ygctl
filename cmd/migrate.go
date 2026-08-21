package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	migrateApp       string
	migrateWorkspace string
	migrateDomain    string
	migrateTable     string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage centralized migrations in apps/<app>/migrations",
	Long: `Manage database migrations.

Migrations are centralized in the app (apps/<app>/migrations), NOT in domains.
File naming: NNNNNN_<domain>_<desc>.up.sql / .down.sql.

Commands:
  create   Create a migration file pair
  up       Apply pending migrations
  down     Roll back migrations
  version  Show current version
  list     List migration files

Databases are never auto-created: before up/down the target database is
probed and, if missing, you are prompted to create it manually.`,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.PersistentFlags().StringVar(&migrateApp, "app", "", "Target app (default: auto-detect the sole app with migrations/)")
	migrateCmd.PersistentFlags().StringVar(&migrateWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create <desc>",
	Short: "Create a migration file pair",
	Long: `Create NNNNNN_<domain>_<desc>.up.sql and .down.sql in apps/<app>/migrations/.
The sequence number is max+1 of existing files.

Example:
  ygctl migrate create add_avatar_to_admins --domain admin --table admins`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.MigrateConfig{
			WorkspacePath: migrateWorkspace,
			AppName:       migrateApp,
			DomainKey:     migrateDomain,
			Desc:          args[0],
			TableName:     migrateTable,
		}
		up, down, err := cfg.CreateMigration()
		if err != nil {
			return err
		}
		color.Green("✅ Migration created:")
		fmt.Printf("  %s\n  %s\n", up, down)
		fmt.Println()
		color.Yellow("Edit the SQL, then run: ygctl migrate up")
		return nil
	},
}

func newMigrateRunCmd(use, short, scriptCmd string, maxArgs int) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MaximumNArgs(maxArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptArgs := append([]string{scriptCmd}, args...)
			return generator.RunMigrateScript(migrateWorkspace, migrateApp, scriptArgs...)
		},
	}
}

func init() {
	migrateCmd.AddCommand(migrateCreateCmd)
	migrateCreateCmd.Flags().StringVar(&migrateDomain, "domain", "", "Domain key (prefixes the filename)")
	migrateCreateCmd.Flags().StringVar(&migrateTable, "table", "", "Table name for the CREATE TABLE skeleton")
	_ = migrateCreateCmd.MarkFlagRequired("domain")

	migrateCmd.AddCommand(newMigrateRunCmd("up [N]", "Apply pending migrations", "up", 1))
	migrateCmd.AddCommand(newMigrateRunCmd("down [N]", "Roll back migrations", "down", 1))
	migrateCmd.AddCommand(newMigrateRunCmd("version", "Show current migration version", "version", 0))

	migrateCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List migration files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.ListMigrations(migrateWorkspace, migrateApp)
		},
	})
}
