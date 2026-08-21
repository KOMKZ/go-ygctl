package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// CLIGenerator generates CLI application from templates
type CLIGenerator struct {
	config *CLIConfig
}

// NewCLIGenerator creates a new CLI generator
func NewCLIGenerator(config *CLIConfig) *CLIGenerator {
	return &CLIGenerator{config: config}
}

// Generate creates the multi-app project with CLI application
func (g *CLIGenerator) Generate() error {
	if err := g.config.Validate(); err != nil {
		return err
	}

	projectPath := filepath.Join(g.config.OutputPath, g.config.ProjectName)
	appPath := filepath.Join(projectPath, "apps", g.config.AppName)

	// Check if project path exists
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrPathExists, projectPath)
	}

	// Create project-level directory structure
	projectDirs := []string{
		"",
		"apps",
		"domains",
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
		"internal/command",
	}

	for _, dir := range appDirs {
		if err := os.MkdirAll(filepath.Join(appPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create app directory %s: %w", dir, err)
		}
	}

	data := g.templateData()

	// Generate project-level files
	projectFiles := []struct {
		template   string
		output     string
		executable bool
	}{
		{"project/.gitignore.tmpl", ".gitignore", false},
		{"cli/project/README.md.tmpl", "README.md", false},
		{"cli/project/Makefile.tmpl", "Makefile", false},
		{"cli/project/go.work.tmpl", "go.work", false},
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
		{"cli/main.go.tmpl", "main.go"},
		{"cli/go.mod.tmpl", "go.mod"},
		{"cli/Makefile.tmpl", "Makefile"},
		{"cli/config/config.yaml.tmpl", "config/config.yaml"},
		{"cli/config/test.yaml.tmpl", "config/test.yaml"},
		{"cli/internal/app/app.go.tmpl", "internal/app/app.go"},
		{"cli/internal/app/app_test.go.tmpl", "internal/app/app_test.go"},
		{"cli/internal/command/home.go.tmpl", "internal/command/home.go"},
	}

	for _, f := range appFiles {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate app file %s: %w", f.output, err)
		}
	}

	return nil
}

// templateData returns the data for template rendering
func (g *CLIGenerator) templateData() map[string]interface{} {
	appNameSnake := strings.ReplaceAll(g.config.AppName, "-", "_")
	appNameUpper := strings.ToUpper(appNameSnake)
	projectNameSnake := strings.ReplaceAll(g.config.ProjectName, "-", "_")

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
		"GenerateProto":     false, // CLI doesn't need proto

		// App level
		"AppName":           g.config.AppName,
		"AppNamePascal":     ToPascalCase(g.config.AppName),
		"AppNameSnake":      appNameSnake,
		"AppNameUpper":      appNameUpper,
		"ModuleName":        g.config.ModuleName,
		"Description":       g.config.Description,
		"UseLocalFramework": g.config.UseLocalFramework,
		"FrameworkPath":     g.config.FrameworkPath,
	}
}

// renderTemplate renders a template file
func (g *CLIGenerator) renderTemplate(outputDir, tmplName, outputName string, data map[string]interface{}) error {
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
