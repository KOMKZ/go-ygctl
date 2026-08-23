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
	if err := os.MkdirAll(filepath.Join(workspace, "apps", "hrse-cli"), 0755); err != nil {
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
	cfg.AppName = "hrse-cli"
	cfg.ModuleName = "github.com/KOMKZ/hrise-server-app/apps/hrse-cli"
	cfg.OutputPath = dir
	cfg.WorkspacePath = workspace

	if err := NewCLIGenerator(cfg).Generate(); err != nil {
		t.Fatalf("Generate() err = %v", err)
	}

	appPath := filepath.Join(workspace, "apps", "hrse-cli")
	for _, file := range []string{
		"go.mod",
		"main.go",
		"internal/app/app.go",
		"internal/command/hello.go",
	} {
		if _, err := os.Stat(filepath.Join(appPath, file)); err != nil {
			t.Fatalf("generated file %s missing: %v", file, err)
		}
	}
	cmdContent, err := os.ReadFile(filepath.Join(appPath, "internal/command/hello.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cmdContent), `Use:   "hello"`) ||
		!strings.Contains(string(cmdContent), `cmd.Println("hello world")`) {
		t.Fatalf("hello command template mismatch:\n%s", cmdContent)
	}
	updatedGoWork, err := os.ReadFile(filepath.Join(workspace, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(updatedGoWork), "./apps/hrse-cli") != 1 {
		t.Fatalf("go.work should contain one hrse-cli use entry:\n%s", updatedGoWork)
	}

	if err := addGoWorkUse(filepath.Join(workspace, "go.work"), "./apps/hrse-cli"); err != nil {
		t.Fatalf("second go.work registration failed: %v", err)
	}
	updatedGoWork, err = os.ReadFile(filepath.Join(workspace, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(updatedGoWork), "./apps/hrse-cli") != 1 {
		t.Fatalf("go.work registration is not idempotent:\n%s", updatedGoWork)
	}
}

func TestCLIGenerator_RefusesNonEmptyExistingApp(t *testing.T) {
	dir := t.TempDir()
	workspace := filepath.Join(dir, "hrise-server-app")
	appPath := filepath.Join(workspace, "apps", "hrse-cli")
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
	cfg.AppName = "hrse-cli"
	cfg.ModuleName = "github.com/KOMKZ/hrise-server-app/apps/hrse-cli"
	cfg.OutputPath = dir
	cfg.WorkspacePath = workspace

	err := NewCLIGenerator(cfg).Generate()
	if err == nil || !strings.Contains(err.Error(), ErrPathNotEmpty.Error()) {
		t.Fatalf("Generate() err = %v, want ErrPathNotEmpty", err)
	}
}
