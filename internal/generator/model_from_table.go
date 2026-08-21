package generator

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// TableColumn is one column read from information_schema.
type TableColumn struct {
	Name          string
	DataType      string
	ColumnType    string
	IsNullable    string
	ColumnKey     string
	ColumnDefault sql.NullString
	ColumnComment string
	MaxLength     sql.NullInt64
}

// ModelFromTableConfig holds the configuration for model generation from a table.
type ModelFromTableConfig struct {
	WorkspacePath string
	DomainKey     string
	TableName     string
	Force         bool
}

// GenerateModelFromTable reads information_schema and writes a gorm model
// into domains/<domain>/model/<table>.go.
func (c *ModelFromTableConfig) GenerateModelFromTable() (outPath string, err error) {
	if !domainKeyRe.MatchString(c.DomainKey) {
		return "", fmt.Errorf("invalid --domain %q: use lowercase snake/kebab-case", c.DomainKey)
	}
	if c.TableName == "" {
		return "", fmt.Errorf("table name is required")
	}

	workspace := c.WorkspacePath
	if workspace == "" {
		workspace, err = FindWorkspaceRoot("")
		if err != nil {
			return "", err
		}
	} else {
		workspace, err = filepath.Abs(workspace)
		if err != nil {
			return "", err
		}
	}

	domainDir := filepath.Join(workspace, "domains", c.DomainKey)
	if _, err := os.Stat(filepath.Join(domainDir, "go.mod")); err != nil {
		return "", fmt.Errorf("domain %q does not exist (run: ygctl domain init %s): %w", c.DomainKey, c.DomainKey, err)
	}
	outPath = filepath.Join(domainDir, "model", c.TableName+".go")
	if _, err := os.Stat(outPath); err == nil && !c.Force {
		return "", fmt.Errorf("%s already exists; pass --force to overwrite", outPath)
	}

	columns, err := c.readColumns()
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", fmt.Errorf("table %q not found in information_schema (check DB_NAME)", c.TableName)
	}

	content := renderModel(c.TableName, columns)
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", err
	}

	gofmtCmd := exec.Command("gofmt", "-w", outPath)
	if out, err := gofmtCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}

	buildCmd := exec.Command("go", "build", "./domains/"+c.DomainKey+"/...")
	buildCmd.Dir = workspace
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build failed: %v\n%s", err, out)
	}
	return outPath, nil
}

// readColumns queries information_schema.columns for the table.
func (c *ModelFromTableConfig) readColumns() ([]TableColumn, error) {
	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "3306")
	user := envOr("DB_USER", "root")
	pass := os.Getenv("DB_PASS")
	name := envOr("DB_NAME", "hrise_admin_api")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=5s", user, pass, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT column_name, data_type, column_type, is_nullable, column_key,
		column_default, column_comment, character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position`
	rows, err := db.Query(query, name, c.TableName)
	if err != nil {
		return nil, fmt.Errorf("information_schema query failed: %w", err)
	}
	defer rows.Close()

	var columns []TableColumn
	for rows.Next() {
		var col TableColumn
		if err := rows.Scan(&col.Name, &col.DataType, &col.ColumnType, &col.IsNullable,
			&col.ColumnKey, &col.ColumnDefault, &col.ColumnComment, &col.MaxLength); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}
	sort.Slice(columns, func(i, j int) bool { return columns[i].Name < columns[j].Name })
	return columns, rows.Err()
}

// commonInitialisms are words rendered as full uppercase in Go field names.
var commonInitialisms = map[string]bool{
	"id": true, "ip": true, "url": true, "uri": true, "uid": true, "api": true,
	"http": true, "https": true, "ttl": true, "mcp": true, "sql": true, "db": true,
	"uuid": true, "jwt": true, "rpc": true, "cli": true, "ai": true, "ui": true,
	"json": true, "xml": true, "cpu": true, "gpu": true, "os": true, "io": true,
	"dsn": true, "dto": true, "crud": true,
}

// fieldName converts a snake_case column name to a Go field name.
func fieldName(snake string) string {
	parts := strings.Split(snake, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if commonInitialisms[p] {
			parts[i] = strings.ToUpper(p)
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// goType maps a MySQL column to a Go type.
func goType(col TableColumn) string {
	switch col.DataType {
	case "tinyint":
		if strings.HasPrefix(col.ColumnType, "tinyint(1)") {
			return "bool"
		}
		if strings.Contains(col.ColumnType, "unsigned") {
			return "uint8"
		}
		return "int8"
	case "smallint":
		if strings.Contains(col.ColumnType, "unsigned") {
			return "uint16"
		}
		return "int16"
	case "mediumint":
		return "int32"
	case "int":
		if strings.Contains(col.ColumnType, "unsigned") {
			return "uint32"
		}
		return "int"
	case "bigint":
		if strings.Contains(col.ColumnType, "unsigned") {
			return "uint64"
		}
		return "int64"
	case "decimal", "numeric", "float", "double":
		return "float64"
	case "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "enum", "set", "json":
		return "string"
	case "date", "datetime", "timestamp", "time":
		return "time.Time"
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary", "bit":
		return "[]byte"
	default:
		return "string" // unknown types fall back to string; review manually
	}
}

// gormTags builds the gorm tag segment for one column (no backticks).
// The caller joins it with the json segment inside a single backtick pair.
func gormTags(col TableColumn) string {
	tags := []string{}
	if col.ColumnKey == "PRI" {
		tags = append(tags, "primarykey")
	}
	if col.ColumnKey == "UNI" {
		tags = append(tags, "uniqueIndex")
	}
	if col.MaxLength.Valid && (col.DataType == "varchar" || col.DataType == "char") {
		tags = append(tags, fmt.Sprintf("size:%d", col.MaxLength.Int64))
	}
	if col.IsNullable == "NO" && col.ColumnKey != "PRI" {
		tags = append(tags, "not null")
	}
	if col.ColumnDefault.Valid && col.ColumnDefault.String != "" && !strings.HasPrefix(col.ColumnDefault.String, "CURRENT_TIMESTAMP") {
		tags = append(tags, fmt.Sprintf("default:%s", col.ColumnDefault.String))
	}
	if len(tags) == 0 {
		return ""
	}
	return fmt.Sprintf("gorm:%q ", strings.Join(tags, ";"))
}

// renderModel renders the full model source.
func renderModel(table string, columns []TableColumn) string {
	var b strings.Builder
	b.WriteString("package model\n\n")
	b.WriteString("import \"time\"\n\n")
	b.WriteString("// Generated by ygctl model from-table (table: " + table + ").\n")
	b.WriteString("// 生成后人工核对：类型映射（tinyint/decimal/json）、默认值、指针化规则。\n")
	b.WriteString("type " + fieldName(table) + " struct {\n")
	for _, col := range columns {
		name := fieldName(col.Name)
		typ := goType(col)
		// Nullable time columns without default become pointers (matches hand-written models).
		if col.IsNullable == "YES" && typ == "time.Time" && !col.ColumnDefault.Valid {
			typ = "*time.Time"
		}
		comment := ""
		if col.ColumnComment != "" {
			comment = " // " + col.ColumnComment
		}
		// gorm and json segments share ONE backtick-delimited tag.
		b.WriteString(fmt.Sprintf("\t%s %s `%sjson:\"%s\"`%s\n", name, typ, gormTags(col), col.Name, comment))
	}
	b.WriteString("}\n\n")
	b.WriteString("// TableName returns the MySQL table name.\n")
	b.WriteString("func (" + fieldName(table) + ") TableName() string {\n\treturn \"" + table + "\"\n}\n")
	return b.String()
}
