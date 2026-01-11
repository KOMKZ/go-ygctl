package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// PromptHTTPConfig interactively prompts user for HTTP app configuration
func PromptHTTPConfig() (*AppConfig, error) {
	config := NewDefaultConfig()

	// App name
	if err := survey.AskOne(&survey.Input{
		Message: "Application name (e.g., user-api):",
		Help:    "Use kebab-case, this will be the directory name",
	}, &config.AppName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Module name with default suggestion
	defaultModule := fmt.Sprintf("github.com/myorg/%s", config.AppName)
	if err := survey.AskOne(&survey.Input{
		Message: "Go module name:",
		Default: defaultModule,
		Help:    "Full Go module path for go.mod",
	}, &config.ModuleName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Description
	if err := survey.AskOne(&survey.Input{
		Message: "Description (optional):",
		Default: fmt.Sprintf("%s HTTP API", ToPascalCase(config.AppName)),
	}, &config.Description); err != nil {
		return nil, err
	}

	// Output path
	if err := survey.AskOne(&survey.Input{
		Message: "Output directory:",
		Default: ".",
		Help:    "Directory where the app folder will be created",
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
		Help:    "If yes, uses local path; if no, uses remote Go module",
	}, &useLocal); err != nil {
		return nil, err
	}
	config.UseLocalFramework = useLocal

	if useLocal {
		if err := survey.AskOne(&survey.Input{
			Message: "Local framework path:",
			Default: "../../go-yogan-framework",
			Help:    "Relative path from app directory to framework",
		}, &config.FrameworkPath); err != nil {
			return nil, err
		}
	}

	// Optional components
	var components []string
	if err := survey.AskOne(&survey.MultiSelect{
		Message: "Enable components (optional):",
		Options: []string{"database", "redis"},
		Help:    "Select components to enable by default",
	}, &components); err != nil {
		return nil, err
	}

	for _, c := range components {
		switch c {
		case "database":
			config.EnableDatabase = true
		case "redis":
			config.EnableRedis = true
		}
	}

	// Confirmation
	fmt.Println()
	fmt.Println("=== Configuration Summary ===")
	fmt.Printf("  App Name:    %s\n", config.AppName)
	fmt.Printf("  Module:      %s\n", config.ModuleName)
	fmt.Printf("  Output:      %s\n", filepath.Join(config.OutputPath, config.AppName))
	fmt.Printf("  Port:        %d\n", config.ServerPort)
	fmt.Printf("  Framework:   %s\n", frameworkDesc(config))
	fmt.Printf("  Components:  %s\n", componentsDesc(config))
	fmt.Println()

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

func frameworkDesc(c *AppConfig) string {
	if c.UseLocalFramework {
		return fmt.Sprintf("local (%s)", c.FrameworkPath)
	}
	return "remote (go module)"
}

func componentsDesc(c *AppConfig) string {
	var parts []string
	parts = append(parts, "health")
	if c.EnableDatabase {
		parts = append(parts, "database")
	}
	if c.EnableRedis {
		parts = append(parts, "redis")
	}
	return strings.Join(parts, ", ")
}
