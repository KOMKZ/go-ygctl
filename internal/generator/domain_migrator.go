package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DomainMigrateConfig holds the configuration for migrating a legacy domain.
type DomainMigrateConfig struct {
	// WorkspacePath is the workspace root (empty = search upward from cwd).
	WorkspacePath string
	// DomainKey is the target domain name, e.g. "auth".
	DomainKey string
	// SourcePath is the legacy domain directory (e.g. ../yogan-domains/go-yogan-domain-auth).
	SourcePath string
	// AppName is the app receiving the centralized migrations (empty = auto-detect).
	AppName string
	// EntityName passes through to domain init.
	EntityName string
}

// DomainMigrateResult reports what the migration did.
type DomainMigrateResult struct {
	DomainDir        string
	Pure             bool
	CopiedFiles      int
	ReplacedFiles    int
	MigratedSQLFiles []string // new app-side migration filenames
	ErrModule        int
}

var oldSQLFileRe = regexp.MustCompile(`^\d{14}_(.+)\.(up|down)\.sql$`)

var packageDeclRe = regexp.MustCompile(`(?m)^package\s+\w+`)

// DomainMigrator migrates a legacy domain (old go-yogan-domain-* layout) into
// the workspace domains/ with the new structure: module prefix replaced,
// migrations centralized in the app, error module number conflict-resolved.
type DomainMigrator struct {
	config *DomainMigrateConfig
}

// NewDomainMigrator creates a new domain migrator.
func NewDomainMigrator(config *DomainMigrateConfig) *DomainMigrator {
	return &DomainMigrator{config: config}
}

// Migrate runs the full migration flow.
func (m *DomainMigrator) Migrate() (*DomainMigrateResult, error) {
	cfg := m.config
	if cfg.DomainKey == "" || cfg.SourcePath == "" {
		return nil, fmt.Errorf("both <name> and --source are required")
	}

	source, err := filepath.Abs(cfg.SourcePath)
	if err != nil {
		return nil, err
	}
	sourceMod, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("source domain not found: %w", err)
	}
	if !sourceMod.IsDir() {
		return nil, fmt.Errorf("source %s is not a directory", source)
	}

	oldModule, err := readModulePath(source)
	if err != nil {
		return nil, err
	}

	// Detect pure (table-less) domain: no model/ and no repository/ dirs.
	pure := !dirExists(filepath.Join(source, "model")) && !dirExists(filepath.Join(source, "repository"))

	// 1. Generate the target skeleton.
	initGen := NewDomainGenerator(&DomainConfig{
		DomainKey:     cfg.DomainKey,
		EntityName:    cfg.EntityName,
		WorkspacePath: cfg.WorkspacePath,
		Pure:          pure,
	})
	info, err := initGen.Generate()
	if err != nil {
		return nil, err
	}

	result := &DomainMigrateResult{DomainDir: info.DomainDir, Pure: pure, ErrModule: info.ErrModule}

	// 2. Copy legacy files (skip module/legacy/migration artifacts; keep generated CLAUDE.md).
	skipNames := map[string]bool{
		"go.mod": true, "go.sum": true, ".git": true, ".gitignore": true,
		"migrations": true, "logs": true, "CLAUDE.md": true,
	}
	if err := filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if skipNames[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".out") {
			return nil
		}

		dest := filepath.Join(info.DomainDir, rel)
		// Legacy domains keep code in the domain root package; the new rule is
		// "no code at domain root" — root-level .go files move to port/.
		movedToPort := !d.IsDir() && !strings.Contains(rel, string(filepath.Separator)) && strings.HasSuffix(rel, ".go")
		if movedToPort {
			dest = filepath.Join(info.DomainDir, "port", rel)
		}
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if isTextFile(d.Name()) && oldModule != "" {
			replaced := strings.ReplaceAll(string(content), oldModule, info.DomainImport)
			if replaced != string(content) {
				result.ReplacedFiles++
			}
			content = []byte(replaced)
			// Legacy domains import their own root package (where interfaces
			// lived); the new structure keeps those in port/.
			rootImportRe := regexp.MustCompile(`(\w+\s+)?"` + regexp.QuoteMeta(info.DomainImport) + `"`)
			content = rootImportRe.ReplaceAll(content, []byte(`${1}"`+info.DomainImport+`/port"`))
		}
		if movedToPort {
			// Rewrite the legacy root package declaration to the port package.
			content = packageDeclRe.ReplaceAll(content, []byte("package port"))
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return err
		}
		result.CopiedFiles++
		return nil
	}); err != nil {
		return nil, fmt.Errorf("copy failed: %w", err)
	}

	// 3. Remove skeleton placeholders superseded by migrated code
	// (legacy files may have different names, e.g. auth_errors.go vs errors.go).
	removeSkeletonPlaceholders(info.DomainDir)

	// 4. Resolve error module number conflicts against the new workspace.
	finalModule, err := m.resolveErrModule(info.DomainDir, info.ErrModule)
	if err != nil {
		return nil, err
	}
	result.ErrModule = finalModule
	if err := syncCLAUDEMModule(info.DomainDir, finalModule); err != nil {
		return nil, err
	}

	// 5. Centralize legacy migrations into the app.
	if err := m.centralizeMigrations(source, result); err != nil {
		return nil, err
	}

	// 6. gofmt + tidy + build + test verification.
	gofmtArgs := []string{"-w"}
	_ = filepath.Walk(info.DomainDir, func(path string, d os.FileInfo, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".go") {
			gofmtArgs = append(gofmtArgs, path)
		}
		return nil
	})
	if out, err := exec.Command("gofmt", gofmtArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = info.DomainDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %v\n%s", err, out)
	}

	workspace := initGen.config.WorkspacePath
	buildCmd := exec.Command("go", "build", "./domains/"+cfg.DomainKey+"/...")
	buildCmd.Dir = workspace
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go build failed: %v\n%s", err, out)
	}

	testCmd := exec.Command("go", "test", "./domains/"+cfg.DomainKey+"/...")
	testCmd.Dir = workspace
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		return nil, fmt.Errorf("domain tests failed; fix the migrated code before delivering")
	}

	return result, nil
}

// readModulePath parses the module line of <dir>/go.mod.
func readModulePath(dir string) (string, error) {
	content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("source has no go.mod: %w", err)
	}
	m := moduleLineRe.FindSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("cannot parse module path from %s/go.mod", dir)
	}
	return string(m[1]), nil
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isTextFile(name string) bool {
	return strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".md")
}

// resolveErrModule keeps the legacy module number unless it collides with an
// already-allocated number in the new workspace, in which case the number
// allocated by domain init is used instead. Returns the final module number.
func (m *DomainMigrator) resolveErrModule(domainDir string, allocated int) (int, error) {
	// Collect numbers in use by OTHER domains of the new workspace.
	inUse := map[int]bool{}
	otherRoot := filepath.Join(filepath.Dir(filepath.Dir(domainDir)), "*", "errors", "*.go")
	others, _ := filepath.Glob(otherRoot)
	for _, f := range others {
		if strings.HasPrefix(f, domainDir) {
			continue
		}
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, mm := range errModuleRe.FindAllSubmatch(content, -1) {
			if n, err := strconv.Atoi(string(mm[1])); err == nil {
				inUse[n] = true
			}
		}
	}

	moduleNameRe := regexp.MustCompile(`Module\w+`)
	matches, err := filepath.Glob(filepath.Join(domainDir, "errors", "*.go"))
	if err != nil {
		return 0, err
	}

	finalModule := allocated
	for _, f := range matches {
		content, err := os.ReadFile(f)
		if err != nil {
			return 0, err
		}
		text := string(content)
		for _, mm := range errModuleRe.FindAllStringSubmatch(text, -1) {
			old, err := strconv.Atoi(mm[1])
			if err != nil {
				continue
			}
			if inUse[old] {
				name := moduleNameRe.FindString(mm[0])
				replacement := fmt.Sprintf("const %s = %d", name, allocated)
				text = strings.Replace(text, mm[0], replacement, 1)
			} else {
				finalModule = old // legacy number is free, keep it
			}
		}
		if text != string(content) {
			if err := os.WriteFile(f, []byte(text), 0644); err != nil {
				return 0, err
			}
		}
	}
	return finalModule, nil
}

// syncCLAUDEMModule rewrites the module number line in the domain's CLAUDE.md.
func syncCLAUDEMModule(domainDir string, module int) error {
	claudePath := filepath.Join(domainDir, "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	if err != nil {
		return nil // CLAUDE.md is optional
	}
	re := regexp.MustCompile(`错误码模块号：\*\*\d+\*\*`)
	updated := re.ReplaceAllString(string(content), fmt.Sprintf("错误码模块号：**%d**", module))
	if updated != string(content) {
		return os.WriteFile(claudePath, []byte(updated), 0644)
	}
	return nil
}

// removeSkeletonPlaceholders deletes skeleton placeholder files when migrated
// code provides the real content (possibly under different file names).
func removeSkeletonPlaceholders(domainDir string) {
	placeholders := []string{
		"errors/errors.go",
		"model/model.go",
		"repository/repository.go",
		"repository/repository_mysql.go",
		"service/service.go",
		"service/service_test.go",
		"provider/do/provider.go",
		"port/port.go",
	}
	for _, p := range placeholders {
		dir := filepath.Join(domainDir, filepath.Dir(p))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		goCount := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
				goCount++
			}
		}
		if goCount >= 2 {
			_ = os.Remove(filepath.Join(domainDir, p))
		}
	}
}

// centralizeMigrations copies legacy domain migrations into
// apps/<app>/migrations/ with the new NNNNNN_<domain>_<desc> naming.
func (m *DomainMigrator) centralizeMigrations(source string, result *DomainMigrateResult) error {
	legacyDir := filepath.Join(source, "migrations")
	if !dirExists(legacyDir) {
		return nil
	}

	cfg := &MigrateConfig{
		WorkspacePath: m.config.WorkspacePath,
		AppName:       m.config.AppName,
		DomainKey:     m.config.DomainKey,
	}
	_, _, migrationsDir, err := cfg.resolve()
	if err != nil {
		return err
	}
	seq, err := NextMigrationSeq(migrationsDir)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return err
	}
	var upFiles []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && oldSQLFileRe.MatchString(name) && strings.HasSuffix(name, ".up.sql") {
			upFiles = append(upFiles, name)
		}
	}
	sort.Strings(upFiles)

	for _, up := range upFiles {
		mm := oldSQLFileRe.FindStringSubmatch(up)
		if mm == nil {
			continue
		}
		desc := mm[1]
		down := strings.TrimSuffix(up, ".up.sql") + ".down.sql"

		// Idempotent re-runs: skip if this domain+desc pair is already centralized.
		existing, _ := filepath.Glob(filepath.Join(migrationsDir, fmt.Sprintf("[0-9][0-9][0-9][0-9][0-9][0-9]_%s_%s.up.sql", m.config.DomainKey, desc)))
		if len(existing) > 0 {
			continue
		}

		upContent, err := os.ReadFile(filepath.Join(legacyDir, up))
		if err != nil {
			return err
		}
		downContent, err := os.ReadFile(filepath.Join(legacyDir, down))
		if err != nil {
			return fmt.Errorf("missing down migration for %s: %w", up, err)
		}

		base := fmt.Sprintf("%06d_%s_%s", seq, m.config.DomainKey, desc)
		if err := os.WriteFile(filepath.Join(migrationsDir, base+".up.sql"), upContent, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(migrationsDir, base+".down.sql"), downContent, 0644); err != nil {
			return err
		}
		result.MigratedSQLFiles = append(result.MigratedSQLFiles, base+".up.sql", base+".down.sql")
		seq++
	}
	return nil
}
