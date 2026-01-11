package generator

import "strings"

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
