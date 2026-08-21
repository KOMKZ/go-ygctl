package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cronInteractive bool
	cronProject     string
	cronOrg         string
	cronModule      string
	cronOutput      string
	cronLocalFW     bool
	cronFWPath      string
)

var newCronCmd = &cobra.Command{
	Use:   "cron [app-name]",
	Short: "Create a new Cron application",
	Long: `Create a multi-app project with a Cron (scheduled tasks) application.

If app-name is omitted (or --interactive is set), the command prompts
for configuration interactively.

Example:
  go-ygctl new cron demo-cron --project demo-proj --org github.com/KOMKZ --output .`,
	RunE: runNewCron,
}

func init() {
	newCmd.AddCommand(newCronCmd)
	newCronCmd.Flags().BoolVarP(&cronInteractive, "interactive", "i", false, "Interactive mode")
	newCronCmd.Flags().StringVar(&cronProject, "project", "", "Project name (workspace root, kebab-case)")
	newCronCmd.Flags().StringVar(&cronOrg, "org", "github.com/KOMKZ", "Organization module prefix")
	newCronCmd.Flags().StringVarP(&cronModule, "module", "m", "", "App Go module name (auto-generated if empty)")
	newCronCmd.Flags().StringVarP(&cronOutput, "output", "o", ".", "Output directory")
	newCronCmd.Flags().BoolVar(&cronLocalFW, "local-framework", true, "Use local framework with replace directive")
	newCronCmd.Flags().StringVar(&cronFWPath, "framework-path", "../../../go-yogan-framework", "Local framework path (relative to apps/<app>)")
}

func runNewCron(cmd *cobra.Command, args []string) error {
	var config *generator.CronConfig
	var err error

	appName := ""
	if len(args) > 0 {
		appName = args[0]
	}

	if cronInteractive || appName == "" {
		config, err = generator.PromptCronConfig()
		if err != nil {
			return err
		}
	} else {
		config = generator.NewDefaultCronConfig()
		config.AppName = appName
		config.ProjectName = cronProject
		config.OrgName = cronOrg
		if cronModule != "" {
			config.ModuleName = cronModule
		} else {
			config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)
		}
		config.OutputPath = cronOutput
		config.UseLocalFramework = cronLocalFW
		config.FrameworkPath = cronFWPath
		config.Description = fmt.Sprintf("%s scheduled tasks", generator.ToPascalCase(config.AppName))

		if config.ProjectName == "" {
			config.ProjectName = "my-project"
		}
		if config.OrgName == "" {
			config.OrgName = "github.com/KOMKZ"
		}
	}

	color.Cyan("\n🚀 Generating multi-app project with Cron application: %s", config.ProjectName)

	gen := generator.NewCronGenerator(config)
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
	color.Yellow("  3. Start cron scheduler:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run .")
	fmt.Println()

	return nil
}
