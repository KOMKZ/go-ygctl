package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cliInteractive bool
	cliProject     string
	cliOrg         string
	cliModule      string
	cliOutput      string
	cliLocalFW     bool
	cliFWPath      string
)

var newCLICmd = &cobra.Command{
	Use:   "cli [app-name]",
	Short: "Create a new CLI application",
	Long: `Create a multi-app project with a CLI application.

If app-name is omitted (or --interactive is set), the command prompts
for configuration interactively.

Example:
  go-ygctl new cli demo-cli --project demo-proj --org github.com/KOMKZ --output .`,
	RunE: runNewCLI,
}

func init() {
	newCmd.AddCommand(newCLICmd)
	newCLICmd.Flags().BoolVarP(&cliInteractive, "interactive", "i", false, "Interactive mode")
	newCLICmd.Flags().StringVar(&cliProject, "project", "", "Project name (workspace root, kebab-case)")
	newCLICmd.Flags().StringVar(&cliOrg, "org", "github.com/KOMKZ", "Organization module prefix")
	newCLICmd.Flags().StringVarP(&cliModule, "module", "m", "", "App Go module name (auto-generated if empty)")
	newCLICmd.Flags().StringVarP(&cliOutput, "output", "o", ".", "Output directory")
	newCLICmd.Flags().BoolVar(&cliLocalFW, "local-framework", true, "Use local framework with replace directive")
	newCLICmd.Flags().StringVar(&cliFWPath, "framework-path", "../../../go-yogan-framework", "Local framework path (relative to apps/<app>)")
}

func runNewCLI(cmd *cobra.Command, args []string) error {
	var config *generator.CLIConfig
	var err error

	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	if cliInteractive || appName == "" {
		config, err = generator.PromptCLIConfig()
		if err != nil {
			return err
		}
	} else {
		config = generator.NewDefaultCLIConfig()
		config.AppName = appName
		config.ProjectName = cliProject
		config.OrgName = cliOrg
		if cliModule != "" {
			config.ModuleName = cliModule
		} else {
			config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)
		}
		config.OutputPath = cliOutput
		config.UseLocalFramework = cliLocalFW
		config.FrameworkPath = cliFWPath
		config.Description = fmt.Sprintf("%s CLI", generator.ToPascalCase(config.AppName))

		if config.ProjectName == "" {
			config.ProjectName = "my-project"
		}
		if config.OrgName == "" {
			config.OrgName = "github.com/KOMKZ"
		}
	}

	color.Cyan("\n🚀 Generating multi-app project with CLI application: %s", config.ProjectName)

	gen := generator.NewCLIGenerator(config)
	if err := gen.Generate(); err != nil {
		return err
	}

	projectPath := filepath.Join(config.OutputPath, config.ProjectName)
	appPath := filepath.Join(projectPath, "apps", config.AppName)
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
	color.Yellow("  3. Run CLI application:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run . --help")
	fmt.Println("     go run . home        # Run home command")
	fmt.Println()

	return nil
}
