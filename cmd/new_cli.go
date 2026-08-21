package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newCLICmd = &cobra.Command{
	Use:   "cli",
	Short: "Create a new CLI application",
	Long: `Create a new CLI application with Yogan Framework.

This command runs in interactive mode and will guide you through
the configuration process.

Example:
  go-ygctl new cli`,
	RunE: runNewCLI,
}

func init() {
	newCmd.AddCommand(newCLICmd)
}

func runNewCLI(cmd *cobra.Command, args []string) error {
	// Interactive mode
	config, err := generator.PromptCLIConfig()
	if err != nil {
		return err
	}

	// Generate
	color.Cyan("\n🚀 Generating multi-app project with CLI application: %s", config.ProjectName)

	gen := generator.NewCLIGenerator(config)
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

	color.Yellow("  2. Run CLI application:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run main.go --help")
	fmt.Println("     go run main.go home        # Run home command")
	fmt.Println()

	return nil
}
