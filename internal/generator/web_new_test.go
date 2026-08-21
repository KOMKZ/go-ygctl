package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebNew_Generate(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "hrise-admin-web-demo")
	cfg := &WebNewConfig{
		AppPath:        out,
		AppName:        "hrise-admin-web-demo",
		AppTitle:       "HRise Admin Demo",
		UILink:         "link:../rong-admin-ui",
		AppPort:        3200,
		APIProxyTarget: "http://localhost:9201",
		StoragePrefix:  "demo",
	}
	result, err := cfg.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if result.Files == 0 {
		t.Fatal("no files generated")
	}

	// 关键文件存在
	for _, f := range []string{
		"package.json", "Makefile", "index.html", "vite.config.ts", "README.md",
		"src/main.ts", "src/app.ts", "src/router/index.ts",
		"src/views/login/index.vue", "src/views/dashboard/index.vue",
		"src/layouts/components/SidebarMenu.vue",
		"src/api/adapter.ts", "src/api/types.ts",
		"scripts/app.sh", "scripts/runtime.sh",
		".gitignore", "tsconfig.json",
	} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing generated file %s: %v", f, err)
		}
	}

	// 参数化渲染生效
	pkg, _ := os.ReadFile(filepath.Join(out, "package.json"))
	if !strings.Contains(string(pkg), `"name": "hrise-admin-web-demo"`) ||
		!strings.Contains(string(pkg), `"link:../rong-admin-ui"`) {
		t.Errorf("package.json not parameterized:\n%s", pkg)
	}
	vite, _ := os.ReadFile(filepath.Join(out, "vite.config.ts"))
	if !strings.Contains(string(vite), "port: 3200") {
		t.Errorf("vite port not parameterized:\n%s", vite)
	}
	router, _ := os.ReadFile(filepath.Join(out, "src/router/index.ts"))
	if !strings.Contains(string(router), "HRise Admin Demo") {
		t.Errorf("router title not parameterized:\n%s", router)
	}
	if !strings.Contains(string(router), routeMarkerBegin) || !strings.Contains(string(router), routeMarkerEnd) {
		t.Errorf("router markers missing in generated app")
	}
	mainTS, _ := os.ReadFile(filepath.Join(out, "src/main.ts"))
	if !strings.Contains(string(mainTS), "demo-login-hint") {
		t.Errorf("storage prefix not parameterized in main.ts")
	}

	// shell 脚本原样复制（bash [[ ]] 与模板定界符不冲突）
	runtime, _ := os.ReadFile(filepath.Join(out, "scripts/runtime.sh"))
	if !strings.Contains(string(runtime), `if [[ -z "$command" ]]`) {
		t.Errorf("runtime.sh verbatim copy broken:\n%s", runtime)
	}
}

func TestWebNew_InvalidName(t *testing.T) {
	cfg := &WebNewConfig{AppName: "BadName", UILink: "link:x"}
	_, err := cfg.Generate()
	if err == nil || !strings.Contains(err.Error(), "lowercase kebab-case") {
		t.Errorf("error = %v, want name validation message", err)
	}
}

func TestWebNew_RequiresUILink(t *testing.T) {
	cfg := &WebNewConfig{AppName: "demo-app", AppPath: t.TempDir() + "/demo-app"}
	_, err := cfg.Generate()
	if err == nil || !strings.Contains(err.Error(), "ui link") {
		t.Errorf("error = %v, want ui link required message", err)
	}
}
