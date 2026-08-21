package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureWebLifecycleConfig configures ygctl web ensure: injects the Makefile
// + scripts lifecycle into an existing web app (idempotent).
type EnsureWebLifecycleConfig struct {
	AppPath       string
	Project       string // Makefile PROJECT; default: base name of AppPath
	AppPort       int    // dev port; default 3100
	BackendAppDir string // startall/stopall backend dir; default hrise-admin-api
}

// EnsureWebLifecycleResult reports the ensured files.
type EnsureWebLifecycleResult struct {
	Files []string
}

// lifecycleFiles lists the Makefile + runtime scripts injected by ensure.
var lifecycleFiles = []struct {
	tmpl string // path under templates/web/new/ (".tmpl" 表示需渲染)
	out  string // relative path in the app
}{
	{"Makefile.tmpl", "Makefile"},
	{"scripts/app.sh.tmpl", "scripts/app.sh"},
	{"scripts/runtime.sh", "scripts/runtime.sh"},
	{"scripts/toolchain.sh", "scripts/toolchain.sh"},
}

// ygctlMark identifies ygctl-managed files (safe to overwrite).
const ygctlMark = "ygctl"

// Generate ensures Makefile + scripts exist and are ygctl-managed versions.
// 已存在的文件：含 ygctl 标记 → 覆盖刷新；不含标记 → 拒绝（防误伤手写文件）。
func (c *EnsureWebLifecycleConfig) Generate() (*EnsureWebLifecycleResult, error) {
	if c.AppPath == "" {
		return nil, fmt.Errorf("app path is required (--app)")
	}
	if _, err := os.Stat(filepath.Join(c.AppPath, "package.json")); err != nil {
		return nil, fmt.Errorf("app %q does not look like a web app (package.json missing)", c.AppPath)
	}
	if c.Project == "" {
		c.Project = filepath.Base(c.AppPath)
	}
	if c.AppPort == 0 {
		c.AppPort = 3100
	}
	if c.BackendAppDir == "" {
		c.BackendAppDir = "hrise-admin-api"
	}

	data := &webNewData{
		AppName:       c.Project,
		AppTitle:      kebabToTitle(c.Project),
		AppPort:       c.AppPort,
		BackendAppDir: c.BackendAppDir,
	}

	result := &EnsureWebLifecycleResult{}
	for _, f := range lifecycleFiles {
		outPath := filepath.Join(c.AppPath, filepath.FromSlash(f.out))
		if err := ensureManagedFile(outPath, f.tmpl, data); err != nil {
			return nil, err
		}
		result.Files = append(result.Files, outPath)
	}

	// shell 脚本恢复执行位
	for _, f := range result.Files {
		if strings.HasSuffix(f, ".sh") {
			if err := os.Chmod(f, 0755); err != nil {
				return nil, fmt.Errorf("chmod %s: %w", f, err)
			}
		}
	}
	return result, nil
}

// ensureManagedFile writes one lifecycle file, refusing to overwrite
// non-ygctl-managed files.
func ensureManagedFile(outPath, tmplName string, data *webNewData) error {
	existing, err := os.ReadFile(outPath)
	if err == nil && !strings.Contains(string(existing), ygctlMark) {
		return fmt.Errorf("%s exists and is not ygctl-managed; remove it or add a ygctl marker to allow overwrite", outPath)
	}

	if strings.HasSuffix(tmplName, ".tmpl") {
		return renderNewTemplate(filepath.Join("templates/web/new", tmplName), outPath, data)
	}
	content, err := webTemplates.ReadFile("templates/web/new/" + tmplName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, content, 0755)
}
