package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	domainEntity    string
	domainWorkspace string
	domainModule    string
	domainPure      bool
	domainJobDomain string
	domainJobTask   string
	domainJobMode   string
)

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage domains in a multi-app workspace",
	Long: `Manage domain packages under <workspace>/domains/<name>.

Domains are ordinary packages of the workspace root module (no per-domain
go.mod). Migrations are NOT stored in domains — they live centralized in
apps/<app>/migrations/ (see "ygctl migrate").

Commands:
  init    Generate a full domain skeleton`,
}

var domainInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Generate a full domain skeleton under <workspace>/domains/<name>",
	Long: `Generate the complete skeleton of a domain package:

  domains/<name>/
  ├── CLAUDE.md                     # domain usage & rules (for AI dev)
  ├── errors/ errors.go             # error codes (module number auto-allocated)
  ├── model/                        # gorm entity placeholder
  ├── repository/                   # interface + MySQL impl (BaseRepository)
  ├── service/                      # domain logic placeholder + test
  ├── provider/do/                  # DI providers
  ├── event/ permissions/ policy/ assembler/ contract/

After generation: gofmt, go mod tidy and go build are run to verify the
skeleton compiles. No business logic is generated.

Example:
  ygctl domain init auth
  ygctl domain init verification-code --entity code`,
	Args: cobra.ExactArgs(1),
	RunE: runDomainInit,
}

func init() {
	rootCmd.AddCommand(domainCmd)
	domainCmd.AddCommand(domainInitCmd)
	domainCmd.AddCommand(domainJobCmd)
	domainJobCmd.AddCommand(domainJobAddCmd)
	domainInitCmd.Flags().StringVar(&domainEntity, "entity", "", "Entity name (default: singularized domain key)")
	domainInitCmd.Flags().StringVar(&domainWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	domainInitCmd.Flags().StringVar(&domainModule, "module", "", "Module path base (default: derived from first app under apps/)")
	domainInitCmd.Flags().BoolVar(&domainPure, "pure", false, "Table-less logic domain: skip model/repository skeleton")
	domainJobAddCmd.Flags().StringVar(&domainJobDomain, "domain", "", "Domain name, e.g. export")
	domainJobAddCmd.Flags().StringVar(&domainJobTask, "task", "", "Task type, e.g. export.demo")
	domainJobAddCmd.Flags().StringVar(&domainJobMode, "mode", "queue", "Default producer mode: queue or sync")
	domainJobAddCmd.Flags().StringVar(&domainWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	domainJobAddCmd.Flags().StringVar(&domainModule, "module", "", "Module path base (default: derived from first app under apps/)")
}

var domainJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Manage domain-owned job publisher and executor files",
}

var domainJobAddCmd = &cobra.Command{
	Use:   "add --domain <domain> --task <task-type>",
	Short: "Generate domain-owned job publisher and executor files",
	Long: `Generate job publisher and executor files under domains/<domain>/job.

Example:
  ygctl domain job add --domain export --task export.demo --mode queue`,
	Args: cobra.NoArgs,
	RunE: runDomainJobAdd,
}

func runDomainInit(cmd *cobra.Command, args []string) error {
	config := &generator.DomainConfig{
		DomainKey:     args[0],
		EntityName:    domainEntity,
		WorkspacePath: domainWorkspace,
		ModuleBase:    domainModule,
		Pure:          domainPure,
	}

	color.Cyan("\n🚀 Generating domain: %s", config.DomainKey)

	gen := generator.NewDomainGenerator(config)
	info, err := gen.Generate()
	if err != nil {
		return err
	}

	color.Green("✅ Domain generated successfully!")
	fmt.Printf("   Path:        %s\n", info.DomainDir)
	fmt.Printf("   Import:      %s\n", info.DomainImport)
	fmt.Printf("   Err module:  %d\n", info.ErrModule)
	fmt.Println()
	color.Yellow("Next steps:")
	step := 1
	if !config.Pure {
		fmt.Printf("  %d. Fill in model (ygctl model from-table <table> --domain %s)\n", step, config.DomainKey)
		step++
	}
	fmt.Printf("  %d. Add domain logic in service/\n", step)
	fmt.Printf("  %d. Wire providers in the app, register routes in internal/module/\n", step+1)
	return nil
}

func runDomainJobAdd(cmd *cobra.Command, args []string) error {
	if domainJobDomain == "" {
		return fmt.Errorf("--domain is required")
	}
	if domainJobTask == "" {
		return fmt.Errorf("--task is required")
	}

	config := &generator.DomainJobConfig{
		DomainKey:     domainJobDomain,
		TaskType:      domainJobTask,
		Mode:          domainJobMode,
		WorkspacePath: domainWorkspace,
		ModuleBase:    domainModule,
	}

	color.Cyan("\n🚀 Generating domain job: %s (%s)", config.DomainKey, config.TaskType)
	if err := generator.NewDomainJobGenerator(config).Generate(); err != nil {
		return err
	}
	color.Green("✅ Domain job generated successfully!")
	fmt.Printf("   Domain: %s\n", config.DomainKey)
	fmt.Printf("   Task:   %s\n", config.TaskType)
	fmt.Printf("   Mode:   %s\n", config.Mode)
	return nil
}
