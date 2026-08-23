package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkerGenerator_GenerateIntoExistingWorkspace(t *testing.T) {
	workspace := newTestWorkspace(t)

	cfg := NewDefaultWorkerConfig()
	cfg.ProjectName = "hrise-server-app"
	cfg.OrgName = "github.com/KOMKZ"
	cfg.AppName = "hrise-worker"
	cfg.ModuleName = "github.com/KOMKZ/hrise-server-app/apps/hrise-worker"
	cfg.WorkspacePath = workspace
	cfg.Profiles = []string{"default", "media"}

	if err := NewWorkerGenerator(cfg).Generate(); err != nil {
		t.Fatalf("Generate() err = %v", err)
	}

	appPath := filepath.Join(workspace, "apps", "hrise-worker")
	assertFilesExist(t, appPath, []string{
		"go.mod",
		"main.go",
		"config/config.yaml",
		"internal/app/app.go",
		"internal/app/handlers.go",
	})
	assertFileContains(t, filepath.Join(appPath, "config/config.yaml"), "media:")
	assertFileContains(t, filepath.Join(appPath, "config/config.yaml"), "brokers:")
	assertFileContains(t, filepath.Join(appPath, "config/config.yaml"), "workers:")
	assertFileContains(t, filepath.Join(appPath, "internal/app/handlers.go"), "registerHandlers")
	assertFileContains(t, filepath.Join(workspace, "go.work"), "./apps/hrise-worker")
	assertFileNotContains(t, filepath.Join(appPath, "config/config.yaml"), "max_retry")
	assertFileNotContains(t, filepath.Join(appPath, "config/config.yaml"), "key_prefix")
}

func TestDomainJobGenerator_Generate(t *testing.T) {
	workspace := newTestWorkspace(t)
	domainDir := filepath.Join(workspace, "domains", "export")
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &DomainJobConfig{
		DomainKey:     "export",
		TaskType:      "export.demo",
		Mode:          "queue",
		WorkspacePath: workspace,
	}
	if err := NewDomainJobGenerator(cfg).Generate(); err != nil {
		t.Fatalf("Generate() err = %v", err)
	}

	jobDir := filepath.Join(domainDir, "job")
	assertFilesExist(t, jobDir, []string{
		"demo_task_defs.go",
		"demo_executor.go",
		"demo_publisher.go",
	})
	assertFileContains(t, filepath.Join(jobDir, "demo_task_defs.go"), `TaskTypeExportDemo = "export.demo"`)
	assertFileContains(t, filepath.Join(jobDir, "demo_task_defs.go"), "frameworkqueue.TaskDefinition[DemoCommand]")
	assertFileContains(t, filepath.Join(jobDir, "demo_executor.go"), "func (e *DemoExecutor) Execute")
	assertFileContains(t, filepath.Join(jobDir, "demo_publisher.go"), "jobruntime.Publish")
	assertFileNotContains(t, filepath.Join(jobDir, "demo_publisher.go"), "LedgerPort")
	assertFileNotContains(t, filepath.Join(jobDir, "demo_publisher.go"), "MarkEnqueueFailed")
}

func newTestWorkspace(t *testing.T) string {
	t.Helper()
	workspace := filepath.Join(t.TempDir(), "hrise-server-app")
	if err := os.MkdirAll(filepath.Join(workspace, "apps", "hrise-admin-api"), 0755); err != nil {
		t.Fatal(err)
	}
	goWork := `go 1.24.1

use (
	./apps/hrise-admin-api
)
`
	if err := os.WriteFile(filepath.Join(workspace, "go.work"), []byte(goWork), 0644); err != nil {
		t.Fatal(err)
	}
	goMod := "module github.com/KOMKZ/hrise-server-app/apps/hrise-admin-api\n\ngo 1.24.1\n"
	if err := os.WriteFile(filepath.Join(workspace, "apps", "hrise-admin-api", "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func assertFilesExist(t *testing.T, root string, files []string) {
	t.Helper()
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatalf("generated file %s missing: %v", file, err)
		}
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, content)
	}
}

func assertFileNotContains(t *testing.T, path, forbidden string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), forbidden) {
		t.Fatalf("%s contains forbidden %q:\n%s", path, forbidden, content)
	}
}
