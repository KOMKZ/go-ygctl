package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	apiWorkspace     string
	apiDefFile       string
	apiInitDomain    string
	apiInitTable     string
	apiInitRouteBase string
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Generate domains and API layers from an api def (DSL)",
	Long: `DSL-driven generation chain:

  api init <entity>          Generate a def template at defs/<entity>.yaml
  dao gen -f defs/x.yaml     Generate model/repository/errors from the def
  api gen -f defs/x.yaml     Generate service/handler/dto/router/migration from the def

Generation is deterministic (text/template, no AI): the skeleton carries no
business logic. Fill the def, generate, then implement business rules.`,
}

var daoCmd = &cobra.Command{
	Use:   "dao",
	Short: "Generate the DAO layer (model/repository/errors) from an api def",
}

func init() {
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(daoCmd)
	apiCmd.PersistentFlags().StringVar(&apiWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	daoCmd.PersistentFlags().StringVar(&apiWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
}

var apiInitCmd = &cobra.Command{
	Use:   "init <entity>",
	Short: "Generate a def template at defs/<entity>.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := generator.InitDefFileWithConfig(generator.InitDefConfig{
			WorkspacePath: apiWorkspace,
			Domain:        apiInitDomain,
			Entity:        args[0],
			Table:         apiInitTable,
			RouteBase:     apiInitRouteBase,
		})
		if err != nil {
			return err
		}
		color.Green("✅ Def template created: %s", path)
		fmt.Println()
		color.Yellow("Fill in the business fields, then run: ygctl dao gen -f " + path)
		return nil
	},
}

var daoGenCmd = &cobra.Command{
	Use:   "gen -f <def>",
	Short: "Generate the DAO layer (model/repository/errors) from a def",
	Long: `Generate the DAO layer into domains/<domain>/:

  model/<entity>.go               gorm entity (fields from def)
  errors/errors.go                module number + not_found / xxx_exists codes
  repository/repository.go        interface (CRUD + def queries)
  repository/repository_mysql.go  BaseRepository implementation + compile assert

If domains/<domain> does not exist, a skeleton is initialized first
(pure is auto-detected: dao gen always produces a table-backed domain).
Then: gofmt + go mod tidy + go build verification.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.DAOGenConfig{
			WorkspacePath: apiWorkspace,
			DefFile:       apiDefFile,
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ DAO layer generated: %s", result.DomainDir)
		fmt.Println()
		color.Yellow("Next: ygctl api gen -f " + apiDefFile)
		return nil
	},
}

var apiGenCmd = &cobra.Command{
	Use:   "gen -f <def>",
	Short: "Generate the API layer (service/handler/dto/router/migration) from a def",
	Long: `Generate the API layer on top of the DAO layer:

  domains/<domain>/service/<entity>_service.go   CRUD skeleton (no business rules)
  domains/<domain>/provider/do/provider.go       DI registration
  domains/<domain>/permissions/                  permission codes per endpoint
  domains/<domain>/contract/contract.md          service method contract
  apps/<app>/internal/module/<entity>/           handler/dto/provider skeleton
  apps/<app>/internal/router/<entity>_router.go  route registration
  apps/<app>/migrations/NNNNNN_<domain>_create_<table>.sql  (generated, NOT applied)

Then: gofmt + go mod tidy + go build verification.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.APIGenConfig{
			WorkspacePath: apiWorkspace,
			DefFile:       apiDefFile,
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ API layer generated: %s", result.DomainDir)
		return nil
	},
}

var apiSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Scan all backend defs in the workspace and regenerate DAO/API layers",
	Long: `Scan <workspace>/defs/*.yaml (excluding *.ui.yaml), then run
dao gen + api gen for each def in stable order.

This is the workspace-level append/regenerate path: backend def files are treated
as the source of truth for table-backed API skeletons. All permission declaration
files are regenerated from defs/permissions/*.yaml and should not be edited by hand.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace := apiWorkspace
		if workspace == "" {
			root, err := generator.FindWorkspaceRoot("")
			if err != nil {
				return err
			}
			workspace = root
		} else {
			abs, err := filepath.Abs(workspace)
			if err != nil {
				return err
			}
			workspace = abs
		}

		defFiles, err := generator.ListDefFiles(workspace)
		if err != nil {
			return err
		}
		if len(defFiles) == 0 {
			color.Yellow("No backend def files found under %s/defs", workspace)
		} else {
			color.Cyan("\n🔄 Syncing workspace defs from %s", filepath.Join(workspace, "defs"))
			for _, defFile := range defFiles {
				color.Yellow("  • %s", filepath.Base(defFile))
				daoResult, err := (&generator.DAOGenConfig{
					WorkspacePath: workspace,
					DefFile:       defFile,
				}).Generate()
				if err != nil {
					return fmt.Errorf("dao gen failed for %s: %w", defFile, err)
				}
				_ = daoResult
				if _, err := (&generator.APIGenConfig{
					WorkspacePath: workspace,
					DefFile:       defFile,
				}).Generate(); err != nil {
					return fmt.Errorf("api gen failed for %s: %w", defFile, err)
				}
			}
		}

		if err := generator.SyncPermissionDeclarations(workspace); err != nil {
			return fmt.Errorf("permission sync failed: %w", err)
		}

		color.Green("✅ Workspace defs synced: %d file(s)", len(defFiles))
		return nil
	},
}

func init() {
	apiCmd.AddCommand(apiInitCmd)
	daoCmd.AddCommand(daoGenCmd)
	apiCmd.AddCommand(apiGenCmd)
	apiCmd.AddCommand(apiSyncCmd)
	for _, c := range []*cobra.Command{daoGenCmd, apiGenCmd} {
		c.Flags().StringVarP(&apiDefFile, "file", "f", "", "Api def file (required)")
		_ = c.MarkFlagRequired("file")
	}
	apiInitCmd.Flags().StringVar(&apiInitDomain, "domain", "", "Domain key (default: entity)")
	apiInitCmd.Flags().StringVar(&apiInitTable, "table", "", "Database table name (default: <entity>s)")
	apiInitCmd.Flags().StringVar(&apiInitRouteBase, "route-base", "", "API route base below /api (default: /<table>)")
}
