package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates/http
var httpTemplates embed.FS

// HTTPGenerator generates HTTP application from templates
type HTTPGenerator struct {
	config *AppConfig
}

// NewHTTPGenerator creates a new HTTP generator
func NewHTTPGenerator(config *AppConfig) *HTTPGenerator {
	return &HTTPGenerator{config: config}
}

// Generate creates the HTTP application
func (g *HTTPGenerator) Generate() error {
	if err := g.config.Validate(); err != nil {
		return err
	}

	appPath := filepath.Join(g.config.OutputPath, g.config.AppName)

	// Check if path exists
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrPathExists, appPath)
	}

	// Create directory structure
	dirs := []string{
		"",
		"build",
		"configs",
		"migrations",
		"scripts",
		"internal/app",
		"internal/config",
		"internal/domain/home/model",
		"internal/module/home",
		"internal/router",
		"pkg/util",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(appPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files from templates
	files := []struct {
		template   string
		output     string
		executable bool
	}{
		{".gitignore.tmpl", ".gitignore", false},
		{"README.md.tmpl", "README.md", false},
		{"main.go.tmpl", "main.go", false},
		{"go.mod.tmpl", "go.mod", false},
		{"Makefile.tmpl", "Makefile", false},
		{"build/.gitkeep.tmpl", "build/.gitkeep", false},
		{"configs/config.yaml.tmpl", "configs/config.yaml", false},
		{"migrations/README.md.tmpl", "migrations/README.md", false},
		{"scripts/migrate.sh.tmpl", "scripts/migrate.sh", true},
		{"internal/app/app.go.tmpl", "internal/app/app.go", false},
		{"internal/app/callbacks.go.tmpl", "internal/app/callbacks.go", false},
		{"internal/app/components.go.tmpl", "internal/app/components.go", false},
		{"internal/app/router.go.tmpl", "internal/app/router.go", false},
		{"internal/config/config.go.tmpl", "internal/config/config.go", false},
		{"internal/domain/home/model/home.go.tmpl", "internal/domain/home/model/home.go", false},
		{"internal/domain/home/repository.go.tmpl", "internal/domain/home/repository.go", false},
		{"internal/domain/home/service.go.tmpl", "internal/domain/home/service.go", false},
		{"internal/module/home/handler.go.tmpl", "internal/module/home/handler.go", false},
		{"internal/module/home/request.go.tmpl", "internal/module/home/request.go", false},
		{"internal/module/home/response.go.tmpl", "internal/module/home/response.go", false},
		{"internal/router/home.go.tmpl", "internal/router/home.go", false},
		{"pkg/util/ptr.go.tmpl", "pkg/util/ptr.go", false},
		{"pkg/util/string.go.tmpl", "pkg/util/string.go", false},
		{"pkg/README.md.tmpl", "pkg/README.md", false},
	}

	data := g.templateData()

	for _, f := range files {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", f.output, err)
		}
		// Set executable permission for scripts
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

	return map[string]interface{}{
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
func (g *HTTPGenerator) renderTemplate(appPath, tmplName, outputName string, data map[string]interface{}) error {
	tmplPath := filepath.Join("templates/http", tmplName)
	content, err := httpTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	outputPath := filepath.Join(appPath, outputName)
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
