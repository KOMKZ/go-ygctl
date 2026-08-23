package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var taskTypeRe = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$`)

// DomainJobConfig configures domain job file generation.
type DomainJobConfig struct {
	DomainKey     string
	TaskType      string
	Mode          string
	WorkspacePath string
	ModuleBase    string
}

// DomainJobGenerator generates domain-owned job publisher/executor files.
type DomainJobGenerator struct {
	config     *DomainJobConfig
	rootModule string
}

// NewDomainJobGenerator creates a generator for domain job files.
func NewDomainJobGenerator(config *DomainJobConfig) *DomainJobGenerator {
	return &DomainJobGenerator{config: config}
}

// Generate writes domains/<domain>/job/*.go files.
func (g *DomainJobGenerator) Generate() error {
	if err := g.prepare(); err != nil {
		return err
	}
	domainDir := filepath.Join(g.config.WorkspacePath, "domains", g.config.DomainKey)
	if _, err := os.Stat(domainDir); err != nil {
		return fmt.Errorf("domain %q does not exist (run: ygctl domain init %s): %w", g.config.DomainKey, g.config.DomainKey, err)
	}

	jobDir := filepath.Join(domainDir, "job")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return fmt.Errorf("failed to create job dir: %w", err)
	}

	data := g.templateData()
	files := []struct {
		template string
		output   string
	}{
		{"job/task_defs.go.tmpl", data["TaskFilePrefix"].(string) + "_task_defs.go"},
		{"job/executor.go.tmpl", data["TaskFilePrefix"].(string) + "_executor.go"},
		{"job/publisher.go.tmpl", data["TaskFilePrefix"].(string) + "_publisher.go"},
	}

	var goFiles []string
	for _, f := range files {
		outPath := filepath.Join(jobDir, f.output)
		if _, err := os.Stat(outPath); err == nil {
			return fmt.Errorf("%w: %s", ErrPathExists, outPath)
		}
		if err := g.render(outPath, f.template, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", f.output, err)
		}
		goFiles = append(goFiles, outPath)
	}

	args := append([]string{"-w"}, goFiles...)
	if out, err := exec.Command("gofmt", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}
	return nil
}

func (g *DomainJobGenerator) prepare() error {
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
	if g.config.Mode == "" {
		g.config.Mode = "queue"
	}
	if g.config.Mode != "queue" && g.config.Mode != "sync" {
		return fmt.Errorf("invalid mode %q: use queue or sync", g.config.Mode)
	}
	if !domainKeyRe.MatchString(g.config.DomainKey) {
		return fmt.Errorf("invalid domain name %q: use lowercase kebab-case", g.config.DomainKey)
	}
	if !taskTypeRe.MatchString(g.config.TaskType) {
		return fmt.Errorf("invalid task type %q: use dotted lowercase form, e.g. export.demo", g.config.TaskType)
	}
	if g.config.ModuleBase != "" {
		g.rootModule = g.config.ModuleBase
		return nil
	}
	base, err := deriveModuleBase(g.config.WorkspacePath)
	if err != nil {
		return err
	}
	g.rootModule = base
	return nil
}

func (g *DomainJobGenerator) templateData() map[string]any {
	taskName := taskLocalName(g.config.TaskType)
	domainPascal := ToPascalCase(g.config.DomainKey)
	taskPascal := ToPascalCase(taskName)
	return map[string]any{
		"DomainKey":        g.config.DomainKey,
		"DomainPascal":     domainPascal,
		"DomainImport":     g.rootModule + "/domains/" + g.config.DomainKey,
		"JobImport":        g.rootModule + "/domains/job",
		"JobRuntimeImport": g.rootModule + "/domains/job/runtime",
		"TaskType":         g.config.TaskType,
		"TaskConstName":    "TaskType" + domainPascal + taskPascal,
		"TaskFilePrefix":   ToSnakeCase(taskPascal),
		"TaskPascal":       taskPascal,
		"Mode":             g.config.Mode,
	}
}

func taskLocalName(taskType string) string {
	parts := strings.Split(taskType, ".")
	if len(parts) == 0 {
		return taskType
	}
	return parts[len(parts)-1]
}

func (g *DomainJobGenerator) render(outPath, tmplName string, data map[string]any) error {
	tmplPath := filepath.Join("templates", "domain", tmplName)
	content, err := domainTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}
	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, data)
}
