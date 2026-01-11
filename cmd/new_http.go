package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	httpAppName     string
	httpModuleName  string
	httpOutputPath  string
	httpPort        int
	httpInteractive bool
	httpLocalFw     bool
	httpFwPath      string
)

var newHTTPCmd = &cobra.Command{
	Use:   "http [app-name]",
	Short: "Create a new HTTP application",
	Long: `Create a new Gin-based HTTP application with Yogan Framework.

Examples:
  # Interactive mode
  go-ygctl new http --interactive

  # Quick mode with arguments
  go-ygctl new http my-api --module github.com/myorg/my-api

  # Specify output path
  go-ygctl new http my-api --output ./apps --port 8090`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNewHTTP,
}

func init() {
	newCmd.AddCommand(newHTTPCmd)

	newHTTPCmd.Flags().BoolVarP(&httpInteractive, "interactive", "i", false, "Interactive mode")
	newHTTPCmd.Flags().StringVarP(&httpModuleName, "module", "m", "", "Go module name")
	newHTTPCmd.Flags().StringVarP(&httpOutputPath, "output", "o", ".", "Output directory")
	newHTTPCmd.Flags().IntVarP(&httpPort, "port", "p", 8080, "Server port")
	newHTTPCmd.Flags().BoolVar(&httpLocalFw, "local-framework", true, "Use local framework with replace directive")
	newHTTPCmd.Flags().StringVar(&httpFwPath, "framework-path", "../../go-yogan-framework", "Local framework path")
}

func runNewHTTP(cmd *cobra.Command, args []string) error {
	var config *generator.AppConfig
	var err error

	if httpInteractive {
		// Interactive mode
		config, err = generator.PromptHTTPConfig()
		if err != nil {
			return err
		}
	} else {
		// Quick mode
		if len(args) < 1 {
			return fmt.Errorf("app name required, use --interactive for guided setup")
		}

		config = generator.NewDefaultConfig()
		config.AppName = args[0]
		config.OutputPath = httpOutputPath
		config.ServerPort = httpPort
		config.UseLocalFramework = httpLocalFw
		config.FrameworkPath = httpFwPath

		if httpModuleName != "" {
			config.ModuleName = httpModuleName
		} else {
			config.ModuleName = fmt.Sprintf("github.com/myorg/%s", config.AppName)
		}

		config.Description = fmt.Sprintf("%s HTTP API", generator.ToPascalCase(config.AppName))
	}

	// Generate
	color.Cyan("🚀 Generating HTTP application: %s", config.AppName)

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
		// Local mode: just tidy and run
		fmt.Println("  go mod tidy")
	} else {
		// Remote mode: need to fetch framework first
		fmt.Println("  go get github.com/KOMKZ/go-yogan-framework@latest")
		fmt.Println("  go mod tidy")
	}

	fmt.Println("  go run main.go")
	fmt.Println()

	return nil
}
