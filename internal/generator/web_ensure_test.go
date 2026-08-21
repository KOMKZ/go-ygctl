package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureWebLifecycle_Generate(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "legacy-app")
	if err := os.MkdirAll(filepath.Join(app, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	// 伪造 package.json（存量应用判定）
	if err := os.WriteFile(filepath.Join(app, "package.json"), []byte(`{"name":"legacy-app"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &EnsureWebLifecycleConfig{AppPath: app, AppPort: 3200}
	result, err := cfg.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 4 {
		t.Errorf("files = %d, want 4", len(result.Files))
	}
	mk, _ := os.ReadFile(filepath.Join(app, "Makefile"))
	if !strings.Contains(string(mk), "PROJECT := legacy-app") ||
		!strings.Contains(string(mk), "DEV_PORT ?= 3200") ||
		!strings.Contains(string(mk), "test-e2e-smoke") {
		t.Errorf("Makefile content wrong:\n%s", mk)
	}
	// 脚本执行位
	info, err := os.Stat(filepath.Join(app, "scripts", "runtime.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("runtime.sh not executable: %v", info.Mode())
	}
	// 幂等：重复 ensure 成功
	if _, err := cfg.Generate(); err != nil {
		t.Errorf("second ensure failed: %v", err)
	}
}

func TestEnsureWebLifecycle_RefusesManualFile(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "manual-app")
	if err := os.MkdirAll(app, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "package.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Makefile"), []byte("# my hand-written Makefile\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := (&EnsureWebLifecycleConfig{AppPath: app}).Generate()
	if err == nil || !strings.Contains(err.Error(), "not ygctl-managed") {
		t.Errorf("error = %v, want refuse-overwrite message", err)
	}
}
