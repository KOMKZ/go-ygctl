package generator

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed all:templates/domain
var domainTemplates embed.FS

// DomainConfig holds the configuration for generating a domain package.
type DomainConfig struct {
	// DomainKey is the domain directory name (snake/kebab-case), e.g. "auth".
	DomainKey string
	// EntityName is the primary entity (optional; defaults to singularized key).
	EntityName string
	// WorkspacePath is the workspace root containing go.work.
	// Empty means search upward from the current directory.
	WorkspacePath string
	// ModuleBase is the module path base, e.g. "github.com/KOMKZ/hrise-server-app".
	// The domain module becomes <base>/domains/<key>.
	// Empty means derive from the first app's go.mod under apps/.
	ModuleBase string
	// Pure marks a table-less logic domain: model/ and repository/ are skipped
	// and the service/provider carry no repository dependency.
	Pure bool
}

// DomainInfo reports the result of a domain generation.
type DomainInfo struct {
	DomainDir    string
	DomainImport string
	ErrModule    int
}

// DomainGenerator generates a full domain skeleton under <workspace>/domains/<key>.
// Skeleton only — no business logic is generated.
type DomainGenerator struct {
	config     *DomainConfig
	rootModule string
	errModule  int
}

// NewDomainGenerator creates a new domain generator.
func NewDomainGenerator(config *DomainConfig) *DomainGenerator {
	return &DomainGenerator{config: config}
}

var (
	domainKeyRe  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	moduleLineRe = regexp.MustCompile(`^module\s+(\S+)`)
	errModuleRe  = regexp.MustCompile(`const\s+Module\w+\s*=\s*(\d+)`)
	singleUseRe  = regexp.MustCompile(`^\s*use\s+(\S+)\s*$`)
)

// domainFile 描述一个领域骨架输出项：模板名 + 相对领域根目录的输出路径。
type domainFile struct {
	tmpl string
	out  string
}

// validateDomainKey 校验领域名：小写字母开头，仅小写字母/数字/连字符（kebab-case）。
func validateDomainKey(key string) error {
	if !domainKeyRe.MatchString(key) {
		return fmt.Errorf("invalid domain name %q: use lowercase kebab-case (a-z, 0-9, -)", key)
	}
	return nil
}

// domainFilePlan 输出领域骨架文件计划（模板名 → 输出路径）。
// 骨架矩阵来源：hrise-server-app 现行领域治理 + 金样域（resource-tags 表域 /
// session-management、user-mini-orchestration、admin-orchestration 纯域）：
//   - uploads/README.md 是所有业务领域的上传能力槽位（根 AGENTS.md 媒体资源治理），
//     纯逻辑域与表域一致保留；无用例时只渲染 README，不生成空 Go 文件；
//   - 纯逻辑域（--pure）裁剪 port/event/policy/assembler/contract 与表骨架，
//     与金样纯域（errors/service/provider/do/uploads）基线一致；
//   - 表域额外生成 model/repository/cache/read 读写真骨架。
func domainFilePlan(pure bool, entitySnake string) []domainFile {
	files := []domainFile{
		{"go.mod.tmpl", "go.mod"},
		{".gitignore.tmpl", ".gitignore"},
		{"CLAUDE.md.tmpl", "CLAUDE.md"},
		{"errors/errors.go.tmpl", "errors/errors.go"},
		{"service/types.go.tmpl", "service/" + entitySnake + "_types.go"},
		{"service/service.go.tmpl", "service/" + entitySnake + "_service.go"},
		{"service/service_test.go.tmpl", "service/" + entitySnake + "_service_test.go"},
		{"provider/do/provider.go.tmpl", "provider/do/" + entitySnake + "_provider.go"},
		{"uploads/README.md.tmpl", "uploads/README.md"},
	}
	if pure {
		return files
	}
	files = append(files,
		domainFile{"port/port.go.tmpl", "port/port.go"},
		domainFile{"event/event.go.tmpl", "event/event.go"},
		domainFile{"policy/policy.go.tmpl", "policy/policy.go"},
		domainFile{"assembler/assembler.go.tmpl", "assembler/assembler.go"},
		domainFile{"contract/contract.md.tmpl", "contract/contract.md"},
		domainFile{"model/model.go.tmpl", "model/" + entitySnake + ".go"},
		domainFile{"repository/repository.go.tmpl", "repository/" + entitySnake + "_repository.go"},
		domainFile{"repository/repository_mysql.go.tmpl", "repository/" + entitySnake + "_repository_mysql.go"},
		domainFile{"cache/AGENTS.md.tmpl", "cache/AGENTS.md"},
		domainFile{"cache/domain_cache.go.tmpl", "cache/" + entitySnake + "_cache.go"},
		domainFile{"cache/redis_domain_cache.go.tmpl", "cache/redis_" + entitySnake + "_cache.go"},
		domainFile{"cache/entity_rows_cache.go.tmpl", "cache/" + entitySnake + "_rows_cache.go"},
		domainFile{"read/model/read_list_item.go.tmpl", "read/model/" + entitySnake + "_read_list_item.go"},
		domainFile{"read/model/read_detail_item.go.tmpl", "read/model/" + entitySnake + "_read_detail_item.go"},
		domainFile{"read/repository/read_repository.go.tmpl", "read/repository/" + entitySnake + "_read_repository.go"},
		domainFile{"read/repository/read_repository_mysql.go.tmpl", "read/repository/" + entitySnake + "_read_repository_mysql.go"},
		domainFile{"read/service/read_query_types.go.tmpl", "read/service/" + entitySnake + "_read_query_types.go"},
		domainFile{"read/service/read_query_service.go.tmpl", "read/service/" + entitySnake + "_read_query_service.go"},
		domainFile{"read/claude.md.tmpl", "read/claude.md"},
	)
	return files
}

// Generate creates the domain skeleton and verifies it compiles.
func (g *DomainGenerator) Generate() (*DomainInfo, error) {
	if err := g.resolveWorkspace(); err != nil {
		return nil, err
	}
	key := g.config.DomainKey
	if err := validateDomainKey(key); err != nil {
		return nil, err
	}

	domainsDir := filepath.Join(g.config.WorkspacePath, "domains")
	domainDir := filepath.Join(domainsDir, key)
	if _, err := os.Stat(domainDir); !os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrPathExists, domainDir)
	}
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create domain dir: %w", err)
	}

	if err := g.loadModuleBase(); err != nil {
		return nil, err
	}
	g.errModule = allocateErrModule(domainsDir)

	data := g.templateData(key)
	files := domainFilePlan(g.config.Pure, data["EntitySnake"].(string))

	var goFiles []string
	for _, f := range files {
		outPath := filepath.Join(domainDir, f.out)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return nil, err
		}
		if err := renderDomainFile(outPath, f.tmpl, data); err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", f.out, err)
		}
		if strings.HasSuffix(f.out, ".go") {
			goFiles = append(goFiles, outPath)
		}
	}

	// gofmt all generated Go files
	gofmtArgs := append([]string{"-w"}, goFiles...)
	if out, err := exec.Command("gofmt", gofmtArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}

	// Register the domain module in go.work (idempotent).
	if err := registerGoWork(g.config.WorkspacePath, key); err != nil {
		return nil, err
	}

	// Pull dependencies for the domain module.
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = domainDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %v\n%s", err, out)
	}

	// Verify the skeleton compiles from the workspace.
	buildTarget := "./domains/" + key + "/..."
	buildCmd := exec.Command("go", "build", buildTarget)
	buildCmd.Dir = g.config.WorkspacePath
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go build failed: %v\n%s", err, out)
	}

	return &DomainInfo{
		DomainDir:    domainDir,
		DomainImport: data["DomainImport"].(string),
		ErrModule:    g.errModule,
	}, nil
}

// FindWorkspaceRoot locates the workspace root (directory containing go.work)
// starting from startDir (or the current directory if startDir is empty) upward.
func FindWorkspaceRoot(startDir string) (string, error) {
	dir := startDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.work found upward from %s; run inside the workspace or pass --workspace", dir)
		}
		dir = parent
	}
}

// resolveWorkspace resolves the configured workspace root.
func (g *DomainGenerator) resolveWorkspace() error {
	if g.config.WorkspacePath != "" {
		abs, err := filepath.Abs(g.config.WorkspacePath)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(abs, "go.work")); err != nil {
			return fmt.Errorf("workspace %s has no go.work: %w", abs, err)
		}
		g.config.WorkspacePath = abs
		return nil
	}
	root, err := FindWorkspaceRoot("")
	if err != nil {
		return err
	}
	g.config.WorkspacePath = root
	return nil
}

// loadModuleBase resolves the module path base (e.g. "github.com/KOMKZ/hrise-server-app").
// The domain module becomes <base>/domains/<key>.
// Base resolution: explicit --module > first app's go.mod under apps/.
func (g *DomainGenerator) loadModuleBase() error {
	if g.config.ModuleBase != "" {
		g.rootModule = g.config.ModuleBase
		return nil
	}

	appsRoot := filepath.Join(g.config.WorkspacePath, "apps")
	entries, err := os.ReadDir(appsRoot)
	if err != nil {
		return fmt.Errorf("cannot read %s (pass --module): %w", appsRoot, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(appsRoot, e.Name(), "go.mod"))
		if err != nil {
			continue
		}
		m := moduleLineRe.FindSubmatch(content)
		if m == nil {
			continue
		}
		appModule := string(m[1])
		// Strip the trailing /apps/<name> segment(s) to get the base.
		base := strings.TrimSuffix(appModule, "/apps/"+e.Name())
		if base != appModule {
			g.rootModule = base
			return nil
		}
	}
	return fmt.Errorf("cannot derive module base from apps/*/go.mod; pass --module")
}

// registerGoWork adds "use ./domains/<key>" to go.work (idempotent).
// The closing paren is matched on its own line so comments containing ")"
// are not mistaken for the end of the use block.
// `go work sync` rewrites a single-entry use block into the single-line
// `use <path>` form; that form is converted back to a block so the next
// domain init can append its entry.
func registerGoWork(workspacePath, key string) error {
	goWorkPath := filepath.Join(workspacePath, "go.work")
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		return fmt.Errorf("failed to read go.work: %w", err)
	}

	entry := "./domains/" + key
	lines := strings.Split(string(content), "\n")
	for _, ln := range lines {
		if strings.TrimSpace(ln) == entry {
			return nil // exact line match avoids ./domains/x matching ./domains/xy
		}
	}
	useLine := -1
	inserted := false
	for i, ln := range lines {
		if strings.Contains(ln, "use (") {
			useLine = i
			break
		}
	}
	if useLine < 0 {
		// Single-line `use <path>` form (produced by `go work sync` for
		// single-module workspaces): convert it to a block listing both;
		// the new entry is already part of the block, so insertion is skipped.
		for i, ln := range lines {
			single := singleUseRe.FindStringSubmatch(ln)
			if single == nil || strings.TrimSpace(single[1]) == entry {
				continue
			}
			block := []string{"use (", "\t" + strings.TrimSpace(single[1]), "\t" + entry, ")"}
			lines = append(lines[:i], append(block, lines[i+1:]...)...)
			useLine = i
			inserted = true
			break
		}
	}
	if useLine < 0 {
		return fmt.Errorf("go.work has no 'use' block")
	}

	closeLine := -1
	for j := useLine + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == ")" {
			closeLine = j
			break
		}
	}
	if closeLine < 0 {
		return fmt.Errorf("go.work 'use' block is not closed")
	}

	if !inserted {
		lines = append(lines[:closeLine], append([]string{"\t" + entry}, lines[closeLine:]...)...)
	}
	if err := os.WriteFile(goWorkPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to update go.work: %w", err)
	}

	// Keep the workspace state (go.work.sum) in sync so subsequent go
	// commands resolve the new module immediately.
	syncCmd := exec.Command("go", "work", "sync")
	syncCmd.Dir = workspacePath
	if out, err := syncCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go work sync failed: %v\n%s", err, out)
	}
	return nil
}

// allocateErrModule scans existing domains' error modules and returns max+1.
func allocateErrModule(domainsDir string) int {
	maxModule := 0
	errorsRoot := filepath.Join(domainsDir, "*", "errors", "*.go")
	if matches, err := filepath.Glob(errorsRoot); err == nil {
		for _, f := range matches {
			content, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			for _, m := range errModuleRe.FindAllSubmatch(content, -1) {
				if n, err := parseInt(string(m[1])); err == nil && n > maxModule {
					maxModule = n
				}
			}
		}
	}
	return maxModule + 1
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func (g *DomainGenerator) templateData(key string) map[string]any {
	entity := g.config.EntityName
	if entity == "" {
		// Singularize: strip trailing "s" (same rule as legacy domain-init).
		entity = strings.TrimSuffix(key, "s")
		if entity == "" {
			entity = key
		}
	}

	domainSnake := strings.ReplaceAll(key, "-", "_")
	entityPascal := ToPascalCase(entity)
	entitySnake := ToSnakeCase(entityPascal)

	return map[string]any{
		"DomainKey":     key,
		"DomainSnake":   domainSnake,
		"DomainPascal":  ToPascalCase(key),
		"EntityName":    entity,
		"EntityPascal":  entityPascal,
		"EntitySnake":   entitySnake,
		"TableName":     entitySnake + "s",
		"DomainImport":  g.rootModule + "/domains/" + key,
		"RootModule":    g.rootModule,
		"ErrModule":     g.errModule,
		"FrameworkPath": "github.com/KOMKZ/go-yogan-framework",
		"Pure":          g.config.Pure,
	}
}

// renderDomainFile renders one embedded domain template.
func renderDomainFile(outPath, tmplName string, data map[string]any) error {
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
