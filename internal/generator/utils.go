package generator

import (
	"path/filepath"
	"strings"
	"unicode"
)

// ToPascalCase converts kebab-case or snake_case to PascalCase
// e.g., "http-demo-app" -> "HttpDemoApp"; "avatar_storage_id" -> "AvatarStorageID"
func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' || r == '_' })
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

// resolveFrameworkAbs resolves the framework path against the app directory
// for go.work replace computation. Absolute paths are kept untouched —
// filepath.Join would wrongly prefix them with appPath.
func resolveFrameworkAbs(appPath, frameworkPath string) string {
	if filepath.IsAbs(frameworkPath) {
		return frameworkPath
	}
	return filepath.Join(appPath, frameworkPath)
}
