package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	apiWorkspace string
	apiDefFile   string
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
		path, err := generator.InitDefFile(apiWorkspace, args[0])
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

func init() {
	apiCmd.AddCommand(apiInitCmd)
	daoCmd.AddCommand(daoGenCmd)
	apiCmd.AddCommand(apiGenCmd)
	for _, c := range []*cobra.Command{daoGenCmd, apiGenCmd} {
		c.Flags().StringVarP(&apiDefFile, "file", "f", "", "Api def file (required)")
		_ = c.MarkFlagRequired("file")
	}
}
