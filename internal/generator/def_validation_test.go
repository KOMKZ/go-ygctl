package generator

import (
	"strings"
	"testing"
)

// TestValidationRules_EmailUsesEmailFormat 覆盖生成器根因修复：
// is.Email 对常见地址误拒，后端统一使用 is.EmailFormat。
func TestValidationRules_EmailUsesEmailFormat(t *testing.T) {
	f := DefField{Name: "email", Type: "string", Validate: "email"}
	rules := ValidationRules(f)
	if !containsRule(rules, "is.EmailFormat") {
		t.Errorf("rules = %v, want is.EmailFormat", rules)
	}
}

// TestValidationRules_InTypedLiterals 覆盖生成器根因修复：
// in: 字面量必须按字段 Go 类型生成（int8/int16 带转换），
// 否则 ozzo In 与字段值 reflect.DeepEqual 永远不匹配（创建/更新 100% 校验失败）。
func TestValidationRules_InTypedLiterals(t *testing.T) {
	cases := []struct {
		name    string
		field   DefField
		wantIn  string
	}{
		{
			name:   "int8 typed literals",
			field:  DefField{Name: "role", Type: "int8", Validate: "in:1|2"},
			wantIn: "validation.In(int8(1), int8(2))",
		},
		{
			name:   "int16 typed literals",
			field:  DefField{Name: "level", Type: "int16", Validate: "in:1|2"},
			wantIn: "validation.In(int16(1), int16(2))",
		},
		{
			name:   "string quoted literals",
			field:  DefField{Name: "status", Type: "string", Validate: "in:on|off"},
			wantIn: `validation.In("on", "off")`,
		},
		{
			name:   "int untyped literals",
			field:  DefField{Name: "kind", Type: "int", Validate: "in:1|2"},
			wantIn: "validation.In(1, 2)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules := ValidationRules(tc.field)
			if !containsRule(rules, tc.wantIn) {
				t.Errorf("rules = %v, want %q", rules, tc.wantIn)
			}
		})
	}
}

func containsRule(rules []string, want string) bool {
	for _, r := range rules {
		if strings.Contains(r, want) {
			return true
		}
	}
	return false
}
