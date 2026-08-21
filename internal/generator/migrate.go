package generator

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var migrateSeqRe = regexp.MustCompile(`^(\d{6})_.*\.up\.sql$`)

// MigrateConfig holds the configuration for migration management.
type MigrateConfig struct {
	// WorkspacePath is the workspace root (empty = search upward from cwd).
	WorkspacePath string
	// AppName is the target app, e.g. "hrise-admin-api".
	// Empty = auto-detect when the workspace has exactly one app with migrations/.
	AppName string
	// DomainKey prefixes the migration filename (traceability).
	DomainKey string
	// Desc is the migration description (snake_case).
	Desc string
	// TableName seeds the CREATE TABLE skeleton (optional, default "table_name").
	TableName string
}

// ResolveAppDir returns the app directory (apps/<app>) and validates its migrations dir.
func (c *MigrateConfig) resolve() (workspace, appDir, migrationsDir string, err error) {
	workspace = c.WorkspacePath
	if workspace == "" {
		workspace, err = FindWorkspaceRoot("")
		if err != nil {
			return "", "", "", err
		}
	} else {
		workspace, err = filepath.Abs(workspace)
		if err != nil {
			return "", "", "", err
		}
	}

	appName := c.AppName
	if appName == "" {
		appName, err = detectSingleApp(workspace)
		if err != nil {
			return "", "", "", err
		}
	}

	appDir = filepath.Join(workspace, "apps", appName)
	migrationsDir = filepath.Join(appDir, "migrations")
	if _, err := os.Stat(migrationsDir); err != nil {
		return "", "", "", fmt.Errorf("app %q has no migrations dir: %w", appName, err)
	}
	return workspace, appDir, migrationsDir, nil
}

// detectSingleApp returns the sole app name under apps/ that has a migrations dir.
func detectSingleApp(workspace string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(workspace, "apps"))
	if err != nil {
		return "", fmt.Errorf("cannot list apps: %w", err)
	}
	var apps []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(workspace, "apps", e.Name(), "migrations")); err == nil {
			apps = append(apps, e.Name())
		}
	}
	if len(apps) == 1 {
		return apps[0], nil
	}
	return "", fmt.Errorf("found %d apps with migrations (%v); pass --app", len(apps), apps)
}

// NextMigrationSeq scans migrations dir and returns max sequence + 1.
func NextMigrationSeq(migrationsDir string) (int, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, err
	}
	maxSeq := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrateSeqRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		seq, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}
	return maxSeq + 1, nil
}

// CreateMigration writes a NNNNNN_<domain>_<desc>.up/down.sql pair.
func (c *MigrateConfig) CreateMigration() (upPath, downPath string, err error) {
	if !domainKeyRe.MatchString(c.DomainKey) {
		return "", "", fmt.Errorf("invalid --domain %q: use lowercase snake/kebab-case", c.DomainKey)
	}
	if c.Desc == "" {
		return "", "", fmt.Errorf("description is required")
	}
	desc := strings.ToLower(strings.ReplaceAll(c.Desc, " ", "_"))
	table := c.TableName
	if table == "" {
		table = "table_name"
	}

	_, _, migrationsDir, err := c.resolve()
	if err != nil {
		return "", "", err
	}

	seq, err := NextMigrationSeq(migrationsDir)
	if err != nil {
		return "", "", err
	}

	base := fmt.Sprintf("%06d_%s_%s", seq, c.DomainKey, desc)
	upPath = filepath.Join(migrationsDir, base+".up.sql")
	downPath = filepath.Join(migrationsDir, base+".down.sql")

	date := time.Now().Format("2006-01-02")
	up := fmt.Sprintf(`-- %s
-- 域：%s
-- 日期：%s

CREATE TABLE IF NOT EXISTS %s (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='%s';
`, desc, c.DomainKey, date, table, desc)

	down := fmt.Sprintf(`-- 回滚：%s
-- 域：%s

DROP TABLE IF EXISTS %s;
`, desc, c.DomainKey, table)

	if err := os.WriteFile(upPath, []byte(up), 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(downPath, []byte(down), 0644); err != nil {
		return "", "", err
	}
	return upPath, downPath, nil
}

// RunMigrateScript invokes the app's scripts/migrate.sh with the given args.
// Before up/down, the target database is probed and the user is prompted to
// create it manually if missing (databases are never auto-created).
func RunMigrateScript(workspace, appName string, args ...string) error {
	if workspace == "" {
		var err error
		workspace, err = FindWorkspaceRoot("")
		if err != nil {
			return err
		}
	}
	cfg := &MigrateConfig{WorkspacePath: workspace, AppName: appName}
	_, appDir, _, err := cfg.resolve()
	if err != nil {
		return err
	}

	if len(args) > 0 && (args[0] == "up" || args[0] == "down") {
		if err := ensureDatabaseExists(appDir); err != nil {
			return err
		}
	}

	cmd := exec.Command("bash", append([]string{filepath.Join(appDir, "scripts", "migrate.sh")}, args...)...)
	cmd.Dir = appDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ensureDatabaseExists probes the configured database and, if unreachable
// because the database does not exist, instructs the user to create it manually.
func ensureDatabaseExists(appDir string) error {
	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "3306")
	user := envOr("DB_USER", "root")
	pass := os.Getenv("DB_PASS") // no default on purpose, mirrors migrate.sh
	name := envOr("DB_NAME", "hrise_admin_api")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=3s", user, pass, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil // let migrate.sh surface the real error
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unknown database") {
			return fmt.Errorf("database %q does not exist — please create it manually first:\n"+
				"  mysql -h %s -P %s -u %s -p -e 'CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;'", name, host, port, user, name)
		}
		// DB reachable issue (auth, network, down): report but keep going is wrong —
		// report and stop, the script would fail identically.
		return fmt.Errorf("cannot reach database %q: %w", name, err)
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ListMigrations prints all migration files in the app's migrations dir.
func ListMigrations(workspace, appName string) error {
	if workspace == "" {
		var err error
		workspace, err = FindWorkspaceRoot("")
		if err != nil {
			return err
		}
	}
	cfg := &MigrateConfig{WorkspacePath: workspace, AppName: appName}
	_, _, migrationsDir, err := cfg.resolve()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}
