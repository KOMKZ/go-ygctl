package generator

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed all:templates/api
var apiTemplates embed.FS

// APIGenConfig holds the configuration for API layer generation.
type APIGenConfig struct {
	WorkspacePath string
	DefFile       string
}

// APIGenResult reports the API generation result.
type APIGenResult struct {
	DomainDir     string
	ModuleDir     string
	MigrationPath string // empty when a migration already existed
}

// Generate runs the API layer generation from the def (on top of the DAO layer).
func (c *APIGenConfig) Generate() (*APIGenResult, error) {
	def, err := LoadDef(c.DefFile)
	if err != nil {
		return nil, err
	}
	workspace, err := resolveWorkspacePath(c.WorkspacePath)
	if err != nil {
		return nil, err
	}

	domainDir := filepath.Join(workspace, "domains", def.Domain)
	if _, err := os.Stat(filepath.Join(domainDir, "go.mod")); err != nil {
		return nil, fmt.Errorf("domain %q does not exist; run 'ygctl dao gen -f %s' first", def.Domain, c.DefFile)
	}

	// Resolve the target app (required for module/router wiring and migrations).
	migCfg := &MigrateConfig{WorkspacePath: workspace}
	_, appDir, migrationsDir, err := migCfg.resolve()
	if err != nil {
		return nil, err
	}
	appModule, err := readModulePath(appDir)
	if err != nil {
		return nil, err
	}
	appName := filepath.Base(appDir)

	data, err := buildDefTemplateData(workspace, domainDir, def)
	if err != nil {
		return nil, err
	}
	data.AppModule = appModule
	data.AppName = appName

	result := &APIGenResult{DomainDir: domainDir, ModuleDir: filepath.Join(appDir, "internal", "module", data.AppModuleSnake)}

	// ---- Domain-side files ----
	domainFiles := []struct {
		tmpl string
		out  string
	}{
		{"service/types.go.tmpl", "service/" + data.EntitySnake + "_types.go"},
		{"service/service.go.tmpl", "service/" + data.EntitySnake + "_service.go"},
		{"service/service_test.go.tmpl", "service/" + data.EntitySnake + "_service_test.go"},
		{"provider/do/provider.go.tmpl", "provider/do/provider.go"},
		{"contract/contract.md.tmpl", "contract/contract.md"},
	}
	for _, f := range domainFiles {
		outPath := filepath.Join(domainDir, f.out)
		if err := renderDefTemplate(apiTemplates, "templates/api", outPath, f.tmpl, data); err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", f.out, err)
		}
	}
	warnings, err := GeneratePermissionDeclarations(workspace, def.Domain, apiTemplates, "templates/api")
	if err != nil {
		return nil, fmt.Errorf("failed to generate permissions: %w", err)
	}
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
	}

	// The domain-init skeleton service files are superseded by the generated
	// <entity>_service.go; remove only those two (do NOT touch dao gen outputs).
	_ = os.Remove(filepath.Join(domainDir, "service", "service.go"))
	_ = os.Remove(filepath.Join(domainDir, "service", "service_test.go"))

	// ---- App-side files ----
	appFiles := []struct {
		tmpl string
		out  string
	}{
		{"appmodule/handler.go.tmpl", "internal/module/" + data.AppModuleSnake + "/" + data.AppFilePrefix + "_handler.go"},
		{"appmodule/dto.go.tmpl", "internal/module/" + data.AppModuleSnake + "/" + data.AppFilePrefix + "_dto.go"},
		{"appmodule/provider.go.tmpl", "internal/module/" + data.AppModuleSnake + "/" + data.AppFilePrefix + "_provider.go"},
		{"approuter/router.go.tmpl", "internal/router/" + data.EntitySnake + "_router.go"},
	}
	for _, f := range appFiles {
		outPath := filepath.Join(appDir, f.out)
		if err := renderDefTemplate(apiTemplates, "templates/api", outPath, f.tmpl, data); err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", f.out, err)
		}
	}

	// ---- App wiring (idempotent) ----
	if err := wireCallbacks(appDir, data); err != nil {
		return nil, err
	}
	if err := wireRouter(appDir, data); err != nil {
		return nil, err
	}

	// ---- Migration SQL skeleton (generated, NOT applied) ----
	migrationPath, err := generateMigrationSkeleton(migrationsDir, data)
	if err != nil {
		return nil, err
	}
	result.MigrationPath = migrationPath

	// ---- Verify: gofmt + tidy (domain and app) + build ----
	gofmtDirs := []string{
		filepath.Join(domainDir, "service"),
		filepath.Join(domainDir, "provider"),
		filepath.Join(domainDir, "permissions"),
		filepath.Join(appDir, "internal", "module", data.AppModuleSnake),
		filepath.Join(appDir, "internal", "router"),
		filepath.Join(appDir, "internal", "app"),
	}
	gofmtArgs := []string{"-w"}
	for _, d := range gofmtDirs {
		_ = filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(path, ".go") {
				gofmtArgs = append(gofmtArgs, path)
			}
			return nil
		})
	}
	if out, err := exec.Command("gofmt", gofmtArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}

	// Only the domain module gets a tidy pass. The app module must NOT be
	// tidied in workspace mode: go mod tidy cannot resolve workspace member
	// modules (Go 1.24 behavior), and a require for them breaks go build too.
	// Cross-member imports resolve via go.work at build time; the build below
	// is the verification.
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = domainDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed in %s: %v\n%s", domainDir, err, out)
	}

	// External deps introduced by generated app code (ozzo-validation for
	// request DTOs) are added with go get — the app module cannot be tidied.
	if data.HasValidate {
		getCmd := exec.Command("go", "get", "github.com/go-ozzo/ozzo-validation/v4")
		getCmd.Dir = appDir
		if out, err := getCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go get ozzo-validation failed: %v\n%s", err, out)
		}
		syncCmd := exec.Command("go", "work", "sync")
		syncCmd.Dir = workspace
		if out, err := syncCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("go work sync failed: %v\n%s", err, out)
		}
	}

	buildTarget := "./domains/" + def.Domain + "/... ./apps/" + appName + "/..."
	buildCmd := exec.Command("go", append([]string{"build"}, strings.Fields(buildTarget)...)...)
	buildCmd.Dir = workspace
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go build failed: %v\n%s", err, out)
	}

	return result, nil
}

// wireCallbacks inserts the module DI registration into internal/app/callbacks.go (idempotent).
func wireCallbacks(appDir string, data *defTemplateData) error {
	path := filepath.Join(appDir, "internal", "app", "callbacks.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read callbacks.go: %w", err)
	}
	text := string(content)

	modImport := fmt.Sprintf("\t%sModule \"%s/internal/module/%s\"\n", data.AppModuleSnake, data.AppModule, data.AppModuleSnake)
	doImport := fmt.Sprintf("\t%sDo \"%s/provider/do\"\n", data.DomainSnake, data.DomainImport)
	provideLine := fmt.Sprintf("\tdo.Provide(injector, %sModule.Provide%sHandler)\n", data.AppModuleSnake, data.EntityPascal)
	domainProvideRepo := fmt.Sprintf("\tdo.Provide(injector, %sDo.Provide%sRepository)\n", data.DomainSnake, data.EntityPascal)
	domainProvideSvc := fmt.Sprintf("\tdo.Provide(injector, %sDo.Provide%sService)\n", data.DomainSnake, data.EntityPascal)

	if strings.Contains(text, "Provide"+data.EntityPascal+"Handler") {
		return nil // already wired
	}

	// Insert imports after the last existing import line.
	lines := strings.Split(text, "\n")
	lastImport := -1
	for i, ln := range lines {
		if strings.HasPrefix(ln, "\t") && strings.Contains(ln, `"`) {
			lastImport = i
		}
		if strings.HasPrefix(ln, ")") && lastImport > 0 {
			break
		}
	}
	if lastImport < 0 {
		return fmt.Errorf("cannot locate import block in callbacks.go")
	}
	insert := []string{}
	if !strings.Contains(text, `internal/module/`+data.AppModuleSnake+`"`) {
		insert = append(insert, modImport)
	}
	if !strings.Contains(text, `"`+data.DomainImport+`/provider/do"`) {
		insert = append(insert, doImport)
	}
	lines = append(lines[:lastImport+1], append(insert, lines[lastImport+1:]...)...)
	text = strings.Join(lines, "\n")

	// Insert do.Provide calls after the last existing do.Provide line
	// (inside the DI registration function, not at end of file).
	anchor := strings.LastIndex(text, "do.Provide(injector, ")
	if anchor < 0 {
		return fmt.Errorf("cannot locate a do.Provide anchor in callbacks.go")
	}
	lineEnd := strings.Index(text[anchor:], "\n")
	if lineEnd < 0 {
		return fmt.Errorf("cannot locate line end in callbacks.go")
	}
	insertPos := anchor + lineEnd + 1
	provisions := domainProvideRepo + domainProvideSvc + provideLine
	text = text[:insertPos] + provisions + text[insertPos:]

	return os.WriteFile(path, []byte(text), 0644)
}

// wireRouter inserts the route registration call into internal/app/router.go (idempotent).
func wireRouter(appDir string, data *defTemplateData) error {
	path := filepath.Join(appDir, "internal", "app", "router.go")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read router.go: %w", err)
	}
	text := string(content)

	call := fmt.Sprintf("router.Register%sRoutes(engine, app)", data.EntityPascal)
	if strings.Contains(text, call) {
		return nil
	}

	line := fmt.Sprintf("\t%s\n", call)
	closeIdx := strings.LastIndex(text, "\n}")
	if closeIdx < 0 {
		return fmt.Errorf("cannot locate function end in router.go")
	}
	text = text[:closeIdx] + "\n" + line + "\n}"
	return os.WriteFile(path, []byte(text), 0644)
}

var migrationFileRe = regexp.MustCompile(`^\d{6}_.+\.up\.sql$`)

// generateMigrationSkeleton writes NNNNNN_<domain>_create_<table>.up/down.sql
// (idempotent: skipped when the domain+desc pair already exists).
func generateMigrationSkeleton(migrationsDir string, data *defTemplateData) (string, error) {
	existing, _ := filepath.Glob(filepath.Join(migrationsDir, fmt.Sprintf("[0-9][0-9][0-9][0-9][0-9][0-9]_%s_create_%s.up.sql", data.DomainKey, data.TableName)))
	if len(existing) > 0 {
		return "", nil // already generated
	}

	seq, err := NextMigrationSeq(migrationsDir)
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf("%06d_%s_create_%s", seq, data.DomainKey, data.TableName)

	var up strings.Builder
	up.WriteString(fmt.Sprintf("-- 创建 %s 表\n-- 域：%s\n-- 生成：ygctl api gen（骨架，人工核对后执行 ygctl migrate up）\n\n", data.TableName, data.DomainKey))
	up.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", data.TableName))
	up.WriteString("    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n")
	for _, f := range data.Fields {
		line := fmt.Sprintf("    `%s` %s", f.Name, f.MySQLCol)
		if f.Unique || f.Required {
			line += " NOT NULL"
		}
		if f.Default != "" {
			line += " DEFAULT " + mysqlDefaultLiteral(f)
		}
		if f.Comment != "" {
			line += fmt.Sprintf(" COMMENT '%s'", f.Comment)
		}
		up.WriteString(line + ",\n")
	}
	up.WriteString("    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,\n")
	up.WriteString("    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n")
	up.WriteString("    PRIMARY KEY (`id`)\n")
	for _, f := range data.Fields {
		if f.Unique {
			up.WriteString(fmt.Sprintf("    ,UNIQUE KEY `uk_%s` (`%s`)\n", f.Name, f.Name))
		} else if f.Index {
			up.WriteString(fmt.Sprintf("    ,KEY `idx_%s` (`%s`)\n", f.Name, f.Name))
		}
	}
	up.WriteString(fmt.Sprintf(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='%s';\n", data.TableName))

	down := fmt.Sprintf("-- 回滚：删除 %s 表\n-- 域：%s\n\nDROP TABLE IF EXISTS `%s`;\n", data.TableName, data.DomainKey, data.TableName)

	upPath := filepath.Join(migrationsDir, base+".up.sql")
	downPath := filepath.Join(migrationsDir, base+".down.sql")
	if err := os.WriteFile(upPath, []byte(up.String()), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(downPath, []byte(down), 0644); err != nil {
		return "", err
	}
	return upPath, nil
}

func mysqlDefaultLiteral(f defFieldData) string {
	switch f.GoType {
	case "string":
		return "'" + strings.ReplaceAll(f.Default, "'", "''") + "'"
	default:
		return f.Default
	}
}
