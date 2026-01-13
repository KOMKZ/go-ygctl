package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates/http all:templates/project
var httpTemplates embed.FS

// HTTPGenerator generates HTTP application from templates
type HTTPGenerator struct {
	config *AppConfig
}

// NewHTTPGenerator creates a new HTTP generator
func NewHTTPGenerator(config *AppConfig) *HTTPGenerator {
	return &HTTPGenerator{config: config}
}

// Generate creates the multi-app project with HTTP application
func (g *HTTPGenerator) Generate() error {
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
		"migrations",
		"internal/app",
		"internal/config",
		"internal/domain/home/model",
		"internal/module/home",
		"internal/router",
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
		{"project/README.md.tmpl", "README.md", false},
		{"project/go.work.tmpl", "go.work", false},
		{"project/domains/.gitkeep.tmpl", "domains/.gitkeep", false},
		{"project/proto/.gitkeep.tmpl", "proto/.gitkeep", false},
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
		template   string
		output     string
		executable bool
	}{
		{"http/main.go.tmpl", "main.go", false},
		{"http/go.mod.tmpl", "go.mod", false},
		{"http/Makefile.tmpl", "Makefile", false},
		{"http/config/config.yaml.tmpl", "config/config.yaml", false},
		{"http/migrations/README.md.tmpl", "migrations/README.md", false},
		{"http/internal/app/app.go.tmpl", "internal/app/app.go", false},
		{"http/internal/app/callbacks.go.tmpl", "internal/app/callbacks.go", false},
		{"http/internal/app/components.go.tmpl", "internal/app/components.go", false},
		{"http/internal/app/router.go.tmpl", "internal/app/router.go", false},
		{"http/internal/config/config.go.tmpl", "internal/config/config.go", false},
		{"http/internal/domain/home/model/home.go.tmpl", "internal/domain/home/model/home.go", false},
		{"http/internal/domain/home/repository.go.tmpl", "internal/domain/home/repository.go", false},
		{"http/internal/domain/home/service.go.tmpl", "internal/domain/home/service.go", false},
		{"http/internal/module/home/handler.go.tmpl", "internal/module/home/handler.go", false},
		{"http/internal/module/home/request.go.tmpl", "internal/module/home/request.go", false},
		{"http/internal/module/home/response.go.tmpl", "internal/module/home/response.go", false},
		{"http/internal/router/home.go.tmpl", "internal/router/home.go", false},
	}

	for _, f := range appFiles {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate app file %s: %w", f.output, err)
		}
		if f.executable {
			outputPath := filepath.Join(appPath, f.output)
			if err := os.Chmod(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to set executable permission for %s: %w", f.output, err)
			}
		}
	}

	return nil
}

// templateData returns the data for template rendering
func (g *HTTPGenerator) templateData() map[string]interface{} {
	appNameSnake := strings.ReplaceAll(g.config.AppName, "-", "_")
	appNameUpper := strings.ToUpper(appNameSnake)
	projectNameSnake := strings.ReplaceAll(g.config.ProjectName, "-", "_")

	// Project module: github.com/myorg/my-project
	projectModule := fmt.Sprintf("%s/%s", g.config.OrgName, g.config.ProjectName)

	// Framework path for pkg (relative to project root)
	// If app framework path is "../../../go-yogan-framework", pkg path is "../go-yogan-framework"
	pkgFrameworkPath := "../go-yogan-framework"
	if g.config.FrameworkPath != "" {
		// Convert from apps/<app>/ relative to project root relative
		// ../../../go-yogan-framework -> ../go-yogan-framework
		pkgFrameworkPath = strings.TrimPrefix(g.config.FrameworkPath, "../../")
	}

	return map[string]interface{}{
		// Project level
		"ProjectName":        g.config.ProjectName,
		"ProjectNameSnake":   projectNameSnake,
		"ProjectModule":      projectModule,
		"OrgName":            g.config.OrgName,
		"PkgFrameworkPath":   pkgFrameworkPath,

		// App level
		"AppName":           g.config.AppName,
		"AppNamePascal":     ToPascalCase(g.config.AppName),
		"AppNameSnake":      appNameSnake,
		"AppNameUpper":      appNameUpper,
		"ModuleName":        g.config.ModuleName,
		"Description":       g.config.Description,
		"ServerPort":        g.config.ServerPort,
		"UseLocalFramework": g.config.UseLocalFramework,
		"FrameworkPath":     g.config.FrameworkPath,
	}
}

// renderTemplate renders a template file
// tmplName should be relative to templates/, e.g., "project/.gitignore.tmpl" or "http/main.go.tmpl"
func (g *HTTPGenerator) renderTemplate(outputDir, tmplName, outputName string, data map[string]interface{}) error {
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
