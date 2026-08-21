package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	rpcInteractive bool
	rpcProject     string
	rpcOrg         string
	rpcModule      string
	rpcOutput      string
	rpcPort        int
	rpcService     string
	rpcLocalFW     bool
	rpcFWPath      string
)

var newRPCCmd = &cobra.Command{
	Use:   "rpc [app-name]",
	Short: "Create a new gRPC application",
	Long: `Create a multi-app project with a gRPC application.

If app-name is omitted (or --interactive is set), the command prompts
for configuration interactively.

Example:
  go-ygctl new rpc demo-rpc \
    --project demo-proj \
    --org github.com/KOMKZ \
    --output . \
    --grpc-port 9101`,
	RunE: runNewRPC,
}

func init() {
	newCmd.AddCommand(newRPCCmd)
	newRPCCmd.Flags().BoolVarP(&rpcInteractive, "interactive", "i", false, "Interactive mode")
	newRPCCmd.Flags().StringVar(&rpcProject, "project", "", "Project name (workspace root, kebab-case)")
	newRPCCmd.Flags().StringVar(&rpcOrg, "org", "github.com/KOMKZ", "Organization module prefix")
	newRPCCmd.Flags().StringVarP(&rpcModule, "module", "m", "", "App Go module name (auto-generated if empty)")
	newRPCCmd.Flags().StringVarP(&rpcOutput, "output", "o", ".", "Output directory")
	newRPCCmd.Flags().IntVar(&rpcPort, "grpc-port", 9000, "gRPC server port")
	newRPCCmd.Flags().StringVar(&rpcService, "service", "", "Proto service name (PascalCase, default derived from app name)")
	newRPCCmd.Flags().BoolVar(&rpcLocalFW, "local-framework", true, "Use local framework with replace directive")
	newRPCCmd.Flags().StringVar(&rpcFWPath, "framework-path", "../../../go-yogan-framework", "Local framework path (relative to apps/<app>)")
}

func runNewRPC(cmd *cobra.Command, args []string) error {
	var config *generator.RPCConfig
	var err error

	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	if rpcInteractive || appName == "" {
		config, err = generator.PromptRPCConfig()
		if err != nil {
			return err
		}
	} else {
		config = generator.NewDefaultRPCConfig()
		config.AppName = appName
		config.ProjectName = rpcProject
		config.OrgName = rpcOrg
		if rpcModule != "" {
			config.ModuleName = rpcModule
		} else {
			config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)
		}
		config.OutputPath = rpcOutput
		config.GRPCPort = rpcPort
		config.ServiceName = rpcService
		if config.ServiceName == "" {
			config.ServiceName = generator.ToPascalCase(config.AppName)
		}
		config.UseLocalFramework = rpcLocalFW
		config.FrameworkPath = rpcFWPath
		config.Description = fmt.Sprintf("%s gRPC API", generator.ToPascalCase(config.AppName))

		if config.ProjectName == "" {
			config.ProjectName = "my-project"
		}
		if config.OrgName == "" {
			config.OrgName = "github.com/KOMKZ"
		}
	}

	color.Cyan("\n🚀 Generating multi-app project with gRPC application: %s", config.ProjectName)

	gen := generator.NewRPCGenerator(config)
	if err := gen.Generate(); err != nil {
		return err
	}

	projectPath := filepath.Join(config.OutputPath, config.ProjectName)
	appPath := filepath.Join(projectPath, "apps", config.AppName)
	absProjectPath, _ := filepath.Abs(projectPath)
	absAppPath, _ := filepath.Abs(appPath)

	color.Green("✅ Project generated successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println()
	color.Yellow("  1. Install dependencies:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go mod tidy")
	fmt.Println()
	color.Yellow("  2. Run tests:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go test ./...")
	fmt.Println()
	color.Yellow("  3. Generate proto (optional):")
	fmt.Printf("     cd %s\n", absProjectPath)
	fmt.Println("     make proto-install  # Install protoc plugins (first time)")
	fmt.Println("     make proto          # Generate Go code from proto")
	fmt.Println()
	color.Yellow("  4. Run application:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run .")
	fmt.Println()

	return nil
}
