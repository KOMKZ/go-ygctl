package generator

import (
	"fmt"
	"strings"
)

// columnsCode renders the DataTableColumn[] entries for the list page.
// 规则（CRUD-SPEC §2）：enum 列带 NTag 映射；datetime 列带 formatDateTime。
func columnsCode(entity string, fields []webGenField) string {
	var sb strings.Builder
	sb.WriteString("  { key: 'id', title: 'ID', width: 70, align: 'center' },\n")
	for _, f := range fields {
		if !f.ColumnShow {
			continue
		}
		width := f.ColumnWidth
		if width == 0 {
			width = defaultColumnWidth(f)
		}
		attrs := []string{}
		if width > 0 {
			attrs = append(attrs, fmt.Sprintf("width: %d", width))
		}
		if f.Label != "" && len(f.Label) > 8 {
			attrs = append(attrs, "ellipsis: true")
		}
		switch {
		case len(f.ColumnOptions) > 0:
			sb.WriteString("  {\n")
			sb.WriteString(fmt.Sprintf("    key: '%s',\n    title: '%s',\n", f.Name, f.Label))
			if len(attrs) > 0 {
				sb.WriteString("    " + strings.Join(attrs, ", ") + ",\n")
			}
			sb.WriteString("    render(row): VNode {\n")
			sb.WriteString("      return h(NTag, { size: 'small' }, { default: () => " + enumLabelExpr(f) + " })\n")
			sb.WriteString("    },\n")
			sb.WriteString("  },\n")
		case f.DSLType == "datetime" || f.DSLType == "date":
			sb.WriteString("  {\n")
			sb.WriteString(fmt.Sprintf("    key: '%s',\n    title: '%s',\n", f.Name, f.Label))
			if len(attrs) > 0 {
				sb.WriteString("    " + strings.Join(attrs, ", ") + ",\n")
			}
			sb.WriteString("    render(row): VNode {\n")
			sb.WriteString("      return h('span', { class: 'cell-muted' }, formatDateTime(row." + f.Name + "))\n")
			sb.WriteString("    },\n")
			sb.WriteString("  },\n")
		default:
			sb.WriteString(fmt.Sprintf("  { key: '%s', title: '%s'", f.Name, f.Label))
			for _, a := range attrs {
				sb.WriteString(", " + a)
			}
			sb.WriteString(" },\n")
		}
	}
	// 行操作列（fixed right）
	sb.WriteString("  {\n    key: 'actions',\n    title: '操作',\n    width: 140,\n    fixed: 'right',\n    render(row): VNode {\n")
	sb.WriteString("      return h(NSpace, { size: 'small' }, () => [\n")
	sb.WriteString("        h(NButton, { size: 'small', type: 'primary', text: true, 'data-testid': `" + entity + "-edit-${row.id}`, onClick: () => handleEdit(row) }, { default: () => '编辑' }),\n")
	sb.WriteString("        h(NButton, { size: 'small', type: 'error', text: true, 'data-testid': `" + entity + "-delete-${row.id}`, onClick: () => handleDelete(row) }, { default: () => '删除' }),\n")
	sb.WriteString("      ])\n    },\n  },\n")
	return sb.String()
}

// enumLabelExpr renders the enum column display expression (label map lookup).
func enumLabelExpr(f webGenField) string {
	return fmt.Sprintf("%sLabels[row.%s] ?? '—'", f.Name, f.Name)
}

// enumLabelsTS renders the enum label map constant for one field.
func enumLabelsTS(f webGenField) string {
	entries := make([]string, 0, len(f.ColumnOptions))
	for _, v := range sortedOptionKeys(f.ColumnOptions) {
		entries = append(entries, fmt.Sprintf("'%s': '%s'", v, f.ColumnOptions[v]))
	}
	return "{ " + strings.Join(entries, ", ") + " }"
}

// backendItemFieldsTS renders the PascalCase backend item interface fields
// (后端无 json tag 的 PascalCase 序列化契约，adapter 归一化用).
func backendItemFieldsTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if f.IsJSONHidden() {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s: %s\n", f.PascalName, tsTypeOf(f.DSLType)))
	}
	return sb.String()
}

// recordFieldsTS renders the snake_case frontend record interface fields.
func recordFieldsTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if f.IsJSONHidden() {
			continue
		}
		sb.WriteString(fmt.Sprintf("  %s: %s\n", f.Name, tsTypeOf(f.DSLType)))
	}
	return sb.String()
}

// formModelFieldsTS renders the form model interface fields.
func formModelFieldsTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.FormShow {
			continue
		}
		name := f.Name
		optional := ""
		if f.FormCreateOnly {
			optional = "?"
		}
		sb.WriteString(fmt.Sprintf("  %s%s: %s\n", name, optional, tsTypeOf(f.DSLType)))
	}
	return sb.String()
}

// normalizeAssignmentsTS renders record <- backend item assignments.
func normalizeAssignmentsTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if f.IsJSONHidden() {
			continue
		}
		sb.WriteString(fmt.Sprintf("    %s: raw.%s,\n", f.Name, f.PascalName))
	}
	return sb.String()
}

// recordFillAssignmentsTS renders form model <- record assignments（编辑回填）.
func recordFillAssignmentsTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.FormShow || f.FormCreateOnly {
			continue
		}
		sb.WriteString(fmt.Sprintf("      %s: detail.%s,\n", f.Name, f.Name))
	}
	return sb.String()
}

// filterValuesCode renders the reactive filter model initializer.
func filterValuesCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			sb.WriteString(fmt.Sprintf("  %s: null,\n", f.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  %s: '',\n", f.Name))
		}
	}
	return sb.String()
}

// filterLocalsCode renders per-field locals extracted from filterValues
// inside the filteredRows computed（输入字段 trim+lowercase，select 取原值）.
func filterLocalsCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			sb.WriteString(fmt.Sprintf("  const %s = filterValues.%s as number | null\n", f.Name, f.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  const %s = String(filterValues.%s ?? '').trim().toLowerCase()\n", f.Name, f.Name))
		}
	}
	return sb.String()
}

// filterGuardExpr renders the "no active filter -> return all rows" guard.
func filterGuardExpr(fields []webGenField) string {
	var active []string
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			active = append(active, fmt.Sprintf("%s == null", f.Name))
		} else {
			active = append(active, fmt.Sprintf("!%s", f.Name))
		}
	}
	if len(active) == 0 {
		return "  if (true) return rows.value\n"
	}
	return "  if (" + strings.Join(active, " && ") + ") return rows.value\n"
}

// clientFilterCode renders per-field client-side filter checks
// （契约缺口 #1：后端 /page 无搜索参数，当前页客户端过滤）.
func clientFilterCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			sb.WriteString(fmt.Sprintf("    if (%s != null && row.%s !== %s) return false\n", f.Name, f.Name, f.Name))
		} else {
			sb.WriteString(fmt.Sprintf("    if (%s && !row.%s.toLowerCase().includes(%s)) return false\n", f.Name, f.Name, f.Name))
		}
	}
	return sb.String()
}

// resetFilterCode renders the handleReset assignments for search fields.
func resetFilterCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			sb.WriteString(fmt.Sprintf("  filterValues.%s = null\n", f.Name))
		} else {
			sb.WriteString(fmt.Sprintf("  filterValues.%s = ''\n", f.Name))
		}
	}
	return sb.String()
}

// emptyModelTS renders the empty form model initializer.
func emptyModelTS(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.FormShow {
			continue
		}
		sb.WriteString(fmt.Sprintf("    %s: %s,\n", f.Name, emptyFieldValue(f)))
	}
	return sb.String()
}

func emptyFieldValue(f webGenField) string {
	switch f.FormType {
	case "select":
		if f.Default != "" {
			if _, ok := f.FormOptions[f.Default]; ok {
				return tsValueLiteral(f.Default)
			}
		}
		keys := sortedOptionKeys(f.FormOptions)
		if len(keys) > 0 {
			return tsValueLiteral(keys[0])
		}
		return "null"
	case "number":
		if f.Default != "" {
			return f.Default
		}
		return "null"
	default:
		return "''"
	}
}

// tsTypeOf maps a DSL type to its TypeScript type.
func tsTypeOf(dslType string) string {
	switch dslType {
	case "datetime", "date":
		return "string | null"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

func defaultColumnWidth(f webGenField) int {
	switch f.DSLType {
	case "datetime", "date":
		return 170
	case "string", "text":
		return 0 // 自适应
	default:
		return 110
	}
}

// filterSchemaCode renders the FormFieldSchema[] for the filter bar.
func filterSchemaCode(fields []webGenField) string {
	var sb strings.Builder
	for _, f := range fields {
		if !f.SearchShow {
			continue
		}
		if f.SearchType == "select" {
			sb.WriteString("  { key: '" + f.Name + "', label: '" + f.Label + "', type: 'select', options: " + tsOptions(f.SearchOptions) + " },\n")
		} else {
			sb.WriteString("  { key: '" + f.Name + "', label: '" + f.Label + "', type: 'input', placeholder: '" + f.Placeholder + "' },\n")
		}
	}
	return sb.String()
}

// formSchemaCode renders create/edit FormFieldSchema[] entries.
func formSchemaCode(fields []webGenField) (createTS, editTS string) {
	var create, edit strings.Builder
	for _, f := range fields {
		if !f.FormShow {
			continue
		}
		entry := formSchemaEntry(f)
		if f.FormCreateOnly {
			create.WriteString(entry)
			continue
		}
		create.WriteString(entry)
		edit.WriteString(entry)
	}
	return create.String(), edit.String()
}

func formSchemaEntry(f webGenField) string {
	var sb strings.Builder
	sb.WriteString("  {\n")
	sb.WriteString(fmt.Sprintf("    key: '%s',\n    label: '%s',\n", f.Name, f.Label))
	switch f.FormType {
	case "select":
		sb.WriteString("    type: 'select',\n")
		sb.WriteString("    options: " + tsOptions(f.FormOptions) + ",\n")
	case "textarea":
		sb.WriteString("    type: 'textarea',\n    placeholder: '" + f.Placeholder + "',\n")
	case "password":
		sb.WriteString("    type: 'custom',\n")
		sb.WriteString("    component: NInput,\n")
		sb.WriteString("    componentProps: { type: 'password', showPasswordOn: 'click', placeholder: '" + f.Placeholder + "' },\n")
	case "number":
		sb.WriteString("    type: 'number',\n    placeholder: '" + f.Placeholder + "',\n")
	default:
		sb.WriteString("    type: 'input',\n    placeholder: '" + f.Placeholder + "',\n")
	}
	if len(f.FormRules) > 0 {
		sb.WriteString("    rules: [\n")
		for _, r := range f.FormRules {
			sb.WriteString("      " + ruleEntry(r, f.Label) + ",\n")
		}
		sb.WriteString("    ],\n")
	}
	sb.WriteString("  },\n")
	return sb.String()
}

// ruleEntry renders one or more rule object literals（已含花括号），
// 供 formSchemaEntry 写入 rules: [...] 数组。
func ruleEntry(r webGenRule, label string) string {
	msg := r.Message
	if msg == "" {
		msg = "请输入" + label
	}
	parts := []string{}
	if r.Required {
		parts = append(parts, fmt.Sprintf("{ required: true, message: '%s', trigger: 'blur' }", msg))
	}
	if r.Min > 0 {
		parts = append(parts, fmt.Sprintf("{ min: %d, message: '%s', trigger: 'blur' }", r.Min, msg))
	}
	if r.Max > 0 {
		parts = append(parts, fmt.Sprintf("{ max: %d, message: '%s', trigger: 'blur' }", r.Max, msg))
	}
	if r.Pattern != "" {
		pat := r.Pattern
		if !strings.HasPrefix(pat, "/") {
			pat = "/" + pat + "/"
		}
		parts = append(parts, fmt.Sprintf("{ pattern: %s, message: '%s', trigger: 'blur' }", pat, msg))
	}
	return strings.Join(parts, ", ")
}

// tsOptions renders a TS FormFieldOption[] literal from a value->label map.
func tsOptions(m map[string]string) string {
	if len(m) == 0 {
		return "[]"
	}
	items := make([]string, 0, len(m))
	for _, v := range sortedOptionKeys(m) {
		items = append(items, fmt.Sprintf("{ label: '%s', value: %s }", m[v], tsValueLiteral(v)))
	}
	return "[ " + strings.Join(items, ", ") + " ]"
}

// sortedOptionKeys returns option keys in stable order (numeric first asc).
func sortedOptionKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// 数字枚举按数值排序，字符串按字典序
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if optionKeyGreater(keys[i], keys[j]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func optionKeyGreater(a, b string) bool {
	var ai, bi int
	_, aErr := fmt.Sscanf(a, "%d", &ai)
	_, bErr := fmt.Sscanf(b, "%d", &bi)
	if aErr == nil && bErr == nil {
		return ai > bi
	}
	return a > b
}

// tsValueLiteral renders an option value: numbers bare, strings quoted.
func tsValueLiteral(v string) string {
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
		return v
	}
	return fmt.Sprintf("'%s'", v)
}
