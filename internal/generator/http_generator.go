package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/http/*
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
		"configs",
		"internal/app",
		"internal/config",
		"internal/domain/demo/model",
		"internal/module/demo",
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
		template string
		output   string
	}{
		{"main.go.tmpl", "main.go"},
		{"go.mod.tmpl", "go.mod"},
		{"configs/config.yaml.tmpl", "configs/config.yaml"},
		{"internal/app/app.go.tmpl", "internal/app/app.go"},
		{"internal/app/callbacks.go.tmpl", "internal/app/callbacks.go"},
		{"internal/app/components.go.tmpl", "internal/app/components.go"},
		{"internal/app/router.go.tmpl", "internal/app/router.go"},
		{"internal/config/config.go.tmpl", "internal/config/config.go"},
		{"internal/domain/demo/model/demo.go.tmpl", "internal/domain/demo/model/demo.go"},
		{"internal/domain/demo/repository.go.tmpl", "internal/domain/demo/repository.go"},
		{"internal/domain/demo/repository_memory.go.tmpl", "internal/domain/demo/repository_memory.go"},
		{"internal/domain/demo/service.go.tmpl", "internal/domain/demo/service.go"},
		{"internal/module/demo/handler.go.tmpl", "internal/module/demo/handler.go"},
		{"internal/module/demo/request.go.tmpl", "internal/module/demo/request.go"},
		{"internal/module/demo/response.go.tmpl", "internal/module/demo/response.go"},
		{"internal/router/demo.go.tmpl", "internal/router/demo.go"},
		{"internal/router/health.go.tmpl", "internal/router/health.go"},
		{"pkg/util/ptr.go.tmpl", "pkg/util/ptr.go"},
		{"pkg/util/string.go.tmpl", "pkg/util/string.go"},
		{"pkg/README.md.tmpl", "pkg/README.md"},
	}

	data := g.templateData()

	for _, f := range files {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", f.output, err)
		}
	}

	return nil
}

// templateData returns the data for template rendering
func (g *HTTPGenerator) templateData() map[string]interface{} {
	// Convert app-name to APP_NAME for env prefix
	appNameUpper := strings.ToUpper(strings.ReplaceAll(g.config.AppName, "-", "_"))

	return map[string]interface{}{
		"AppName":           g.config.AppName,
		"AppNamePascal":     ToPascalCase(g.config.AppName),
		"AppNameUpper":      appNameUpper,
		"ModuleName":        g.config.ModuleName,
		"Description":       g.config.Description,
		"ServerPort":        g.config.ServerPort,
		"UseLocalFramework": g.config.UseLocalFramework,
		"FrameworkPath":     g.config.FrameworkPath,
		"EnableDatabase":    g.config.EnableDatabase,
		"EnableRedis":       g.config.EnableRedis,
	}
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
		"Replace": strings.ReplaceAll,
	}
}

// renderTemplate renders a template file
func (g *HTTPGenerator) renderTemplate(appPath, tmplName, outputName string, data map[string]interface{}) error {
	tmplPath := filepath.Join("templates/http", tmplName)
	content, err := httpTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Funcs(templateFuncs()).Parse(string(content))
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
