package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// WorkerGenerator generates a long-running queue worker application.
type WorkerGenerator struct {
	config *WorkerConfig
}

// NewWorkerGenerator creates a worker generator.
func NewWorkerGenerator(config *WorkerConfig) *WorkerGenerator {
	return &WorkerGenerator{config: config}
}

// Generate creates apps/<worker-name> in an existing multi-app workspace.
func (g *WorkerGenerator) Generate() error {
	if err := g.prepareConfig(); err != nil {
		return err
	}

	projectPath := g.config.WorkspacePath
	appPath := filepath.Join(projectPath, "apps", g.config.AppName)
	if err := ensureEmptyAppDir(projectPath, appPath); err != nil {
		return err
	}
	if err := g.createDirs(appPath); err != nil {
		return err
	}
	if err := g.renderFiles(appPath); err != nil {
		return err
	}
	return addGoWorkUse(filepath.Join(projectPath, "go.work"), "./apps/"+g.config.AppName)
}

func (g *WorkerGenerator) prepareConfig() error {
	if g.config.WorkspacePath == "" {
		root, err := FindWorkspaceRoot("")
		if err != nil {
			return err
		}
		g.config.WorkspacePath = root
	}
	abs, err := filepath.Abs(g.config.WorkspacePath)
	if err != nil {
		return err
	}
	g.config.WorkspacePath = abs
	if g.config.ProjectName == "" {
		g.config.ProjectName = filepath.Base(abs)
	}
	if g.config.OrgName == "" {
		g.config.OrgName = "github.com/KOMKZ"
	}
	if g.config.ModuleName == "" {
		g.config.ModuleName = fmt.Sprintf("%s/%s/apps/%s", g.config.OrgName, g.config.ProjectName, g.config.AppName)
	}
	if g.config.Description == "" {
		g.config.Description = fmt.Sprintf("%s worker", ToPascalCase(g.config.AppName))
	}
	if g.config.OutputPath == "" {
		g.config.OutputPath = "."
	}
	return g.config.Validate()
}

func ensureEmptyAppDir(projectPath, appPath string) error {
	if _, err := os.Stat(filepath.Join(projectPath, "go.work")); err != nil {
		return fmt.Errorf("workspace %q must contain go.work: %w", projectPath, err)
	}
	entries, err := os.ReadDir(appPath)
	if err == nil && len(entries) > 0 {
		return fmt.Errorf("%w: %s", ErrPathNotEmpty, appPath)
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect app directory %s: %w", appPath, err)
	}
	return nil
}

func (g *WorkerGenerator) createDirs(appPath string) error {
	dirs := []string{
		"",
		"build",
		"config",
		"internal/app",
		"internal/handler",
		"logs",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(appPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create worker directory %s: %w", dir, err)
		}
	}
	return nil
}

func (g *WorkerGenerator) renderFiles(appPath string) error {
	data := g.templateData()
	files := []struct {
		template string
		output   string
	}{
		{"worker/main.go.tmpl", "main.go"},
		{"worker/go.mod.tmpl", "go.mod"},
		{"worker/Makefile.tmpl", "Makefile"},
		{"worker/config/config.yaml.tmpl", "config/config.yaml"},
		{"worker/config/config.yaml.tmpl", "config/config.yaml.example"},
		{"worker/config/test.yaml.tmpl", "config/test.yaml"},
		{"worker/config/test.yaml.tmpl", "config/test.yaml.example"},
		{"worker/internal/app/app.go.tmpl", "internal/app/app.go"},
		{"worker/internal/app/callbacks.go.tmpl", "internal/app/callbacks.go"},
		{"worker/internal/app/handlers.go.tmpl", "internal/app/handlers.go"},
		{"worker/internal/app/app_test.go.tmpl", "internal/app/app_test.go"},
		{"worker/internal/handler/README.md.tmpl", "internal/handler/README.md"},
	}
	for _, f := range files {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate worker file %s: %w", f.output, err)
		}
	}
	return nil
}

func (g *WorkerGenerator) templateData() map[string]any {
	appNameSnake := strings.ReplaceAll(g.config.AppName, "-", "_")
	appNameUpper := strings.ToUpper(appNameSnake)
	return map[string]any{
		"AppName":           g.config.AppName,
		"AppNamePascal":     ToPascalCase(g.config.AppName),
		"AppNameSnake":      appNameSnake,
		"AppNameUpper":      appNameUpper,
		"Description":       g.config.Description,
		"Driver":            g.config.Driver,
		"ModuleName":        g.config.ModuleName,
		"ProjectName":       g.config.ProjectName,
		"Profiles":          g.config.Profiles,
		"UseLocalFramework": g.config.UseLocalFramework,
		"FrameworkPath":     g.config.FrameworkPath,
	}
}

func (g *WorkerGenerator) renderTemplate(outputDir, tmplName, outputName string, data map[string]any) error {
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
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()
	return tmpl.Execute(file, data)
}
