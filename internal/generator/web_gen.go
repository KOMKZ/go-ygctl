package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates/web
var webTemplates embed.FS

// webGenFile maps one template to its output path inside the app.
type webGenFile struct {
	tmpl string // path under templates/web/
	out  string // output path pattern inside the app (one %s for entity)
}

// webGenStyle is one registered CRUD template style.
// 新增风格：在 webGenStyles 登记 + 提供 templates/web/gen/<style>/ 模板。
type webGenStyle struct {
	name  string
	files []webGenFile
	// routeFragment renders the route+menu fragment injected between markers.
	routeFragment string
}

var webGenStyles = map[string]webGenStyle{
	"dialog": {
		name: "dialog",
		files: []webGenFile{
			{tmpl: "gen/common/api.ts.tmpl", out: "src/api/%s.ts"},
			{tmpl: "gen/dialog/index.vue.tmpl", out: "src/views/%s/index.vue"},
			{tmpl: "gen/dialog/form-dialog.vue.tmpl", out: "src/views/%s/form-dialog.vue"},
		},
		routeFragment: "gen/dialog/routes.ts.tmpl",
	},
	"page": {
		name: "page",
		files: []webGenFile{
			{tmpl: "gen/common/api.ts.tmpl", out: "src/api/%s.ts"},
			{tmpl: "gen/page/index.vue.tmpl", out: "src/views/%s/index.vue"},
			{tmpl: "gen/page/create.vue.tmpl", out: "src/views/%s/create.vue"},
			{tmpl: "gen/page/edit.vue.tmpl", out: "src/views/%s/edit.vue"},
		},
		routeFragment: "gen/page/routes.ts.tmpl",
	},
}

// Marker comments for idempotent route/menu fragment injection.
const (
	routeMarkerBegin = "// @ygctl-web-gen:routes - 由 ygctl web gen 管理，勿手改"
	routeMarkerEnd   = "// @ygctl-web-gen:routes:end"
	iconMarkerBegin  = "// @ygctl-web-gen:icons - 由 ygctl web gen 管理，勿手改"
	iconMarkerEnd    = "// @ygctl-web-gen:icons:end"
)

// WebGenConfig configures ygctl web gen.
type WebGenConfig struct {
	AppPath string // target admin app root (contains src/)
	DefFile string // backend def: defs/<entity>.yaml
	UIFile  string // optional frontend def; empty = auto-discover <name>.ui.yaml
	Style   string // optional style override; empty = ui def style or default
}

// WebGenResult reports generated files and the entity name.
type WebGenResult struct {
	Entity        string
	Style         string
	Files         []string
	RouteInjected bool
	IconInjected  bool
}

// Generate renders a CRUD module into the app in the resolved style.
func (c *WebGenConfig) Generate() (*WebGenResult, error) {
	def, err := LoadDef(c.DefFile)
	if err != nil {
		return nil, err
	}
	if c.AppPath == "" {
		return nil, fmt.Errorf("app path is required (--app)")
	}
	if _, err := os.Stat(filepath.Join(c.AppPath, "package.json")); err != nil {
		return nil, fmt.Errorf("app %q does not look like a web app (package.json missing)", c.AppPath)
	}

	// 前端 DSL 独立文件：defs/<name>.ui.yaml；缺失时默认规则兜底。
	ui := &UIDef{Style: DefaultUIStyle}
	uiPath := c.UIFile
	if uiPath == "" {
		uiPath = UIDefPathFor(c.DefFile)
	}
	if _, err := os.Stat(uiPath); err == nil {
		ui, err = LoadUIDef(uiPath)
		if err != nil {
			return nil, err
		}
	}
	if err := ui.Validate(); err != nil {
		return nil, err
	}
	if err := ui.ValidateAgainstDef(def); err != nil {
		return nil, err
	}
	if c.Style != "" {
		ui.Style = c.Style
		if _, ok := webGenStyles[ui.Style]; !ok {
			return nil, fmt.Errorf("unknown style %q (registered: %s)", ui.Style, strings.Join(RegisteredWebGenStyles(), ", "))
		}
	}
	style := webGenStyles[ui.Style]

	data, err := buildWebGenData(def, ui)
	if err != nil {
		return nil, err
	}

	result := &WebGenResult{Entity: def.Entity, Style: ui.Style}

	// 1. 渲染模块文件（确定性；重复运行覆盖为相同内容）
	for _, f := range style.files {
		outPath := filepath.Join(c.AppPath, fmt.Sprintf(f.out, def.Entity))
		if err := renderWebTemplate(f.tmpl, outPath, data); err != nil {
			return nil, err
		}
		result.Files = append(result.Files, outPath)
	}

	// 2. 路由片段注入（幂等：marker 块内替换）
	if err := injectRouteFragment(c.AppPath, style.routeFragment, data); err != nil {
		return nil, err
	}
	result.RouteInjected = true
	// 3. 菜单图标片段注入（幂等）
	if err := injectIconFragment(c.AppPath, data); err != nil {
		return nil, err
	}
	result.IconInjected = true
	return result, nil
}

// renderWebTemplate renders one web template with [[ ]] delimiters so Vue's
// {{ }} mustache syntax in generated SFCs stays intact.
func renderWebTemplate(tmplName, outPath string, data *webGenData) error {
	content, err := webTemplates.ReadFile("templates/web/" + tmplName)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}
	tmpl, err := template.New(tmplName).Delims("[[", "]]").Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return tmpl.Execute(file, data)
}

// injectRouteFragment replaces the block between route markers in
// src/router/index.ts. Markers must exist (web new 模板预置；既有应用需先补)。
func injectRouteFragment(appPath, routeTmpl string, data *webGenData) error {
	content, err := webTemplates.ReadFile("templates/web/" + routeTmpl)
	if err != nil {
		return err
	}
	tmpl, err := template.New(routeTmpl).Delims("[[", "]]").Parse(string(content))
	if err != nil {
		return err
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return err
	}
	return injectBlock(
		filepath.Join(appPath, "src", "router", "index.ts"),
		routeMarkerBegin, routeMarkerEnd, sb.String(),
	)
}

// injectIconFragment replaces the block between icon markers in
// src/layouts/components/SidebarMenu.vue.
func injectIconFragment(appPath string, data *webGenData) error {
	iconLine := fmt.Sprintf("  %s,\n", data.MenuIcon)
	return injectBlock(
		filepath.Join(appPath, "src", "layouts", "components", "SidebarMenu.vue"),
		iconMarkerBegin, iconMarkerEnd, iconLine,
	)
}

// injectBlock writes content between begin/end marker lines (idempotent:
// existing block is replaced). Errors when markers are absent.
func injectBlock(filePath, begin, end, content string) error {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot inject into %s: %w", filePath, err)
	}
	text := string(raw)
	beginIdx := strings.Index(text, begin)
	if beginIdx == -1 {
		return fmt.Errorf("%s: marker %q not found; add it to enable ygctl web gen", filePath, begin)
	}
	beginEnd := beginIdx + len(begin)
	afterBegin := text[beginEnd:]
	endRel := strings.Index(afterBegin, end)
	if endRel == -1 {
		return fmt.Errorf("%s: closing marker %q not found", filePath, end)
	}
	block := text[:beginEnd] + "\n" + content + text[beginEnd+endRel:]
	return os.WriteFile(filePath, []byte(block), 0644)
}

// buildWebGenData resolves every presentation decision (defaults + ui def)
// into template data, including pre-rendered TS fragments.
func buildWebGenData(def *Def, ui *UIDef) (*webGenData, error) {
	fields, err := resolveWebFields(def, ui)
	if err != nil {
		return nil, err
	}

	entityLabel := ui.Menu.EntityLabel
	if entityLabel == "" {
		entityLabel = ui.Menu.Title
	}
	data := &webGenData{
		Entity:       def.Entity,
		EntityPascal: ToPascalCase(def.Entity),
		Table:        def.Table,
		RoutePath:    "/" + def.Entity,
		RouteName:    ToPascalCase(def.Entity) + "List",
		MenuTitle:    ui.Menu.Title,
		EntityLabel:  entityLabel,
		MenuIcon:     ui.Menu.Icon,
		Style:        ui.Style,
		Fields:       fields,
	}

	// 预渲染 TS 片段（Go 内构建，模板只做壳插值）
	data.ColumnsTS = columnsCode(def.Entity, fields)
	data.FilterSchemaTS = filterSchemaCode(fields)
	data.CreateSchemaTS, data.EditSchemaTS = formSchemaCode(fields)
	data.HasCreateOnlyFields = anyCreateOnlyField(fields)
	data.BackendItemFieldsTS = backendItemFieldsTS(fields)
	data.RecordFieldsTS = recordFieldsTS(fields)
	data.FormModelFieldsTS = formModelFieldsTS(fields)
	data.NormalizeAssignmentsTS = normalizeAssignmentsTS(fields)
	data.RecordFillAssignmentsTS = recordFillAssignmentsTS(fields)
	data.EnumLabelsTS = enumLabelsCode(fields)
	data.FilterValuesTS = filterValuesCode(fields)
	data.FilterLocalsTS = filterLocalsCode(fields)
	data.FilterGuardExpr = filterGuardExpr(fields)
	data.ClientFilterExpr = clientFilterCode(fields)
	data.ResetFilterCode = resetFilterCode(fields)
	data.EmptyModelTS = emptyModelTS(fields)
	return data, nil
}

// enumLabelsCode renders label map constants for enum columns.
func enumLabelsCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if len(f.ColumnOptions) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("const %sLabels: Record<string, string> = %s\n", f.Name, enumLabelsTS(f)))
	}
	return sb.String()
}

// webGenField is one resolved field for template consumption.
type webGenField struct {
	Name           string // snake_case
	PascalName     string
	Label          string
	Comment        string
	DSLType        string
	JSONHidden     bool // def json:"-"：不出现在响应与记录类型中
	ColumnShow     bool
	ColumnWidth    int
	ColumnOptions  map[string]string // enum display map（value->label）
	SearchShow     bool
	SearchType     string // input|select
	SearchOptions  map[string]string
	FormShow       bool
	FormCreateOnly bool   // e.g. password：仅新建表单
	FormType       string // input|select|textarea|password|number
	FormOptions    map[string]string
	FormRules      []webGenRule
	Placeholder    string
	Default        string // def default（表单默认值，select 优先取 def default）
}

// JSONHidden reports whether the field is excluded from JSON responses.
func (f webGenField) IsJSONHidden() bool { return f.JSONHidden }

type webGenRule struct {
	Required bool
	Min      int
	Max      int
	Pattern  string
	Message  string
}

// resolveWebFields merges backend def fields with ui def overrides and
// applies the default presentation rules（CRUD-SPEC §2 映射规则兜底）。
func resolveWebFields(def *Def, ui *UIDef) ([]webGenField, error) {
	fields := make([]webGenField, 0, len(def.Fields))
	for _, f := range def.Fields {
		// PascalName 必须用 GoFieldName（与后端 Go 结构体字段同名）：
		// 后端 JSON 键来自 Go 字段名，前端 BackendItem 类型必须逐字对齐
		// （含 commonInitialisms，如 avatar_storage_id -> AvatarStorageID）。
		w := webGenField{
			Name:       f.Name,
			PascalName: GoFieldName(f.Name),
			Label:      f.Comment,
			Comment:    f.Comment,
			DSLType:    f.Type,
			JSONHidden: f.JSON == "-",
			Default:    f.Default,
		}
		if w.Label == "" {
			w.Label = f.Name
		}
		w.Placeholder = "请输入" + w.Label

		enumValues := enumValuesOf(f)
		isEnum := len(enumValues) > 0

		// ---- 默认规则 ----
		w.ColumnShow = f.JSON != "-"
		w.ColumnWidth = 0
		if isEnum {
			w.ColumnOptions = optionMap(enumValues, nil)
		}
		// 搜索默认 opt-in：只有 ui def 显式声明 search 段才进筛选栏
		// （避免 phone/avatar 等字段被默认塞进搜索；最小惊讶原则）
		w.SearchShow = false
		w.SearchType = "input"
		w.FormShow = f.Type != "datetime" && f.Type != "date" && f.Type != "blob"
		if f.JSON == "-" {
			w.FormCreateOnly = true
		}
		switch {
		case f.Type == "datetime" || f.Type == "date":
			w.FormType = ""
		case isEnum:
			w.FormType = "select"
			w.FormOptions = optionMap(enumValues, nil)
			w.SearchType = "select"
			w.SearchOptions = w.FormOptions
		case f.Type == "text":
			w.FormType = "textarea"
		case f.Type == "int" || f.Type == "int8" || f.Type == "int16" || f.Type == "int32" ||
			f.Type == "int64" || f.Type == "uint" || f.Type == "uint8" || f.Type == "uint16" ||
			f.Type == "uint32" || f.Type == "uint64" || f.Type == "float64":
			w.FormType = "number"
			w.SearchShow = false // 数值字段默认不进搜索
		default:
			w.FormType = "input"
		}

		// def 派生校验规则
		if f.Required && w.FormShow {
			w.FormRules = append(w.FormRules, webGenRule{Required: true, Message: "请输入" + w.Label})
		}
		if f.Type == "string" && f.Size > 0 {
			w.FormRules = append(w.FormRules, webGenRule{Max: f.Size})
		}
		for _, v := range strings.Split(f.Validate, ",") {
			v = strings.TrimSpace(v)
			switch {
			case v == "email":
				w.FormRules = append(w.FormRules, webGenRule{Pattern: `^\S+@\S+\.\S+$`, Message: "请输入正确的邮箱格式"})
			case strings.HasPrefix(v, "min:"):
				var n int
				fmt.Sscanf(strings.TrimPrefix(v, "min:"), "%d", &n)
				w.FormRules = append(w.FormRules, webGenRule{Min: n, Message: fmt.Sprintf("至少%d位", n)})
			}
		}

		// ---- ui def 覆盖 ----
		if uf, ok := ui.Fields[f.Name]; ok {
			if uf.Column != nil {
				w.ColumnShow = uf.Column.Show
				if uf.Column.Width > 0 {
					w.ColumnWidth = uf.Column.Width
				}
				if uf.Column.Label != "" {
					w.Label = uf.Column.Label
				}
				if len(uf.Column.Options) > 0 {
					w.ColumnOptions = optionMap(enumValues, uf.Column.Options)
				}
			}
			if uf.Search != nil {
				w.SearchShow = uf.Search.Show
				if uf.Search.Type != "" {
					w.SearchType = uf.Search.Type
				}
				if len(uf.Search.Options) > 0 {
					w.SearchOptions = optionMap(enumValues, uf.Search.Options)
				}
			}
			if uf.Form != nil {
				w.FormShow = uf.Form.Show
				if uf.Form.Type != "" {
					w.FormType = uf.Form.Type
				}
				if len(uf.Form.Options) > 0 {
					w.FormOptions = optionMap(enumValues, uf.Form.Options)
				}
				if uf.Form.Placeholder != "" {
					w.Placeholder = uf.Form.Placeholder
				}
				if uf.Form.Rules != nil {
					w.FormRules = nil
					r := uf.Form.Rules
					if r.Required {
						w.FormRules = append(w.FormRules, webGenRule{Required: true, Message: r.Message})
					}
					if r.Min > 0 {
						w.FormRules = append(w.FormRules, webGenRule{Min: r.Min, Message: r.Message})
					}
					if r.Max > 0 {
						w.FormRules = append(w.FormRules, webGenRule{Max: r.Max, Message: r.Message})
					}
					if r.Pattern != "" {
						w.FormRules = append(w.FormRules, webGenRule{Pattern: r.Pattern, Message: r.Message})
					}
				}
			}
		}
		fields = append(fields, w)
	}
	return fields, nil
}

// enumValuesOf extracts in:a|b values from the def validate expression.
func enumValuesOf(f DefField) []string {
	for _, v := range strings.Split(f.Validate, ",") {
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "in:") {
			items := strings.Split(strings.TrimPrefix(v, "in:"), "|")
			out := make([]string, 0, len(items))
			for _, it := range items {
				out = append(out, strings.TrimSpace(it))
			}
			return out
		}
	}
	return nil
}

// optionMap builds a value->label map; ui labels override defaults (label=value).
func optionMap(values []string, labels map[string]string) map[string]string {
	m := make(map[string]string, len(values))
	for _, v := range values {
		l := v
		if labels != nil {
			if custom, ok := labels[v]; ok {
				l = custom
			}
		}
		m[v] = l
	}
	return m
}

func anyCreateOnlyField(fields []webGenField) bool {
	for _, f := range fields {
		if f.FormCreateOnly {
			return true
		}
	}
	return false
}

// webGenData is the template data for all web gen templates.
type webGenData struct {
	Entity       string
	EntityPascal string
	Table        string
	RoutePath    string
	RouteName    string
	MenuTitle    string
	EntityLabel  string
	MenuIcon     string
	Style        string
	Fields       []webGenField

	// 预渲染 TS 片段
	ColumnsTS               string
	FilterSchemaTS          string
	CreateSchemaTS          string
	EditSchemaTS            string
	HasCreateOnlyFields     bool
	BackendItemFieldsTS     string
	RecordFieldsTS          string
	FormModelFieldsTS       string
	NormalizeAssignmentsTS  string
	RecordFillAssignmentsTS string
	EnumLabelsTS            string
	FilterValuesTS          string
	FilterLocalsTS          string
	FilterGuardExpr         string
	ClientFilterExpr        string
	ResetFilterCode         string
	EmptyModelTS            string
}
