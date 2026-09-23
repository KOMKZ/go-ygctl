// 本文件是 DomainGenerator 的 fixture/golden 回归测试（ticket 000188 阶段 B）。
// 锁定三件事：
//  1. 骨架文件计划与 hrise-server-app 现行领域基线一致——所有领域都有 uploads/README.md
//     上传能力槽位，--pure 裁剪表骨架与 event/policy/assembler/contract/port；
//  2. 模板渲染以 domain key 渲染且字节级确定（两次渲染结果一致，无未渲染占位）；
//  3. 非法/重复领域名在触达 go 工具链前被拒绝。
//
// 上游入口：DomainGenerator.Generate（生产路径）。
// 下游依赖：templates/domain 嵌入模板；不执行 gofmt/go mod tidy/go build（那些由
// 仓库外临时 workspace 的真实 domain init 冒烟覆盖）。
package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// golden 输出路径集合：表域（非 pure）完整骨架。
func goldenTableDomainFiles(entitySnake string) map[string]bool {
	files := map[string]bool{
		"go.mod": true, ".gitignore": true, "CLAUDE.md": true,
		"errors/errors.go":                                             true,
		"service/" + entitySnake + "_types.go":                         true,
		"service/" + entitySnake + "_service.go":                       true,
		"service/" + entitySnake + "_service_test.go":                  true,
		"provider/do/" + entitySnake + "_provider.go":                  true,
		"uploads/README.md":                                            true,
		"port/port.go":                                                 true,
		"event/event.go":                                               true,
		"policy/policy.go":                                             true,
		"assembler/assembler.go":                                       true,
		"contract/contract.md":                                         true,
		"model/" + entitySnake + ".go":                                 true,
		"repository/" + entitySnake + "_repository.go":                 true,
		"repository/" + entitySnake + "_repository_mysql.go":           true,
		"cache/AGENTS.md":                                              true,
		"cache/" + entitySnake + "_cache.go":                           true,
		"cache/redis_" + entitySnake + "_cache.go":                     true,
		"cache/" + entitySnake + "_rows_cache.go":                      true,
		"read/model/" + entitySnake + "_read_list_item.go":             true,
		"read/model/" + entitySnake + "_read_detail_item.go":           true,
		"read/repository/" + entitySnake + "_read_repository.go":       true,
		"read/repository/" + entitySnake + "_read_repository_mysql.go": true,
		"read/service/" + entitySnake + "_read_query_types.go":         true,
		"read/service/" + entitySnake + "_read_query_service.go":       true,
		"read/claude.md":                                               true,
	}
	return files
}

func TestDomainFilePlan_TableDomainMatchesBaseline(t *testing.T) {
	got := domainFilePlan(false, "favorite")
	if len(got) != len(goldenTableDomainFiles("favorite")) {
		t.Fatalf("table domain plan size = %d, want %d", len(got), len(goldenTableDomainFiles("favorite")))
	}
	seen := map[string]bool{}
	for _, f := range got {
		if !goldenTableDomainFiles("favorite")[f.out] {
			t.Fatalf("unexpected output %q in table domain plan", f.out)
		}
		seen[f.out] = true
	}
	for out := range goldenTableDomainFiles("favorite") {
		if !seen[out] {
			t.Fatalf("missing output %q in table domain plan", out)
		}
	}
}

func TestDomainFilePlan_IncludesUploadsSlotForPureAndTable(t *testing.T) {
	for _, pure := range []bool{true, false} {
		plan := domainFilePlan(pure, "favorite")
		found := false
		for _, f := range plan {
			if f.out == "uploads/README.md" {
				found = true
			}
			if strings.HasPrefix(f.out, "uploads/") && f.out != "uploads/README.md" {
				t.Fatalf("pure=%v: unexpected uploads artifact %q (slot is README-only, no empty Go files)", pure, f.out)
			}
		}
		if !found {
			t.Fatalf("pure=%v: uploads/README.md missing from plan", pure)
		}
	}
}

func TestDomainFilePlan_PureTrimsTableSkeletonAndCruft(t *testing.T) {
	got := domainFilePlan(true, "favorite")
	for _, f := range got {
		for _, banned := range []string{"model/", "repository/", "cache/", "read/", "port/", "event/", "policy/", "assembler/", "contract/"} {
			if strings.HasPrefix(f.out, banned) {
				t.Fatalf("pure plan contains %q (banned prefix %q)", f.out, banned)
			}
		}
	}
}

func TestRenderDomainFiles_RendersDomainKeyAndIsDeterministic(t *testing.T) {
	data := map[string]any{
		"DomainKey":     "favorites",
		"DomainSnake":   "favorites",
		"DomainPascal":  "Favorites",
		"EntityName":    "favorite",
		"EntityPascal":  "Favorite",
		"EntitySnake":   "favorite",
		"TableName":     "favorites",
		"DomainImport":  "github.com/KOMKZ/hrise-server-app/domains/favorites",
		"RootModule":    "github.com/KOMKZ/hrise-server-app",
		"ErrModule":     128,
		"FrameworkPath": "github.com/KOMKZ/go-yogan-framework",
		"Pure":          false,
	}

	render := func() map[string]string {
		dir := t.TempDir()
		out := map[string]string{}
		for _, f := range domainFilePlan(false, "favorite") {
			outPath := filepath.Join(dir, f.out)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				t.Fatalf("mkdir for %s: %v", f.out, err)
			}
			if err := renderDomainFile(outPath, f.tmpl, data); err != nil {
				t.Fatalf("render %s: %v", f.tmpl, err)
			}
			content, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read %s: %v", f.out, err)
			}
			out[f.out] = string(content)
		}
		return out
	}

	first := render()
	second := render()
	for name, content := range first {
		if first[name] != second[name] {
			t.Fatalf("output %s is not deterministic between renders", name)
		}
		if strings.Contains(content, "<no value>") || strings.Contains(content, "{{.") {
			t.Fatalf("output %s contains unrendered template syntax", name)
		}
	}

	// uploads/README.md golden 断言：以 domain key 渲染治理入口，不绑定任何业务特例。
	readme := first["uploads/README.md"]
	if !strings.Contains(readme, "# favorites uploads") {
		t.Fatalf("uploads README header not rendered with domain key: %.80q", readme)
	}
	if strings.Contains(readme, "content_creation") || strings.Contains(readme, "favorite_") {
		t.Fatalf("uploads README must stay generic, got business-specific content")
	}

	// CLAUDE.md golden 断言：目录表声明 uploads 槽位边界。
	if !strings.Contains(first["CLAUDE.md"], "uploads/") {
		t.Fatalf("CLAUDE.md does not document the uploads slot")
	}
}

func TestValidateDomainKey(t *testing.T) {
	valid := []string{"favorites", "content-creation-v3", "user-mini-orchestration", "a1-b2"}
	for _, key := range valid {
		if err := validateDomainKey(key); err != nil {
			t.Fatalf("validateDomainKey(%q) = %v, want nil", key, err)
		}
	}
	invalid := []string{"", "Favorites", "favorite_x", "1favorite", "-favorite", "favorite--x", "收藏"}
	for _, key := range invalid {
		if err := validateDomainKey(key); err == nil {
			t.Fatalf("validateDomainKey(%q) = nil, want error", key)
		}
	}
}

func TestGenerate_RejectsExistingDomainDirBeforeToolchain(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "go.work"), []byte("go 1.24.1\n\nuse (\n)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(ws, "domains", "favorites")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}

	gen := NewDomainGenerator(&DomainConfig{DomainKey: "favorites", WorkspacePath: ws})
	_, err := gen.Generate()
	if err == nil {
		t.Fatal("Generate() = nil error, want ErrPathExists for existing domain dir")
	}
	if !errors.Is(err, ErrPathExists) {
		t.Fatalf("Generate() err = %v, want ErrPathExists", err)
	}
}

func TestGenerate_RejectsInvalidDomainKeyBeforeToolchain(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "go.work"), []byte("go 1.24.1\n\nuse (\n)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gen := NewDomainGenerator(&DomainConfig{DomainKey: "Favorites_X", WorkspacePath: ws})
	_, err := gen.Generate()
	if err == nil || !strings.Contains(err.Error(), "invalid domain name") {
		t.Fatalf("Generate() err = %v, want invalid domain name error", err)
	}
}

// registerGoWork 的 golden 回归：块形式追加、单行 use 形式（go work sync 产物）转换、重复执行幂等。
// go work sync 要求 use 条目真实存在，桩目录补最小 go.mod。
func TestRegisterGoWork(t *testing.T) {
	writeGoWork := func(t *testing.T, content string, modules ...string) string {
		t.Helper()
		ws := t.TempDir()
		if err := os.WriteFile(filepath.Join(ws, "go.work"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		for _, mod := range modules {
			dir := filepath.Join(ws, mod)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			stub := "module example.com/" + filepath.Base(mod) + "\n\ngo 1.24.1\n"
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(stub), 0644); err != nil {
				t.Fatal(err)
			}
		}
		return ws
	}

	t.Run("appends to use block", func(t *testing.T) {
		ws := writeGoWork(t, "go 1.24.1\n\nuse (\n\t./domains/alpha\n)\n", "domains/alpha", "domains/favorites")
		if err := registerGoWork(ws, "favorites"); err != nil {
			t.Fatalf("registerGoWork() err = %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(ws, "go.work"))
		text := string(got)
		if !strings.Contains(text, "\t./domains/alpha\n") || !strings.Contains(text, "\t./domains/favorites\n") {
			t.Fatalf("use block missing entries: %q", text)
		}
	})

	t.Run("converts single-line use form", func(t *testing.T) {
		ws := writeGoWork(t, "go 1.24.1\n\nuse ./domains/alpha\n", "domains/alpha", "domains/favorites")
		if err := registerGoWork(ws, "favorites"); err != nil {
			t.Fatalf("registerGoWork() err = %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(ws, "go.work"))
		text := string(got)
		if !strings.Contains(text, "use (") || !strings.Contains(text, "\t./domains/alpha\n") || !strings.Contains(text, "\t./domains/favorites\n") {
			t.Fatalf("single-line use form not converted to block: %q", text)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		ws := writeGoWork(t, "go 1.24.1\n\nuse (\n\t./domains/favorites\n)\n", "domains/favorites")
		if err := registerGoWork(ws, "favorites"); err != nil {
			t.Fatalf("registerGoWork() err = %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(ws, "go.work"))
		if strings.Count(string(got), "./domains/favorites") != 1 {
			t.Fatalf("duplicate use entry after idempotent run: %q", string(got))
		}
	})
}
