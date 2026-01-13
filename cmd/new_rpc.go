package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var newRPCCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Create a new gRPC application",
	Long: `Create a new gRPC application with Yogan Framework.

This command runs in interactive mode and will guide you through
the configuration process.

Example:
  go-ygctl new rpc`,
	RunE: runNewRPC,
}

func init() {
	newCmd.AddCommand(newRPCCmd)
}

func runNewRPC(cmd *cobra.Command, args []string) error {
	// Interactive mode
	config, err := generator.PromptRPCConfig()
	if err != nil {
		return err
	}

	// Generate
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

	color.Yellow("  2. Generate proto:")
	fmt.Printf("     cd %s\n", absProjectPath)
	fmt.Println("     make proto-install  # Install protoc plugins (first time)")
	fmt.Println("     make proto          # Generate Go code from proto")
	fmt.Println()

	color.Yellow("  3. Run application:")
	fmt.Printf("     cd %s\n", absAppPath)
	fmt.Println("     go run main.go")
	fmt.Println()

	color.Yellow("  4. Test gRPC service:")
	fmt.Printf("     grpcurl -plaintext localhost:%d list\n", config.GRPCPort)
	fmt.Println()

	return nil
}
