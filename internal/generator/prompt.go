package generator

import (
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

// PromptHTTPConfig interactively prompts user for HTTP app configuration
func PromptHTTPConfig() (*AppConfig, error) {
	config := NewDefaultConfig()

	fmt.Println("\n🚀 Create New HTTP Application")
	fmt.Println("───────────────────────────────")

	// App name
	if err := survey.AskOne(&survey.Input{
		Message: "Application name (e.g., user-api):",
		Help:    "Use kebab-case, this will be the directory name",
	}, &config.AppName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Module name
	defaultModule := fmt.Sprintf("github.com/myorg/%s", config.AppName)
	if err := survey.AskOne(&survey.Input{
		Message: "Go module name:",
		Default: defaultModule,
	}, &config.ModuleName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Description
	if err := survey.AskOne(&survey.Input{
		Message: "Description:",
		Default: fmt.Sprintf("%s HTTP API", ToPascalCase(config.AppName)),
	}, &config.Description); err != nil {
		return nil, err
	}

	// Output path
	if err := survey.AskOne(&survey.Input{
		Message: "Output directory:",
		Default: ".",
	}, &config.OutputPath); err != nil {
		return nil, err
	}

	// Server port
	if err := survey.AskOne(&survey.Input{
		Message: "Server port:",
		Default: "8080",
	}, &config.ServerPort); err != nil {
		return nil, err
	}

	// Framework reference
	var useLocal bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Use local framework (with replace directive)?",
		Default: true,
	}, &useLocal); err != nil {
		return nil, err
	}
	config.UseLocalFramework = useLocal

	if useLocal {
		if err := survey.AskOne(&survey.Input{
			Message: "Local framework path:",
			Default: "../../go-yogan-framework",
		}, &config.FrameworkPath); err != nil {
			return nil, err
		}
	}

	// Summary
	fmt.Println("\n═══════════════════════════════")
	fmt.Println("📋 Configuration Summary")
	fmt.Println("═══════════════════════════════")
	fmt.Printf("  App Name:    %s\n", config.AppName)
	fmt.Printf("  Module:      %s\n", config.ModuleName)
	fmt.Printf("  Output:      %s\n", filepath.Join(config.OutputPath, config.AppName))
	fmt.Printf("  Port:        %d\n", config.ServerPort)
	if config.UseLocalFramework {
		fmt.Printf("  Framework:   local (%s)\n", config.FrameworkPath)
	} else {
		fmt.Printf("  Framework:   remote\n")
	}
	fmt.Println("═══════════════════════════════")

	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Proceed with generation?",
		Default: true,
	}, &confirm); err != nil {
		return nil, err
	}

	if !confirm {
		return nil, fmt.Errorf("generation cancelled")
	}

	return config, nil
}
