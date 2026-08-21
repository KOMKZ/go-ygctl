package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindUsedPorts_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	used, err := FindUsedPorts(tmpDir, "no-such-project")
	if err != nil {
		t.Fatalf("FindUsedPorts: %v", err)
	}
	if len(used) != 0 {
		t.Fatalf("expected no used ports, got %v", used)
	}
}

func TestFindUsedPorts_ScansExistingApps(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, "demo-project", "apps", "demo-api", "config")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "api_server:\n  host: \"0.0.0.0\"\n  port: 9201\n  mode: \"debug\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	used, err := FindUsedPorts(tmpDir, "demo-project")
	if err != nil {
		t.Fatalf("FindUsedPorts: %v", err)
	}
	if !used[9201] {
		t.Fatalf("expected port 9201 to be used, got %v", used)
	}
}

func TestAllocateFreePort_AvoidsUsedPorts(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, "demo-project", "apps", "demo-api", "config")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "api_server:\n  port: 8080\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 8080 is taken -> must allocate 8081
	port, err := AllocateFreePort(tmpDir, "demo-project", 8080)
	if err != nil {
		t.Fatalf("AllocateFreePort: %v", err)
	}
	if port != 8081 {
		t.Fatalf("expected 8081, got %d", port)
	}
}

func TestAllocateFreePort_ReturnsBaseWhenFree(t *testing.T) {
	tmpDir := t.TempDir()

	port, err := AllocateFreePort(tmpDir, "demo-project", 8080)
	if err != nil {
		t.Fatalf("AllocateFreePort: %v", err)
	}
	if port != 8080 {
		t.Fatalf("expected 8080, got %d", port)
	}
}
