package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// PermissionDef is a loose permission DSL file under defs/permissions/*.yaml.
type PermissionDef struct {
	Domain      string              `yaml:"domain"`
	Permissions []PermissionDefItem `yaml:"permissions"`
}

// PermissionDefItem declares one domain-owned permission.
type PermissionDefItem struct {
	Code        string          `yaml:"code"`
	Name        string          `yaml:"name"`
	Type        string          `yaml:"type"`
	Group       string          `yaml:"group"`
	Description string          `yaml:"description"`
	APIs        []PermissionAPI `yaml:"apis"`
}

// PermissionAPI declares one API resource bound to a permission.
type PermissionAPI struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

type declaredPermissionData struct {
	DomainKey   string
	RootModule  string
	Permissions []declaredPermissionItem
	Policies    []declaredPolicyItem
}

type declaredPermissionItem struct {
	Code        string
	Name        string
	Type        string
	Group       string
	Description string
}

type declaredPolicyItem struct {
	Method         string
	Path           string
	PermissionCode string
}

// GeneratePermissionDeclarations scans defs/permissions/*.yaml and regenerates
// the domain permission declaration file from the DSL.
func GeneratePermissionDeclarations(workspace, domain string, fs embed.FS, tmplRoot string) ([]string, error) {
	defs, err := LoadPermissionDefs(workspace)
	if err != nil {
		return nil, err
	}
	def, ok := defs[domain]
	if !ok || len(def.Permissions) == 0 {
		return nil, nil
	}

	return generatePermissionDeclaration(workspace, def, fs, tmplRoot)
}

// GenerateAllPermissionDeclarations regenerates declaration files for every
// permission DSL file under defs/permissions.
func GenerateAllPermissionDeclarations(workspace string, fs embed.FS, tmplRoot string) ([]string, error) {
	defs, err := LoadPermissionDefs(workspace)
	if err != nil {
		return nil, err
	}
	if len(defs) == 0 {
		return nil, nil
	}

	domains := make([]string, 0, len(defs))
	for domain := range defs {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	for _, domain := range domains {
		if _, err := generatePermissionDeclaration(workspace, defs[domain], fs, tmplRoot); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// SyncPermissionDeclarations regenerates all permission declarations from the
// workspace DSL using the built-in API templates.
func SyncPermissionDeclarations(workspace string) error {
	_, err := GenerateAllPermissionDeclarations(workspace, apiTemplates, "templates/api")
	return err
}

// SyncPermissionDeclaration regenerates one domain's permission declaration
// from the workspace DSL using the built-in API templates.
func SyncPermissionDeclaration(workspace, domain string) error {
	defs, err := LoadPermissionDefs(workspace)
	if err != nil {
		return err
	}
	if def, ok := defs[domain]; !ok || len(def.Permissions) == 0 {
		return fmt.Errorf("permission DSL for domain %q not found under defs/permissions", domain)
	}
	_, err = GeneratePermissionDeclarations(workspace, domain, apiTemplates, "templates/api")
	return err
}

func generatePermissionDeclaration(workspace string, def PermissionDef, fs embed.FS, tmplRoot string) ([]string, error) {
	outPath := filepath.Join(workspace, "domains", def.Domain, "permissions", "declared_permissions.go")
	if _, err := os.Stat(outPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	rootModule, err := deriveModuleBase(workspace)
	if err != nil {
		return nil, err
	}
	data := buildDeclaredPermissionData(rootModule, def)
	if err := renderPermissionTemplate(fs, tmplRoot, outPath, "permissions/declared_permissions.go.tmpl", data); err != nil {
		return nil, err
	}
	return nil, nil
}

// LoadPermissionDefs loads all loose permission def files keyed by domain.
func LoadPermissionDefs(workspace string) (map[string]PermissionDef, error) {
	files, err := ListPermissionDefFiles(workspace)
	if err != nil {
		return nil, err
	}

	result := make(map[string]PermissionDef, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read permission def %s: %w", file, err)
		}
		def := PermissionDef{}
		if err := yaml.Unmarshal(content, &def); err != nil {
			return nil, fmt.Errorf("invalid permission def yaml %s: %w", file, err)
		}
		if err := def.Validate(file); err != nil {
			return nil, err
		}
		result[def.Domain] = mergePermissionDef(result[def.Domain], def)
	}
	return result, nil
}

// ListPermissionDefFiles returns sorted loose permission DSL files.
func ListPermissionDefFiles(workspace string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(workspace, "defs", "permissions", "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// Validate checks the permission DSL structure.
func (d *PermissionDef) Validate(file string) error {
	if strings.TrimSpace(d.Domain) == "" {
		return fmt.Errorf("invalid permission def %s: domain is required", file)
	}
	if !domainKeyRe.MatchString(d.Domain) {
		return fmt.Errorf("invalid permission def %s: domain %q must be lowercase snake/kebab-case", file, d.Domain)
	}

	seen := map[string]bool{}
	for i := range d.Permissions {
		item := &d.Permissions[i]
		item.Code = strings.TrimSpace(item.Code)
		item.Name = strings.TrimSpace(item.Name)
		item.Type = strings.ToUpper(strings.TrimSpace(item.Type))
		item.Group = strings.TrimSpace(item.Group)
		item.Description = strings.TrimSpace(item.Description)
		if item.Code == "" || item.Name == "" || item.Type == "" {
			return fmt.Errorf("invalid permission def %s: permission code/name/type are required", file)
		}
		if seen[item.Code] {
			return fmt.Errorf("invalid permission def %s: duplicate permission code %q", file, item.Code)
		}
		seen[item.Code] = true
		if item.Group == "" {
			item.Group = "SYSTEM"
		}
		for j := range item.APIs {
			item.APIs[j].Method = strings.ToUpper(strings.TrimSpace(item.APIs[j].Method))
			item.APIs[j].Path = strings.TrimSpace(item.APIs[j].Path)
			if item.APIs[j].Method == "" || item.APIs[j].Path == "" {
				return fmt.Errorf("invalid permission def %s: api method/path are required for %s", file, item.Code)
			}
		}
	}
	return nil
}

func mergePermissionDef(left, right PermissionDef) PermissionDef {
	if left.Domain == "" {
		return right
	}
	left.Permissions = append(left.Permissions, right.Permissions...)
	return left
}

func buildDeclaredPermissionData(rootModule string, def PermissionDef) declaredPermissionData {
	items := make([]declaredPermissionItem, 0, len(def.Permissions))
	policies := make([]declaredPolicyItem, 0)
	for _, item := range def.Permissions {
		items = append(items, declaredPermissionItem{
			Code:        item.Code,
			Name:        item.Name,
			Type:        item.Type,
			Group:       item.Group,
			Description: item.Description,
		})
		for _, api := range item.APIs {
			policies = append(policies, declaredPolicyItem{
				Method:         api.Method,
				Path:           api.Path,
				PermissionCode: item.Code,
			})
		}
	}
	return declaredPermissionData{
		DomainKey:   def.Domain,
		RootModule:  rootModule,
		Permissions: items,
		Policies:    policies,
	}
}

func renderPermissionTemplate(fs embed.FS, tmplRoot, outPath, tmplName string, data declaredPermissionData) error {
	content, err := fs.ReadFile(filepath.Join(tmplRoot, tmplName))
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}
	tmpl, err := template.New(tmplName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, buf.Bytes(), 0644)
}
