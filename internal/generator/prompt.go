package generator

import (
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

// PromptHTTPConfig interactively prompts user for HTTP app configuration
func PromptHTTPConfig() (*AppConfig, error) {
	config := NewDefaultConfig()

	fmt.Println("\n🚀 Create Multi-App Project with HTTP Application")
	fmt.Println("──────────────────────────────────────────────────")

	// Project name
	if err := survey.AskOne(&survey.Input{
		Message: "Project name (workspace root):",
		Help:    "Use kebab-case, e.g., my-project",
	}, &config.ProjectName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Organization name
	if err := survey.AskOne(&survey.Input{
		Message: "Organization (module prefix):",
		Default: "github.com/myorg",
		Help:    "e.g., github.com/myorg",
	}, &config.OrgName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// App name
	if err := survey.AskOne(&survey.Input{
		Message: "First application name:",
		Default: "admin-api",
		Help:    "Use kebab-case, e.g., admin-api, user-api",
	}, &config.AppName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Auto-generate module name for app
	config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)

	// Description
	if err := survey.AskOne(&survey.Input{
		Message: "Application description:",
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
			Message: "Local framework path (relative to apps/<app>):",
			Default: "../../../go-yogan-framework",
		}, &config.FrameworkPath); err != nil {
			return nil, err
		}
	}

	// Optional: Generate proto example
	var generateProto bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Generate gRPC/Proto example (DemoPaymentService)?",
		Default: false,
		Help:    "Creates proto/ directory with example .proto file and build scripts",
	}, &generateProto); err != nil {
		return nil, err
	}
	config.GenerateProto = generateProto

	// Summary
	projectPath := filepath.Join(config.OutputPath, config.ProjectName)
	fmt.Println("\n═══════════════════════════════════════════════════")
	fmt.Println("📋 Configuration Summary")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  Project:     %s\n", config.ProjectName)
	fmt.Printf("  Output:      %s\n", projectPath)
	fmt.Printf("  App:         apps/%s\n", config.AppName)
	fmt.Printf("  Module:      %s\n", config.ModuleName)
	fmt.Printf("  Port:        %d\n", config.ServerPort)
	if config.UseLocalFramework {
		fmt.Printf("  Framework:   local (%s)\n", config.FrameworkPath)
	} else {
		fmt.Printf("  Framework:   remote\n")
	}
	if config.GenerateProto {
		fmt.Println("  Proto:       yes (DemoPaymentService example)")
	} else {
		fmt.Println("  Proto:       no")
	}
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println("  Generated structure:")
	fmt.Printf("  %s/\n", config.ProjectName)
	fmt.Println("  ├── go.work")
	fmt.Printf("  ├── apps/%s/\n", config.AppName)
	fmt.Println("  ├── domains/")
	if config.GenerateProto {
		fmt.Println("  ├── proto/payment/  (with DemoPaymentService)")
	} else {
		fmt.Println("  ├── proto/")
	}
	fmt.Println("  ├── pkg/")
	fmt.Println("  └── scripts/")
	fmt.Println("═══════════════════════════════════════════════════")

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

// PromptRPCConfig interactively prompts user for gRPC app configuration
func PromptRPCConfig() (*RPCConfig, error) {
	config := NewDefaultRPCConfig()

	fmt.Println("\n🚀 Create Multi-App Project with gRPC Application")
	fmt.Println("──────────────────────────────────────────────────")

	// Project name
	if err := survey.AskOne(&survey.Input{
		Message: "Project name (workspace root):",
		Help:    "Use kebab-case, e.g., my-project",
	}, &config.ProjectName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Organization name
	if err := survey.AskOne(&survey.Input{
		Message: "Organization (module prefix):",
		Default: "github.com/myorg",
		Help:    "e.g., github.com/myorg",
	}, &config.OrgName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// App name
	if err := survey.AskOne(&survey.Input{
		Message: "Application name:",
		Default: "payment-rpc",
		Help:    "Use kebab-case, e.g., payment-rpc, order-rpc",
	}, &config.AppName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Service name (PascalCase)
	defaultServiceName := ToPascalCase(config.AppName)
	// Remove "Rpc" suffix if present
	if len(defaultServiceName) > 3 && defaultServiceName[len(defaultServiceName)-3:] == "Rpc" {
		defaultServiceName = defaultServiceName[:len(defaultServiceName)-3]
	}
	if err := survey.AskOne(&survey.Input{
		Message: "Proto service name (PascalCase):",
		Default: defaultServiceName,
		Help:    "e.g., Payment, Order, User",
	}, &config.ServiceName, survey.WithValidator(survey.Required)); err != nil {
		return nil, err
	}

	// Auto-generate module name for app
	config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", config.OrgName, config.ProjectName, config.AppName)

	// Description
	if err := survey.AskOne(&survey.Input{
		Message: "Application description:",
		Default: fmt.Sprintf("%s gRPC Service", config.ServiceName),
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

	// gRPC port
	if err := survey.AskOne(&survey.Input{
		Message: "gRPC server port:",
		Default: "9000",
	}, &config.GRPCPort); err != nil {
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
			Message: "Local framework path (relative to apps/<app>):",
			Default: "../../../go-yogan-framework",
		}, &config.FrameworkPath); err != nil {
			return nil, err
		}
	}

	// Summary
	projectPath := filepath.Join(config.OutputPath, config.ProjectName)
	serviceLower := ToSnakeCase(config.ServiceName)
	fmt.Println("\n═══════════════════════════════════════════════════")
	fmt.Println("📋 Configuration Summary")
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("  Project:     %s\n", config.ProjectName)
	fmt.Printf("  Output:      %s\n", projectPath)
	fmt.Printf("  App:         apps/%s\n", config.AppName)
	fmt.Printf("  Service:     %sService\n", config.ServiceName)
	fmt.Printf("  Module:      %s\n", config.ModuleName)
	fmt.Printf("  gRPC Port:   %d\n", config.GRPCPort)
	if config.UseLocalFramework {
		fmt.Printf("  Framework:   local (%s)\n", config.FrameworkPath)
	} else {
		fmt.Printf("  Framework:   remote\n")
	}
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println("  Generated structure:")
	fmt.Printf("  %s/\n", config.ProjectName)
	fmt.Println("  ├── go.work")
	fmt.Printf("  ├── apps/%s/\n", config.AppName)
	fmt.Println("  ├── domains/")
	fmt.Printf("  ├── proto/%s/  (%sService)\n", serviceLower, config.ServiceName)
	fmt.Println("  ├── pkg/")
	fmt.Println("  └── scripts/")
	fmt.Println("═══════════════════════════════════════════════════")

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
