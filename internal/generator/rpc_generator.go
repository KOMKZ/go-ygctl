package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// RPCGenerator generates gRPC application from templates
type RPCGenerator struct {
	config *RPCConfig
}

// NewRPCGenerator creates a new RPC generator
func NewRPCGenerator(config *RPCConfig) *RPCGenerator {
	return &RPCGenerator{config: config}
}

// Generate creates the multi-app project with gRPC application
func (g *RPCGenerator) Generate() error {
	if err := g.config.Validate(); err != nil {
		return err
	}

	projectPath := filepath.Join(g.config.OutputPath, g.config.ProjectName)
	appPath := filepath.Join(projectPath, "apps", g.config.AppName)
	serviceLower := ToSnakeCase(g.config.ServiceName)
	protoPath := filepath.Join(projectPath, "proto", serviceLower)

	// Check if project path exists
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrPathExists, projectPath)
	}

	// Create project-level directory structure
	projectDirs := []string{
		"",
		"apps",
		"domains",
		"proto",
		"pkg/errdef",
		"pkg/apputil",
		"scripts",
	}

	for _, dir := range projectDirs {
		if err := os.MkdirAll(filepath.Join(projectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create project directory %s: %w", dir, err)
		}
	}

	// Create app-level directory structure
	appDirs := []string{
		"",
		"config",
		"internal/app",
	}

	for _, dir := range appDirs {
		if err := os.MkdirAll(filepath.Join(appPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create app directory %s: %w", dir, err)
		}
	}

	// Create proto directory
	if err := os.MkdirAll(protoPath, 0755); err != nil {
		return fmt.Errorf("failed to create proto directory: %w", err)
	}

	data := g.templateData()

	// Generate project-level files
	projectFiles := []struct {
		template   string
		output     string
		executable bool
	}{
		{"project/.gitignore.tmpl", ".gitignore", false},
		{"project/README.md.tmpl", "README.md", false},
		{"project/Makefile.tmpl", "Makefile", false},
		{"rpc/project/go.work.tmpl", "go.work", false},
		{"project/domains/.gitkeep.tmpl", "domains/.gitkeep", false},
		{"project/pkg/go.mod.tmpl", "pkg/go.mod", false},
		{"project/pkg/errdef/errors.go.tmpl", "pkg/errdef/errors.go", false},
		{"project/pkg/apputil/registry.go.tmpl", "pkg/apputil/registry.go", false},
		{"project/scripts/build.sh.tmpl", "scripts/build.sh", true},
	}

	for _, f := range projectFiles {
		if err := g.renderTemplate(projectPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate project file %s: %w", f.output, err)
		}
		if f.executable {
			outputPath := filepath.Join(projectPath, f.output)
			if err := os.Chmod(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to set executable permission for %s: %w", f.output, err)
			}
		}
	}

	// Generate app-level files
	appFiles := []struct {
		template string
		output   string
	}{
		{"rpc/main.go.tmpl", "main.go"},
		{"rpc/go.mod.tmpl", "go.mod"},
		{"rpc/Makefile.tmpl", "Makefile"},
		{"rpc/config/config.yaml.tmpl", "config/config.yaml"},
		{"rpc/config/test.yaml.tmpl", "config/test.yaml"},
		{"rpc/internal/app/app.go.tmpl", "internal/app/app.go"},
		{"rpc/internal/app/callbacks.go.tmpl", "internal/app/callbacks.go"},
		{"rpc/internal/app/app_test.go.tmpl", "internal/app/app_test.go"},
	}

	for _, f := range appFiles {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate app file %s: %w", f.output, err)
		}
	}

	// Generate proto files
	protoFiles := []struct {
		template string
		output   string
	}{
		{"rpc/proto/service.proto.tmpl", serviceLower + ".proto"},
		{"rpc/proto/go.mod.tmpl", "go.mod"},
		{"rpc/proto/Makefile.tmpl", "Makefile"},
		{"rpc/proto/README.md.tmpl", "README.md"},
	}

	for _, f := range protoFiles {
		if err := g.renderTemplate(protoPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate proto file %s: %w", f.output, err)
		}
	}

	return nil
}

// templateData returns the data for template rendering
func (g *RPCGenerator) templateData() map[string]interface{} {
	appNameSnake := strings.ReplaceAll(g.config.AppName, "-", "_")
	appNameUpper := strings.ToUpper(appNameSnake)
	projectNameSnake := strings.ReplaceAll(g.config.ProjectName, "-", "_")
	serviceLower := ToSnakeCase(g.config.ServiceName)

	// Project module: github.com/myorg/my-project
	projectModule := fmt.Sprintf("%s/%s", g.config.OrgName, g.config.ProjectName)

	// Framework path for pkg: in workspace mode, relative replace directives in
	// non-main modules resolve against the MAIN module (the app) directory, so
	// pkg must use the SAME path value as the app to reach the same framework
	// directory (go.work rejects conflicting replacements otherwise).
	pkgFrameworkPath := g.config.FrameworkPath

	return map[string]interface{}{
		// Project level
		"ProjectName":       g.config.ProjectName,
		"ProjectNameSnake":  projectNameSnake,
		"ProjectModule":     projectModule,
		"OrgName":           g.config.OrgName,
		"PkgFrameworkPath":  pkgFrameworkPath,
		"GenerateProto":     true, // RPC always has proto

		// App level
		"AppName":           g.config.AppName,
		"AppNamePascal":     ToPascalCase(g.config.AppName),
		"AppNameSnake":      appNameSnake,
		"AppNameUpper":      appNameUpper,
		"ModuleName":        g.config.ModuleName,
		"Description":       g.config.Description,
		"GRPCPort":          g.config.GRPCPort,
		"UseLocalFramework": g.config.UseLocalFramework,
		"FrameworkPath":     g.config.FrameworkPath,

		// Service level
		"ServiceName":       g.config.ServiceName,
		"ServiceNameLower":  serviceLower,
		"ServiceNameSnake":  serviceLower,
	}
}

// renderTemplate renders a template file
func (g *RPCGenerator) renderTemplate(outputDir, tmplName, outputName string, data map[string]interface{}) error {
	tmplPath := filepath.Join("templates", tmplName)
	content, err := httpTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	outputPath := filepath.Join(outputDir, outputName)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", tmplName, err)
	}

	return nil
}
