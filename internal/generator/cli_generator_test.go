package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIGenerator_GenerateIntoExistingWorkspace(t *testing.T) {
	dir := t.TempDir()
	workspace := filepath.Join(dir, "hrise-server-app")
	if err := os.MkdirAll(filepath.Join(workspace, "apps", "hrise-cli"), 0755); err != nil {
		t.Fatal(err)
	}
	goWork := `go 1.24.1

use (
	./apps/hrise-admin-api
	./pkg
)

replace github.com/KOMKZ/go-yogan-framework => ../go-yogan-framework
`
	if err := os.WriteFile(filepath.Join(workspace, "go.work"), []byte(goWork), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefaultCLIConfig()
	cfg.ProjectName = "hrise-server-app"
	cfg.OrgName = "github.com/KOMKZ"
	cfg.AppName = "hrise-cli"
	cfg.ModuleName = "github.com/KOMKZ/hrise-server-app/apps/hrise-cli"
	cfg.OutputPath = dir
	cfg.WorkspacePath = workspace

	if err := NewCLIGenerator(cfg).Generate(); err != nil {
		t.Fatalf("Generate() err = %v", err)
	}

	appPath := filepath.Join(workspace, "apps", "hrise-cli")
	for _, file := range []string{
		"go.mod",
		"main.go",
		"internal/app/app.go",
		"internal/command/root.go",
		"internal/module/hello/command.go",
		"internal/service/hello_service.go",
	} {
		if _, err := os.Stat(filepath.Join(appPath, file)); err != nil {
			t.Fatalf("generated file %s missing: %v", file, err)
		}
	}
	cmdContent, err := os.ReadFile(filepath.Join(appPath, "internal/module/hello/command.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cmdContent), `Use:   "hello"`) ||
		!strings.Contains(string(cmdContent), `cmd.Println(svc.Message(cmd.Context()))`) {
		t.Fatalf("hello command template mismatch:\n%s", cmdContent)
	}
	updatedGoWork, err := os.ReadFile(filepath.Join(workspace, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(updatedGoWork), "./apps/hrise-cli") != 1 {
		t.Fatalf("go.work should contain one hrise-cli use entry:\n%s", updatedGoWork)
	}

	if err := addGoWorkUse(filepath.Join(workspace, "go.work"), "./apps/hrise-cli"); err != nil {
		t.Fatalf("second go.work registration failed: %v", err)
	}
	updatedGoWork, err = os.ReadFile(filepath.Join(workspace, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(updatedGoWork), "./apps/hrise-cli") != 1 {
		t.Fatalf("go.work registration is not idempotent:\n%s", updatedGoWork)
	}
}

func TestCLIGenerator_RefusesNonEmptyExistingApp(t *testing.T) {
	dir := t.TempDir()
	workspace := filepath.Join(dir, "hrise-server-app")
	appPath := filepath.Join(workspace, "apps", "hrise-cli")
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "go.work"), []byte("go 1.24.1\n\nuse (\n)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appPath, "README.md"), []byte("manual\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := NewDefaultCLIConfig()
	cfg.ProjectName = "hrise-server-app"
	cfg.OrgName = "github.com/KOMKZ"
	cfg.AppName = "hrise-cli"
	cfg.ModuleName = "github.com/KOMKZ/hrise-server-app/apps/hrise-cli"
	cfg.OutputPath = dir
	cfg.WorkspacePath = workspace

	err := NewCLIGenerator(cfg).Generate()
	if err == nil || !strings.Contains(err.Error(), ErrPathNotEmpty.Error()) {
		t.Fatalf("Generate() err = %v, want ErrPathNotEmpty", err)
	}
}
