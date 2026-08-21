package generator

import (
	"strings"
	"unicode"
)

// ToPascalCase converts kebab-case to PascalCase
// e.g., "http-demo-app" -> "HttpDemoApp"
func ToPascalCase(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// ToSnakeCase converts PascalCase or camelCase to snake_case
// e.g., "PaymentService" -> "payment_service"
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
