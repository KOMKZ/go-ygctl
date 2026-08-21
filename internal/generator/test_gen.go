package generator

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates/test
var testTemplates embed.FS

// TestGenConfig holds the configuration for e2e test generation.
type TestGenConfig struct {
	WorkspacePath string
	AppName       string
	ModuleName    string // e.g. "auth" -> test/auth/
}

// TestGenResult reports the test generation result.
type TestGenResult struct {
	TestDir string
}

// Generate creates the e2e test skeleton under apps/<app>/test/<module>/.
func (c *TestGenConfig) Generate() (*TestGenResult, error) {
	module := c.ModuleName
	if module == "" || strings.ContainsAny(module, "/\\") {
		return nil, fmt.Errorf("module name must be a plain directory name, got %q", module)
	}

	workspace, err := resolveWorkspacePath(c.WorkspacePath)
	if err != nil {
		return nil, err
	}

	migCfg := &MigrateConfig{WorkspacePath: workspace, AppName: c.AppName}
	_, appDir, _, err := migCfg.resolve()
	if err != nil {
		return nil, err
	}
	appModule, err := readModulePath(appDir)
	if err != nil {
		return nil, err
	}

	testDir := filepath.Join(appDir, "test", module)
	if _, err := os.Stat(testDir); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrPathExists, testDir)
	}
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return nil, err
	}

	data := map[string]string{
		"AppModule":    appModule,
		"ModuleName":   module,
		"ModulePascal": ToPascalCase(module),
	}

	files := []struct{ tmpl, out string }{
		{"setup_test.go.tmpl", "setup_test.go"},
		{"test.go.tmpl", module + "_test.go"},
	}
	for _, f := range files {
		outPath := filepath.Join(testDir, f.out)
		content, err := testTemplates.ReadFile(filepath.Join("templates", "test", f.tmpl))
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", f.tmpl, err)
		}
		tmpl, err := template.New(f.tmpl).Parse(string(content))
		if err != nil {
			return nil, err
		}
		file, err := os.Create(outPath)
		if err != nil {
			return nil, err
		}
		if err := tmpl.Execute(file, data); err != nil {
			file.Close()
			return nil, err
		}
		file.Close()
	}

	// gofmt the generated files.
	for _, f := range []string{"setup_test.go", module + "_test.go"} {
		if out, err := exec.Command("gofmt", "-w", filepath.Join(testDir, f)).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("gofmt failed: %v\n%s", err, out)
		}
	}
	return &TestGenResult{TestDir: testDir}, nil
}
