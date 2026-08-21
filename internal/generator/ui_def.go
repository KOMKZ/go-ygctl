package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// UIDef is the frontend-only def file content (defs/<entity>.ui.yaml).
// 前后端 DSL 分文件：后端 defs/<entity>.yaml 保持纯后端字段/端点；
// 前端呈现（菜单/列/搜索/表单/风格）全部在本文件，文件名以 .ui.yaml 区分。
// api gen 不消费本文件；web gen 消费（缺失时走默认规则兜底）。
type UIDef struct {
	// Style is the CRUD template style: dialog (default) | page.
	// 设计上支持扩展：新增风格 = templates/web/gen/<style>/ + 风格注册表加一项。
	Style string `yaml:"style"`
	Menu  UIMenu `yaml:"menu"`
	// Fields is keyed by backend def field name; only fields needing
	// customization need to be listed, the rest falls back to defaults.
	Fields map[string]UIField `yaml:"fields"`
}

// UIMenu is the sidebar menu spec of the generated CRUD module.
type UIMenu struct {
	Title       string `yaml:"title"`       // 菜单标题（必填）
	EntityLabel string `yaml:"entityLabel"` // 实体单数名（按钮/对话框标题用，如「管理员」）；空 = 用 title
	Icon        string `yaml:"icon"`        // 图标名（必须可被前端 iconMap 解析）
	Order       int    `yaml:"order"`       // 菜单排序（可选，暂不消费）
}

// UIField is the presentation spec of one backend def field.
type UIField struct {
	Column *UIColumn `yaml:"column"` // nil = 默认规则
	Search *UISearch `yaml:"search"`
	Form   *UIForm   `yaml:"form"`
}

// UIColumn controls table column presentation.
type UIColumn struct {
	Show   bool              `yaml:"show"`   // 默认按规则兜底
	Width  int               `yaml:"width"`  // 列宽 px；0 = 不指定
	Label  string            `yaml:"label"`  // 列头；空 = 用 def comment/字段名
	Options map[string]string `yaml:"options"` // value -> display label（枚举列渲染）
}

// UISearch controls filter-bar presentation.
type UISearch struct {
	Show    bool   `yaml:"show"`
	Type    string `yaml:"type"`    // input | select；空 = 按字段类型推断
	Options map[string]string `yaml:"options"` // select 选项（value -> label）
}

// UIForm controls create/edit form presentation.
type UIForm struct {
	Show    bool   `yaml:"show"`
	Type    string `yaml:"type"` // input | select | textarea | password | number；空 = 按字段类型推断
	Rules   *UIRules `yaml:"rules"` // 覆盖 def 派生校验（required/min/max/pattern）
	Options map[string]string `yaml:"options"`
	Placeholder string `yaml:"placeholder"`
}

// UIRules overrides form validation rules for one field.
type UIRules struct {
	Required bool   `yaml:"required"`
	Min      int    `yaml:"min"`
	Max      int    `yaml:"max"`
	Pattern  string `yaml:"pattern"` // JS 正则字符串
	Message  string `yaml:"message"`
}

// DefaultUIStyle is the style used when the ui def does not declare one.
const DefaultUIStyle = "dialog"

// WebGenStyles is the style registry: name -> registered. 新增风格时在
// web_gen.go 的 webGenStyles 中登记并在 templates/web/gen/<style>/ 提供模板。
func RegisteredWebGenStyles() []string {
	names := make([]string, 0, len(webGenStyles))
	for n := range webGenStyles {
		names = append(names, n)
	}
	return names
}

// UIDefPathFor derives the ui def path of a backend def: defs/<name>.ui.yaml.
func UIDefPathFor(backendDefPath string) string {
	dir := filepath.Dir(backendDefPath)
	base := filepath.Base(backendDefPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+".ui.yaml")
}

// LoadUIDef reads and validates a frontend ui def file.
func LoadUIDef(path string) (*UIDef, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ui def %s: %w", path, err)
	}
	ui := &UIDef{}
	if err := yaml.Unmarshal(content, ui); err != nil {
		return nil, fmt.Errorf("invalid ui def yaml: %w", err)
	}
	if err := ui.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ui def %s: %w", path, err)
	}
	return ui, nil
}

// Validate checks the ui def structure. 引用校验（fields 必须存在于后端 def）
// 在 ValidateAgainstDef 中进行（需要联合后端 def）。
func (u *UIDef) Validate() error {
	if u.Style == "" {
		u.Style = DefaultUIStyle
	}
	if _, ok := webGenStyles[u.Style]; !ok {
		return fmt.Errorf("unknown style %q (registered: %s)", u.Style, strings.Join(RegisteredWebGenStyles(), ", "))
	}
	if u.Menu.Title == "" {
		return fmt.Errorf("menu.title is required")
	}
	if u.Menu.Icon == "" {
		return fmt.Errorf("menu.icon is required (frontend sidebar icon contract)")
	}
	for name, f := range u.Fields {
		if !fieldNameRe.MatchString(name) {
			return fmt.Errorf("ui field key %q must be lowercase snake_case", name)
		}
		if f.Column != nil && f.Column.Width < 0 {
			return fmt.Errorf("field %q column.width must be >= 0", name)
		}
		if f.Search != nil {
			if f.Search.Type != "" && !isUISearchType(f.Search.Type) {
				return fmt.Errorf("field %q search.type %q unknown (allowed: input, select)", name, f.Search.Type)
			}
		}
		if f.Form != nil {
			if f.Form.Type != "" && !isUIFormType(f.Form.Type) {
				return fmt.Errorf("field %q form.type %q unknown (allowed: input, select, textarea, password, number)", name, f.Form.Type)
			}
			if f.Form.Rules != nil && f.Form.Rules.Min < 0 {
				return fmt.Errorf("field %q form.rules.min must be >= 0", name)
			}
		}
	}
	return nil
}

// ValidateAgainstDef checks ui field keys reference existing backend def fields.
func (u *UIDef) ValidateAgainstDef(def *Def) error {
	known := map[string]bool{}
	for i := range def.Fields {
		known[def.Fields[i].Name] = true
	}
	for name := range u.Fields {
		if !known[name] {
			return fmt.Errorf("ui field %q does not exist in backend def fields", name)
		}
	}
	return nil
}

func isUISearchType(t string) bool {
	return t == "input" || t == "select"
}

func isUIFormType(t string) bool {
	switch t {
	case "input", "select", "textarea", "password", "number":
		return true
	}
	return false
}
