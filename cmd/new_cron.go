package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCronCmd = &cobra.Command{
	Use:   "cron",
	Short: "Create a new Cron application",
	Long: `Create a new Cron (scheduled tasks) application with Yogan Framework.

This command runs in interactive mode and will guide you through
the configuration process.

Example:
  go-ygctl new cron`,
	RunE: runNewCron,
}

func init() {
	newCmd.AddCommand(newCronCmd)
}

func runNewCron(cmd *cobra.Command, args []string) error {
	// Interactive mode
	config, err := generator.PromptCronConfig()
	if err != nil {
		return err
	}

	// Generate
	color.Cyan("\n🚀 Generating multi-app project with Cron application: %s", config.ProjectName)

	gen := generator.NewCronGenerator(config)
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
	fmt.Printf("     cd %s/pkg\n", absProjectPath)
	if !config.UseLocalFramework {
		fmt.Println("     go get github.com/KOMKZ/go-yogan-framework@latest")
	}
	fmt.Println("     go mod tidy")
	fmt.Println()
	fmt.Printf("     cd %s\n", absAppPath)
	if !config.UseLocalFramework {
		fmt.Println("     go get github.com/KOMKZ/go-yogan-framework@latest")
	}
	fmt.Println("     go mod tidy")
	fmt.Println()

	color.Yellow("  2. Start cron scheduler:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run main.go start")
	fmt.Println()

	color.Yellow("  3. Run single task manually:")
	fmt.Println("     go run main.go run --task demo")
	fmt.Println()

	return nil
}
