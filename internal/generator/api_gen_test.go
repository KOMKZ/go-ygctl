package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIGenDTOUsesSnakeCaseJSONContract(t *testing.T) {
	workspace := t.TempDir()
	appDir := filepath.Join(workspace, "apps", "hrise-admin-api")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(appDir, "go.mod"),
		[]byte("module github.com/KOMKZ/hrise-server-app/apps/hrise-admin-api\n"),
		0644,
	); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	data, err := buildDefTemplateData(workspace, filepath.Join(workspace, "domains", "admin"), &Def{
		Domain: "admin",
		Entity: "admin",
		Table:  "admins",
		Fields: []DefField{
			{Name: "real_name", Type: "string"},
			{Name: "avatar_storage_id", Type: "string"},
		},
	})
	if err != nil {
		t.Fatalf("build template data: %v", err)
	}

	outPath := filepath.Join(workspace, "apps", "hrise-admin-api", "internal", "module", "admin", "dto.go")
	if err := renderDefTemplate(apiTemplates, "templates/api", outPath, "appmodule/dto.go.tmpl", data); err != nil {
		t.Fatalf("render dto template: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated dto: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"RealName string `json:\"real_name\"`",
		"AvatarStorageID string `json:\"avatar_storage_id\"`",
		"type AdminResponse struct",
		"func toAdminResponse(item *service.AdminItem) *AdminResponse",
		"Records  []AdminResponse `json:\"records\"`",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated dto missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{
		"`json:\"RealName\"`",
		"`json:\"AvatarStorageID\"`",
		"`json:\"realName\"`",
		"`json:\"avatarStorageID\"`",
		"Records  []service.AdminItem `json:\"records\"`",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("generated dto contains forbidden contract %q:\n%s", forbidden, text)
		}
	}
}

func TestAPIGenUsesRouteGroupModuleAndBusinessFilePrefix(t *testing.T) {
	workspace := t.TempDir()
	appDir := filepath.Join(workspace, "apps", "hrise-admin-api")
	domainDir := filepath.Join(workspace, "domains", "shop")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir app: %v", err)
	}
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		t.Fatalf("mkdir domain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "go.mod"), []byte("module github.com/KOMKZ/hrise-server-app/apps/hrise-admin-api\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "go.work"), []byte("go 1.24.1\n"), 0644); err != nil {
		t.Fatalf("write go.work: %v", err)
	}

	data, err := buildDefTemplateData(workspace, domainDir, &Def{
		Domain:    "shop",
		Entity:    "attribute-set",
		Table:     "attribute_sets",
		RouteBase: "/shop/attribute-sets",
		Fields:    []DefField{{Name: "set_code", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("build template data: %v", err)
	}
	if data.AppModuleSnake != "shop" {
		t.Fatalf("AppModuleSnake = %q, want shop", data.AppModuleSnake)
	}
	if data.AppFilePrefix != "attr" {
		t.Fatalf("AppFilePrefix = %q, want attr", data.AppFilePrefix)
	}

	outPath := filepath.Join(workspace, "apps", "hrise-admin-api", "internal", "module", data.AppModuleSnake, data.AppFilePrefix+"_handler.go")
	if err := renderDefTemplate(apiTemplates, "templates/api", outPath, "appmodule/handler.go.tmpl", data); err != nil {
		t.Fatalf("render handler template: %v", err)
	}
	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read generated handler: %v", err)
	}
	text := string(content)
	if !strings.HasPrefix(text, "package shop\n") {
		t.Fatalf("generated handler package mismatch:\n%s", text)
	}
}
