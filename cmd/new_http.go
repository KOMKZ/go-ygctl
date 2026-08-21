package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	httpInteractive bool
	httpProject     string
	httpOrg         string
	httpModule      string
	httpOutput      string
	httpPort        int
	httpLocalFW     bool
	httpFWPath      string
)

var newHTTPCmd = &cobra.Command{
	Use:   "http [app-name]",
	Short: "Create a new HTTP application",
	Long: `Create a multi-app project with a Gin-based HTTP application.

If app-name is omitted (or --interactive is set), the command prompts
for configuration interactively.

Example:
  go-ygctl new http hrise-admin-api \
    --project hrise-server-app \
    --org github.com/KOMKZ \
    --output . \
    --port 9201`,
	RunE: runNewHTTP,
}

func init() {
	newCmd.AddCommand(newHTTPCmd)
	newHTTPCmd.Flags().BoolVarP(&httpInteractive, "interactive", "i", false, "Interactive mode")
	newHTTPCmd.Flags().StringVar(&httpProject, "project", "", "Project name (workspace root, kebab-case)")
	newHTTPCmd.Flags().StringVar(&httpOrg, "org", "github.com/KOMKZ", "Organization module prefix")
	newHTTPCmd.Flags().StringVarP(&httpModule, "module", "m", "", "App Go module name (auto-generated if empty)")
	newHTTPCmd.Flags().StringVarP(&httpOutput, "output", "o", ".", "Output directory")
	newHTTPCmd.Flags().IntVar(&httpPort, "port", 8080, "Server port")
	newHTTPCmd.Flags().BoolVar(&httpLocalFW, "local-framework", true, "Use local framework with replace directive")
	newHTTPCmd.Flags().StringVar(&httpFWPath, "framework-path", "../../../go-yogan-framework", "Local framework path (relative to apps/<app>)")
}

func runNewHTTP(cmd *cobra.Command, args []string) error {
	var config *generator.AppConfig
	var err error

	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	if httpInteractive || appName == "" {
		// Interactive mode
		config, err = generator.PromptHTTPConfig()
		if err != nil {
			return err
		}
	} else {
		// Non-interactive mode: app name from args, rest from flags
		config = generator.NewDefaultConfig()
		config.AppName = appName
		config.ProjectName = httpProject
		config.OrgName = httpOrg
		if httpModule != "" {
			config.ModuleName = httpModule
		} else {
			config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)
		}
		config.OutputPath = httpOutput
		config.ServerPort = httpPort
		config.UseLocalFramework = httpLocalFW
		config.FrameworkPath = httpFWPath
		config.Description = fmt.Sprintf("%s HTTP API", generator.ToPascalCase(config.AppName))

		// Backfill defaults for convenience
		if config.ProjectName == "" {
			config.ProjectName = "my-project"
		}
		if config.OrgName == "" {
			config.OrgName = "github.com/KOMKZ"
		}
	}

	// Generate
	color.Cyan("\n🚀 Generating multi-app project: %s", config.ProjectName)

	gen := generator.NewHTTPGenerator(config)
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
	color.Yellow("  3. Run application:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run .")
	fmt.Println()

	if config.GenerateProto {
		color.Yellow("  4. Generate proto (optional):")
		fmt.Printf("     cd %s\n", absProjectPath)
		fmt.Println("     make proto-install  # Install protoc plugins")
		fmt.Println("     make proto          # Generate Go code")
		fmt.Println()
	}

	return nil
}
