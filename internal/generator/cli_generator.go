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

	projectPath := g.projectPath()
	appPath := filepath.Join(projectPath, "apps", g.config.AppName)

	if g.config.WorkspacePath != "" {
		if err := g.ensureExistingWorkspace(projectPath, appPath); err != nil {
			return err
		}
		return g.generateAppOnly(projectPath, appPath)
	}

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

	return g.renderAppFiles(appPath, data)
}

func (g *CLIGenerator) projectPath() string {
	if g.config.WorkspacePath != "" {
		return g.config.WorkspacePath
	}
	return filepath.Join(g.config.OutputPath, g.config.ProjectName)
}

func (g *CLIGenerator) ensureExistingWorkspace(projectPath, appPath string) error {
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

func (g *CLIGenerator) generateAppOnly(projectPath, appPath string) error {
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
	if err := g.renderAppFiles(appPath, g.templateData()); err != nil {
		return err
	}
	return addGoWorkUse(filepath.Join(projectPath, "go.work"), "./apps/"+g.config.AppName)
}

func (g *CLIGenerator) renderAppFiles(appPath string, data map[string]interface{}) error {
	appFiles := []struct {
		template string
		output   string
	}{
		{"cli/main.go.tmpl", "main.go"},
		{"cli/main_test.go.tmpl", "main_test.go"},
		{"cli/go.mod.tmpl", "go.mod"},
		{"cli/Makefile.tmpl", "Makefile"},
		{"cli/config/config.yaml.tmpl", "config/config.yaml"},
		{"cli/config/config.yaml.tmpl", "config/config.yaml.example"},
		{"cli/config/test.yaml.tmpl", "config/test.yaml"},
		{"cli/config/test.yaml.tmpl", "config/test.yaml.example"},
		{"cli/internal/app/app.go.tmpl", "internal/app/app.go"},
		{"cli/internal/app/app_test.go.tmpl", "internal/app/app_test.go"},
		{"cli/internal/command/hello.go.tmpl", "internal/command/hello.go"},
		{"cli/internal/command/hello_test.go.tmpl", "internal/command/hello_test.go"},
	}

	for _, f := range appFiles {
		if err := g.renderTemplate(appPath, f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate app file %s: %w", f.output, err)
		}
	}
	return nil
}

func addGoWorkUse(goWorkPath, usePath string) error {
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return fmt.Errorf("read go.work: %w", err)
	}
	text := string(content)
	if strings.Contains(text, "\n\t"+usePath+"\n") || strings.Contains(text, "\n\t"+usePath+"\r\n") {
		return nil
	}
	useStart := strings.Index(text, "use (")
	if useStart < 0 {
		return fmt.Errorf("go.work must contain a use (...) block")
	}
	blockEndRel := strings.Index(text[useStart:], "\n)")
	if blockEndRel < 0 {
		return fmt.Errorf("go.work use block is not closed")
	}
	insertAt := useStart + blockEndRel
	updated := text[:insertAt] + "\n\t" + usePath + text[insertAt:]
	if err := os.WriteFile(goWorkPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("write go.work: %w", err)
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

	// pkgFrameworkPath is kept for template compatibility; the authoritative
	// replace now lives in go.work (see WorkspaceFrameworkPath).
	pkgFrameworkPath := g.config.FrameworkPath

	// Framework path for go.work (relative to the project root): the
	// workspace-level replace is the single source of truth in workspace mode.
	projectPath := g.projectPath()
	appPath := filepath.Join(projectPath, "apps", g.config.AppName)
	frameworkAbs := resolveFrameworkAbs(appPath, g.config.FrameworkPath)
	workspaceFrameworkPath, relErr := filepath.Rel(projectPath, frameworkAbs)
	if relErr != nil {
		workspaceFrameworkPath = g.config.FrameworkPath
	}

	return map[string]interface{}{
		// Project level
		"ProjectName":            g.config.ProjectName,
		"ProjectNameSnake":       projectNameSnake,
		"ProjectModule":          projectModule,
		"OrgName":                g.config.OrgName,
		"PkgFrameworkPath":       pkgFrameworkPath,
		"WorkspaceFrameworkPath": workspaceFrameworkPath,
		"GenerateProto":          false, // CLI doesn't need proto

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
