package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	testMode   bool
	testName   string
	testModule string
	testOutput string
)

var newHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Create a new HTTP application",
	Long: `Create a new Gin-based HTTP application with Yogan Framework.

This command runs in interactive mode and will guide you through
the configuration process.

Example:
  go-ygctl new http`,
	RunE: runNewHTTP,
}

func init() {
	newCmd.AddCommand(newHTTPCmd)
	// Hidden flags for testing
	newHTTPCmd.Flags().BoolVar(&testMode, "test", false, "Test mode (non-interactive)")
	newHTTPCmd.Flags().StringVar(&testName, "test-name", "", "App name for test mode")
	newHTTPCmd.Flags().StringVar(&testModule, "test-module", "", "Module name for test mode")
	newHTTPCmd.Flags().StringVar(&testOutput, "test-output", ".", "Output path for test mode")
	newHTTPCmd.Flags().MarkHidden("test")
	newHTTPCmd.Flags().MarkHidden("test-name")
	newHTTPCmd.Flags().MarkHidden("test-module")
	newHTTPCmd.Flags().MarkHidden("test-output")
}

func runNewHTTP(cmd *cobra.Command, args []string) error {
	var config *generator.AppConfig
	var err error

	if testMode {
		// Test mode (non-interactive)
		config = generator.NewDefaultConfig()
		config.AppName = testName
		config.ModuleName = testModule
		config.OutputPath = testOutput
		config.Description = fmt.Sprintf("%s HTTP API", generator.ToPascalCase(testName))
	} else {
		// Interactive mode
		config, err = generator.PromptHTTPConfig()
		if err != nil {
			return err
		}
	}

	// Generate
	color.Cyan("\n🚀 Generating HTTP application: %s", config.AppName)

	gen := generator.NewHTTPGenerator(config)
	if err := gen.Generate(); err != nil {
		return err
	}

	appPath := filepath.Join(config.OutputPath, config.AppName)
	absPath, _ := filepath.Abs(appPath)

	color.Green("✅ Application generated successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", absPath)

	if config.UseLocalFramework {
		fmt.Println("  go mod tidy")
	} else {
		fmt.Println("  go get github.com/KOMKZ/go-yogan-framework@latest")
		fmt.Println("  go mod tidy")
	}

	fmt.Println("  go run main.go")
	fmt.Println()

	return nil
}
