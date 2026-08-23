package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefField is one business field declared in an api def.
type DefField struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Size     int    `yaml:"size"`
	Unique   bool   `yaml:"unique"`
	Index    bool   `yaml:"index"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default"`
	Comment  string `yaml:"comment"`
	JSON     string `yaml:"json"` // json tag override; "-" hides the field
	// Validate declares extra validation rules, comma-separated:
	// email | min:N | max:N | in:a|b|c
	Validate string `yaml:"validate"`
}

// EndpointDef is one declared CRUD endpoint with optional permission metadata
// (name/description flow into the permission dictionary and route scan).
type EndpointDef struct {
	Action      string `yaml:"action"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Method      string `yaml:"method"`
	Path        string `yaml:"path"`
	RoutePath   string `yaml:"-"`
}

// UnmarshalYAML accepts both plain strings ("list") and objects
// ({action: list, name: ..., description: ...}).
func (e *EndpointDef) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		e.Action = node.Value
		return nil
	}
	type raw EndpointDef
	return node.Decode((*raw)(e))
}

// Def is the api def file content (defs/<entity>.yaml).
type Def struct {
	Domain    string        `yaml:"domain"`
	Entity    string        `yaml:"entity"`
	Table     string        `yaml:"table"`
	RouteBase string        `yaml:"route_base"`
	Fields    []DefField    `yaml:"fields"`
	Queries   []string      `yaml:"queries"`   // e.g. by_username -> FindByUsername
	Endpoints []EndpointDef `yaml:"endpoints"` // list/get/create/update/delete (string or object form)
}

// DSLType maps a DSL type name to Go type, MySQL column type and default size.
type DSLType struct {
	GoType       string
	MySQLType    string
	DefaultSize  int
	NeedsTimePkg bool
}

var dslTypes = map[string]DSLType{
	"string":   {GoType: "string", MySQLType: "varchar", DefaultSize: 255},
	"text":     {GoType: "string", MySQLType: "text"},
	"int":      {GoType: "int", MySQLType: "int"},
	"int8":     {GoType: "int8", MySQLType: "tinyint"},
	"int16":    {GoType: "int16", MySQLType: "smallint"},
	"int32":    {GoType: "int32", MySQLType: "int"},
	"int64":    {GoType: "int64", MySQLType: "bigint"},
	"uint":     {GoType: "uint", MySQLType: "int unsigned"},
	"uint8":    {GoType: "uint8", MySQLType: "tinyint unsigned"},
	"uint16":   {GoType: "uint16", MySQLType: "smallint unsigned"},
	"uint32":   {GoType: "uint32", MySQLType: "int unsigned"},
	"uint64":   {GoType: "uint64", MySQLType: "bigint unsigned"},
	"bool":     {GoType: "bool", MySQLType: "tinyint(1)"},
	"float64":  {GoType: "float64", MySQLType: "double"},
	"datetime": {GoType: "time.Time", MySQLType: "datetime", NeedsTimePkg: true},
	"date":     {GoType: "time.Time", MySQLType: "date", NeedsTimePkg: true},
	"json":     {GoType: "string", MySQLType: "json"},
	"blob":     {GoType: "[]byte", MySQLType: "longblob"},
}

var (
	fieldNameRe  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	endpointsSet = map[string]bool{"list": true, "get": true, "create": true, "update": true, "delete": true}
)

// LoadDef reads and validates an api def file.
func LoadDef(path string) (*Def, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read def %s: %w", path, err)
	}
	def := &Def{}
	if err := yaml.Unmarshal(content, def); err != nil {
		return nil, fmt.Errorf("invalid def yaml: %w", err)
	}
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("invalid def %s: %w", path, err)
	}
	return def, nil
}

// Validate checks the def for structural and referential errors.
func (d *Def) Validate() error {
	if d.Domain == "" {
		d.Domain = d.Entity
	}
	if !domainKeyRe.MatchString(d.Domain) {
		return fmt.Errorf("domain %q must be lowercase snake/kebab-case", d.Domain)
	}
	if d.Entity == "" {
		return fmt.Errorf("entity is required")
	}
	if !domainKeyRe.MatchString(d.Entity) {
		return fmt.Errorf("entity %q must be lowercase snake/kebab-case", d.Entity)
	}
	if d.Table == "" {
		d.Table = d.Entity + "s"
	}
	d.RouteBase = normalizeRouteBase(d.RouteBase, d.Table)
	if len(d.Fields) == 0 {
		return fmt.Errorf("at least one business field is required")
	}

	seen := map[string]bool{}
	for i := range d.Fields {
		f := &d.Fields[i]
		if !fieldNameRe.MatchString(f.Name) {
			return fmt.Errorf("field %q must be lowercase snake_case", f.Name)
		}
		if seen[f.Name] {
			return fmt.Errorf("duplicate field %q", f.Name)
		}
		seen[f.Name] = true
		if _, ok := dslTypes[f.Type]; !ok {
			return fmt.Errorf("field %q has unknown type %q (allowed: %s)", f.Name, f.Type, strings.Join(dslTypeNames(), ", "))
		}
	}

	// Queries: by_xxx must reference an existing field.
	for _, q := range d.Queries {
		if !strings.HasPrefix(q, "by_") {
			return fmt.Errorf("query %q must start with by_ (e.g. by_username)", q)
		}
		col := strings.TrimPrefix(q, "by_")
		if !seen[col] {
			return fmt.Errorf("query %q references unknown field %q", q, col)
		}
	}

	if len(d.Endpoints) == 0 {
		for _, a := range []string{"list", "get", "create", "update", "delete"} {
			d.Endpoints = append(d.Endpoints, EndpointDef{Action: a})
		}
	}
	for i := range d.Endpoints {
		e := &d.Endpoints[i]
		if !endpointsSet[e.Action] {
			return fmt.Errorf("unknown endpoint %q (allowed: list/get/create/update/delete)", e.Action)
		}
		e.Method = strings.ToUpper(strings.TrimSpace(e.Method))
		if e.Method == "" {
			e.Method = defaultEndpointMethod(e.Action)
		}
		e.Path = strings.TrimSpace(e.Path)
		e.RoutePath = normalizeEndpointRoutePath(e.Path, e.Action)
		if e.Name == "" {
			// 默认权限名：<entity> <action> 权限（可读但建议在 def 中显式定义）
			e.Name = fmt.Sprintf("%s %s 权限", d.Entity, e.Action)
		}
	}
	return nil
}

func normalizeRouteBase(routeBase, table string) string {
	routeBase = strings.TrimSpace(routeBase)
	if routeBase == "" {
		routeBase = "/" + table
	}
	if !strings.HasPrefix(routeBase, "/") {
		routeBase = "/" + routeBase
	}
	if routeBase != "/" {
		routeBase = strings.TrimRight(routeBase, "/")
	}
	return routeBase
}

func defaultEndpointMethod(action string) string {
	switch action {
	case "list", "get":
		return "GET"
	case "create":
		return "POST"
	case "update":
		return "PUT"
	case "delete":
		return "DELETE"
	default:
		return "GET"
	}
}

func normalizeEndpointRoutePath(path, action string) string {
	if path == "" {
		switch action {
		case "list":
			return "/page"
		case "get", "update", "delete":
			return "/:id"
		case "create":
			return ""
		default:
			return ""
		}
	}
	if path == "." || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func dslTypeNames() []string {
	names := make([]string, 0, len(dslTypes))
	for n := range dslTypes {
		names = append(names, n)
	}
	return names
}

// GoFieldName converts a snake_case field name to PascalCase Go field name.
func GoFieldName(snake string) string {
	return fieldName(snake)
}

// FieldGoType returns the Go type for a def field, pointerizing nullable
// datetime fields without default (matches hand-written domain models).
func FieldGoType(f DefField) string {
	t := dslTypes[f.Type]
	goType := t.GoType
	if f.Type == "datetime" && !f.Required && f.Default == "" {
		goType = "*time.Time"
	}
	return goType
}

// FieldStructTag builds the complete struct tag for a def field.
// Go allows exactly ONE backtick-delimited tag per field, so gorm and json
// segments are joined inside a single backtick pair.
func FieldStructTag(f DefField) string {
	parts := []string{}
	gormParts := []string{}
	if f.Unique {
		gormParts = append(gormParts, "uniqueIndex")
	} else if f.Index {
		gormParts = append(gormParts, "index")
	}
	if f.Type == "string" && f.Size > 0 {
		gormParts = append(gormParts, fmt.Sprintf("size:%d", f.Size))
	}
	if f.Required {
		gormParts = append(gormParts, "not null")
	}
	if f.Default != "" {
		gormParts = append(gormParts, "default:"+f.Default)
	}
	if len(gormParts) > 0 {
		parts = append(parts, fmt.Sprintf("gorm:%q", strings.Join(gormParts, ";")))
	}
	if f.JSON == "-" {
		parts = append(parts, `json:"-"`)
	} else {
		parts = append(parts, fmt.Sprintf("json:%q", f.Name))
	}
	return "`" + strings.Join(parts, " ") + "`"
}

// ValidationRules returns the ozzo-validation expressions for a def field:
// derived rules (required, length from size) plus declared ones.
// Each entry is a complete Go expression usable in a validation.Field call.
//
// 修复记录（hrise-frontend-optiz 金样暴露的生成器根因）：
//   - email 使用 is.EmailFormat（is.Email 对常见地址误拒）
//   - in: 字面量按字段 Go 类型生成（int8/int16 必须带类型转换，否则与
//     ozzo reflect.DeepEqual 校验永远不匹配，创建/更新接口 100% 校验失败）
func ValidationRules(f DefField) []string {
	rules := []string{}
	if f.Required {
		rules = append(rules, "validation.Required")
	}
	if f.Type == "string" && f.Size > 0 {
		rules = append(rules, fmt.Sprintf("validation.Length(1, %d)", f.Size))
	}
	for _, v := range strings.Split(f.Validate, ",") {
		v = strings.TrimSpace(v)
		switch {
		case v == "":
		case v == "email":
			rules = append(rules, "is.EmailFormat")
		case strings.HasPrefix(v, "min:"):
			rules = append(rules, fmt.Sprintf("validation.Length(%s, 0)", strings.TrimPrefix(v, "min:")))
		case strings.HasPrefix(v, "max:"):
			rules = append(rules, fmt.Sprintf("validation.Length(1, %s)", strings.TrimPrefix(v, "max:")))
		case strings.HasPrefix(v, "in:"):
			items := strings.Split(strings.TrimPrefix(v, "in:"), "|")
			literals := make([]string, 0, len(items))
			for _, it := range items {
				literals = append(literals, inLiteral(f.Type, strings.TrimSpace(it)))
			}
			rules = append(rules, fmt.Sprintf("validation.In(%s)", strings.Join(literals, ", ")))
		default:
			rules = append(rules, v) // pass through custom rule expressions
		}
	}
	return rules
}

// inLiteral renders one in: item as a Go literal matching the DSL field type.
// int8/int16 need explicit conversions (ozzo In compares via reflect.DeepEqual);
// other numeric DSL types accept untyped constants; string items are quoted.
func inLiteral(dslType, item string) string {
	switch dslType {
	case "int8", "int16":
		return fmt.Sprintf("%s(%s)", dslType, item)
	case "string":
		return fmt.Sprintf("%q", item)
	default:
		return item // int/int32/int64/uint...: untyped numeric literal
	}
}

// MySQLColumnType returns the MySQL column type for the migration skeleton.
func MySQLColumnType(f DefField) string {
	t := dslTypes[f.Type]
	col := t.MySQLType
	if f.Type == "string" {
		size := f.Size
		if size == 0 {
			size = t.DefaultSize
		}
		col = fmt.Sprintf("varchar(%d)", size)
	}
	return col
}

// InitDefFile generates a def template at defs/<entity>.yaml in the workspace.
type InitDefConfig struct {
	WorkspacePath string
	Domain        string
	Entity        string
	Table         string
	RouteBase     string
}

func InitDefFile(workspacePath, entity string) (string, error) {
	return InitDefFileWithConfig(InitDefConfig{WorkspacePath: workspacePath, Entity: entity})
}

func InitDefFileWithConfig(cfg InitDefConfig) (string, error) {
	workspacePath := cfg.WorkspacePath
	entity := strings.TrimSpace(cfg.Entity)
	if workspacePath == "" {
		var err error
		workspacePath, err = FindWorkspaceRoot("")
		if err != nil {
			return "", err
		}
	}
	if !domainKeyRe.MatchString(entity) {
		return "", fmt.Errorf("entity %q must be lowercase snake/kebab-case", entity)
	}
	domain := strings.TrimSpace(cfg.Domain)
	if domain == "" {
		domain = entity
	}
	if !domainKeyRe.MatchString(domain) {
		return "", fmt.Errorf("domain %q must be lowercase snake/kebab-case", domain)
	}
	table := strings.TrimSpace(cfg.Table)
	if table == "" {
		table = entity + "s"
	}
	routeBase := normalizeRouteBase(cfg.RouteBase, table)

	defsDir := filepath.Join(workspacePath, "defs")
	if err := os.MkdirAll(defsDir, 0755); err != nil {
		return "", err
	}
	outPath := filepath.Join(defsDir, entity+".yaml")
	if _, err := os.Stat(outPath); err == nil {
		return "", fmt.Errorf("%w: %s", ErrPathExists, outPath)
	}

	content := fmt.Sprintf(`# %s 领域 api def（由 ygctl api init 生成，按表结构填空）
# 约定：id/created_at/updated_at 由生成器自动附加，这里只写业务字段。
domain: %s
entity: %s
table: %s
route_base: %s
fields:
  # 业务字段示例（按需修改/删除）：
  - {name: name, type: string, size: 50, required: true, comment: "名称"}
  # 常用选项：type(string/text/int/int8..64/uint..64/bool/float64/datetime/date/json/blob)
  #          size(仅 string) unique required default comment json("-" 隐藏)
queries: []
  # 自定义查询示例：- by_name（生成 FindByName，字段必须存在）
endpoints: [list, get, create, update, delete]
`, domain, domain, entity, table, routeBase)

	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return outPath, nil
}

// ListDefFiles returns all backend def files under <workspace>/defs, excluding
// UI-only defs (*.ui.yaml). Results are sorted for stable generation order.
func ListDefFiles(workspacePath string) ([]string, error) {
	if workspacePath == "" {
		var err error
		workspacePath, err = FindWorkspaceRoot("")
		if err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}
	defsDir := filepath.Join(abs, "defs")
	entries, err := os.ReadDir(defsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read defs dir %s: %w", defsDir, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".ui.yaml") {
			continue
		}
		files = append(files, filepath.Join(defsDir, name))
	}
	sort.Strings(files)
	return files, nil
}
