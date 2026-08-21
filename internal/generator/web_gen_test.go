package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebGenStyles_Registered 校验风格注册表完整（新增风格必须登记）。
func TestWebGenStyles_Registered(t *testing.T) {
	for name := range webGenStyles {
		if name != "dialog" && name != "page" {
			continue
		}
		style := webGenStyles[name]
		if len(style.files) == 0 {
			t.Errorf("style %q has no files", name)
		}
		if style.routeFragment == "" {
			t.Errorf("style %q has no route fragment template", name)
		}
		// 模板文件必须存在于 embed FS
		for _, f := range style.files {
			if _, err := webTemplates.ReadFile("templates/web/" + f.tmpl); err != nil {
				t.Errorf("style %q template %s missing: %v", name, f.tmpl, err)
			}
		}
		if _, err := webTemplates.ReadFile("templates/web/" + style.routeFragment); err != nil {
			t.Errorf("style %q route fragment %s missing: %v", name, style.routeFragment, err)
		}
	}
}

// TestResolveWebFields_Defaults 校验默认呈现规则（CRUD-SPEC §2 兜底）。
func TestResolveWebFields_Defaults(t *testing.T) {
	def := &Def{
		Domain: "admin", Entity: "admin", Table: "admins",
		Fields: []DefField{
			{Name: "username", Type: "string", Size: 50, Required: true, Comment: "用户名"},
			{Name: "password", Type: "string", Required: true, JSON: "-"},
			{Name: "role", Type: "int8", Default: "2", Validate: "in:1|2", Comment: "角色"},
			{Name: "last_login_at", Type: "datetime"},
		},
	}
	ui := &UIDef{Style: "dialog"}
	fields, err := resolveWebFields(def, ui)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]webGenField{}
	for _, f := range fields {
		byName[f.Name] = f
	}

	// username：string 默认列+表单（搜索 opt-in：默认不进筛选栏），required 派生校验
	u := byName["username"]
	if !u.ColumnShow || u.SearchShow || !u.FormShow || u.FormType != "input" {
		t.Errorf("username defaults = %+v, want column+form input, search off by default", u)
	}
	if len(u.FormRules) == 0 || !u.FormRules[0].Required {
		t.Errorf("username rules = %+v, want required rule", u.FormRules)
	}

	// password：json:"-" 不展示列/搜索，仅新建表单
	p := byName["password"]
	if p.ColumnShow || p.SearchShow || !p.FormShow || !p.FormCreateOnly {
		t.Errorf("password defaults = %+v, want create-only form field", p)
	}

	// role：enum → 列带 options + select 表单；搜索 opt-in（ui def 显式开启）
	r := byName["role"]
	if !r.ColumnShow || len(r.ColumnOptions) != 2 || r.ColumnOptions["1"] == "" {
		t.Errorf("role column options = %+v, want 2 options", r.ColumnOptions)
	}
	if r.FormType != "select" || r.SearchType != "select" || r.SearchShow {
		t.Errorf("role form/search = %q/%q (show=%v), want select/select/false by default", r.FormType, r.SearchType, r.SearchShow)
	}

	// ui def 显式开启搜索后生效（仅对声明过的字段）
	uiWithSearch := &UIDef{Style: "dialog", Fields: map[string]UIField{
		"username": {Search: &UISearch{Show: true}},
		"role":     {Search: &UISearch{Show: true, Type: "select"}},
	}}
	fields2, err := resolveWebFields(def, uiWithSearch)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fields2 {
		if (f.Name == "username" || f.Name == "role") && !f.SearchShow {
			t.Errorf("field %s search should be enabled by ui def", f.Name)
		}
	}

	// last_login_at：datetime 仅列
	d := byName["last_login_at"]
	if !d.ColumnShow || d.SearchShow || d.FormShow {
		t.Errorf("last_login_at defaults = %+v, want column only", d)
	}
}

// TestBuildWebGenData_Fragments 校验预渲染片段的关键内容。
func TestBuildWebGenData_Fragments(t *testing.T) {
	def := &Def{
		Domain: "admin", Entity: "admin", Table: "admins",
		Fields: []DefField{
			{Name: "username", Type: "string", Required: true, Comment: "用户名"},
			{Name: "role", Type: "int8", Validate: "in:1|2", Comment: "角色"},
		},
	}
	ui := &UIDef{
		Style: "dialog",
		Menu:  UIMenu{Title: "管理员管理", Icon: "Users"},
		Fields: map[string]UIField{
			"username": {Search: &UISearch{Show: true}},
			"role":     {Search: &UISearch{Show: true, Type: "select"}},
		},
	}
	data, err := buildWebGenData(def, ui)
	if err != nil {
		t.Fatal(err)
	}
	if data.RoutePath != "/admin" || data.RouteName != "AdminList" {
		t.Errorf("route data = %q/%q", data.RoutePath, data.RouteName)
	}
	if !strings.Contains(data.ColumnsTS, "key: 'username'") {
		t.Errorf("ColumnsTS missing username: %s", data.ColumnsTS)
	}
	if !strings.Contains(data.ColumnsTS, "roleLabels[row.role]") {
		t.Errorf("ColumnsTS missing enum label expr: %s", data.ColumnsTS)
	}
	if !strings.Contains(data.FilterSchemaTS, "key: 'username'") {
		t.Errorf("FilterSchemaTS missing username: %s", data.FilterSchemaTS)
	}
	if !strings.Contains(data.FilterSchemaTS, "type: 'select'") {
		t.Errorf("FilterSchemaTS missing role select: %s", data.FilterSchemaTS)
	}
	if !strings.Contains(data.CreateSchemaTS, "key: 'role'") || !strings.Contains(data.EditSchemaTS, "key: 'role'") {
		t.Errorf("form schema missing role: create=%q edit=%q", data.CreateSchemaTS, data.EditSchemaTS)
	}
	if !strings.Contains(data.BackendItemFieldsTS, "Username: string") {
		t.Errorf("BackendItemFieldsTS = %s", data.BackendItemFieldsTS)
	}
	if !strings.Contains(data.NormalizeAssignmentsTS, "username: raw.Username") {
		t.Errorf("NormalizeAssignmentsTS = %s", data.NormalizeAssignmentsTS)
	}
	if !strings.Contains(data.EnumLabelsTS, "roleLabels") {
		t.Errorf("EnumLabelsTS = %s", data.EnumLabelsTS)
	}
}

// TestFormSchemaCode_RulesValid 回归：规则条目必须是独立对象字面量，
// 不得出现 "{{"（ruleEntry 曾把已含花括号的对象再包一层导致生成物语法错误）。
func TestFormSchemaCode_RulesValid(t *testing.T) {
	def := &Def{
		Domain: "admin", Entity: "admin", Table: "admins",
		Fields: []DefField{
			{Name: "username", Type: "string", Size: 50, Required: true, Comment: "用户名"},
		},
	}
	ui := &UIDef{Style: "dialog"}
	fields, err := resolveWebFields(def, ui)
	if err != nil {
		t.Fatal(err)
	}
	create, edit := formSchemaCode(fields)
	for _, code := range []string{create, edit} {
		if strings.Contains(code, "{{") {
			t.Errorf("form schema contains double braces: %s", code)
		}
		if !strings.Contains(code, "{ required: true, message: '请输入用户名', trigger: 'blur' }") {
			t.Errorf("form schema missing required rule: %s", code)
		}
	}
}

// TestWebGen_PascalNamesMatchBackend 回归：前端 BackendItem 字段名必须与后端
// Go 结构体字段名逐字一致（GoFieldName 含 commonInitialisms）。
// 金样验收暴露：ToPascalCase 把 avatar_storage_id 转成 Avatar_storage_id，
// raw.Real_name 在后端不存在，normalize 后永远 undefined（编辑回填空值）。
func TestWebGen_PascalNamesMatchBackend(t *testing.T) {
	cases := map[string]string{
		"avatar_storage_id": "AvatarStorageID",
		"last_login_at":     "LastLoginAt",
		"real_name":         "RealName",
		"admin":             "Admin",
	}
	for in, want := range cases {
		if got := GoFieldName(in); got != want {
			t.Errorf("GoFieldName(%q) = %q, want %q", in, got, want)
		}
	}
	def := &Def{
		Domain: "admin", Entity: "admin", Table: "admins",
		Fields: []DefField{{Name: "real_name", Type: "string"}, {Name: "avatar_storage_id", Type: "string"}},
	}
	data, err := buildWebGenData(def, &UIDef{Style: "dialog"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data.NormalizeAssignmentsTS, "real_name: raw.RealName") {
		t.Errorf("NormalizeAssignmentsTS = %s, want raw.RealName", data.NormalizeAssignmentsTS)
	}
	if !strings.Contains(data.NormalizeAssignmentsTS, "avatar_storage_id: raw.AvatarStorageID") {
		t.Errorf("NormalizeAssignmentsTS = %s, want raw.AvatarStorageID", data.NormalizeAssignmentsTS)
	}
}

// TestInjectBlock_Idempotent marker 块注入：重复执行结果一致。
func TestInjectBlock_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.ts")
	original := "const children = [\n" +
		routeMarkerBegin + "\n" +
		"// old content\n" +
		routeMarkerEnd + "\n" +
		"]\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	fragment := "  { path: 'admin', name: 'AdminList' },\n"
	if err := injectBlock(path, routeMarkerBegin, routeMarkerEnd, fragment); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)
	if err := injectBlock(path, routeMarkerBegin, routeMarkerEnd, fragment); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Errorf("injectBlock not idempotent:\nfirst=%q\nsecond=%q", first, second)
	}
	if strings.Count(string(second), "AdminList") != 1 {
		t.Errorf("fragment duplicated: %s", second)
	}
}

// TestInjectBlock_MissingMarker 缺失 marker 必须报错并给出指引。
func TestInjectBlock_MissingMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.ts")
	if err := os.WriteFile(path, []byte("const children = []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	err := injectBlock(path, routeMarkerBegin, routeMarkerEnd, "x")
	if err == nil || !strings.Contains(err.Error(), "marker") {
		t.Errorf("error = %v, want marker guidance", err)
	}
}
