package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	domainMigrateSource string
	domainMigrateApp    string
)

var domainMigrateCmd = &cobra.Command{
	Use:   "migrate <name> --source <legacy-domain-dir>",
	Short: "Migrate a legacy domain (go-yogan-domain-*) into the workspace",
	Long: `Migrate a legacy domain into domains/<name> with the new structure:

1. Detect table-less domains (no model/repository -> --pure skeleton)
2. Generate the new skeleton (go.mod, CLAUDE.md, port/, ...)
3. Copy legacy files, replacing the old module path with the new import path
4. Resolve error module number conflicts (keep legacy number if free)
5. Centralize legacy migrations into apps/<app>/migrations/ (NNNNNN_<domain>_<desc>)
6. gofmt + tidy + build + tests

Example:
  ygctl domain migrate auth --source ../yogan-domains/go-yogan-domain-auth`,
	Args: cobra.ExactArgs(1),
	RunE: runDomainMigrate,
}

func init() {
	domainCmd.AddCommand(domainMigrateCmd)
	domainMigrateCmd.Flags().StringVar(&domainMigrateSource, "source", "", "Legacy domain directory (required)")
	domainMigrateCmd.Flags().StringVar(&domainMigrateApp, "app", "", "Target app for centralized migrations (default: auto-detect)")
	_ = domainMigrateCmd.MarkFlagRequired("source")
}

func runDomainMigrate(cmd *cobra.Command, args []string) error {
	config := &generator.DomainMigrateConfig{
		WorkspacePath: domainWorkspace,
		DomainKey:     args[0],
		SourcePath:    domainMigrateSource,
		AppName:       domainMigrateApp,
		EntityName:    domainEntity,
	}

	color.Cyan("\n🚀 Migrating domain: %s", config.DomainKey)

	result, err := generator.NewDomainMigrator(config).Migrate()
	if err != nil {
		return err
	}

	color.Green("✅ Domain migrated successfully!")
	fmt.Printf("   Path:         %s\n", result.DomainDir)
	fmt.Printf("   Pure (no db): %v\n", result.Pure)
	fmt.Printf("   Copied:       %d files (%d rewritten with new import path)\n", result.CopiedFiles, result.ReplacedFiles)
	if len(result.MigratedSQLFiles) > 0 {
		fmt.Println("   Centralized migrations (app):")
		for _, f := range result.MigratedSQLFiles {
			abs, _ := filepath.Abs(filepath.Join(config.WorkspacePath, "apps", domainMigrateApp, "migrations", f))
			fmt.Printf("     %s\n", abs)
		}
	}
	return nil
}
