package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	modelDomain    string
	modelWorkspace string
	modelForce     bool
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Generate models",
	Long: `Generate gorm models.

Commands:
  from-table   Generate a model from an existing database table`,
}

var modelFromTableCmd = &cobra.Command{
	Use:   "from-table <table>",
	Short: "Generate a gorm model from a database table",
	Long: `Read information_schema and generate a gorm model into
domains/<domain>/model/<table>.go.

Type mapping: tinyint->int8 (tinyint(1)->bool), smallint->int16, int->int,
bigint->int64, unsigned variants->uintN, decimal/float/double->float64,
varchar/char/text/json->string, datetime/timestamp/date->time.Time,
blob->[]byte. Nullable time columns without default become pointers.

DB connection via env: DB_HOST, DB_PORT, DB_USER, DB_PASS, DB_NAME
(same defaults as ygctl migrate).

Example:
  ygctl model from-table admins --domain admin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.ModelFromTableConfig{
			WorkspacePath: modelWorkspace,
			DomainKey:     modelDomain,
			TableName:     args[0],
			Force:         modelForce,
		}
		outPath, err := cfg.GenerateModelFromTable()
		if err != nil {
			return err
		}
		color.Green("✅ Model generated: %s", outPath)
		fmt.Println()
		color.Yellow("Review the generated file: type mapping (tinyint/decimal/json), defaults and pointers.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(modelCmd)
	modelCmd.AddCommand(modelFromTableCmd)
	modelFromTableCmd.Flags().StringVar(&modelDomain, "domain", "", "Target domain key (required)")
	modelFromTableCmd.Flags().StringVar(&modelWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	modelFromTableCmd.Flags().BoolVar(&modelForce, "force", false, "Overwrite existing model file")
	_ = modelFromTableCmd.MarkFlagRequired("domain")
}
