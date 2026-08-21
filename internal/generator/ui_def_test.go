package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUIDef_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.ui.yaml")
	content := `menu:
  title: 管理员管理
  icon: Users
fields:
  username:
    column: { show: true, width: 200 }
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	ui, err := LoadUIDef(path)
	if err != nil {
		t.Fatalf("LoadUIDef: %v", err)
	}
	if ui.Style != DefaultUIStyle {
		t.Errorf("default style = %q, want %q", ui.Style, DefaultUIStyle)
	}
	if ui.Menu.Title != "管理员管理" {
		t.Errorf("menu title = %q", ui.Menu.Title)
	}
	if ui.Fields["username"].Column == nil || ui.Fields["username"].Column.Width != 200 {
		t.Errorf("column override not parsed: %+v", ui.Fields["username"].Column)
	}
}

func TestLoadUIDef_UnknownStyle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.ui.yaml")
	content := `style: wizard
menu: { title: 管理员管理, icon: Users }
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadUIDef(path)
	if err == nil {
		t.Fatal("expected error for unknown style")
	}
	if !strings.Contains(err.Error(), "unknown style") {
		t.Errorf("error = %v, want unknown style message", err)
	}
}

func TestLoadUIDef_MenuRequired(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"missing title", "menu: { icon: Users }\n", "menu.title is required"},
		{"missing icon", "menu: { title: 管理员 }\n", "menu.icon is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".ui.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadUIDef(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestUIDef_ValidateAgainstDef(t *testing.T) {
	def := &Def{
		Domain: "admin",
		Entity: "admin",
		Table:  "admins",
		Fields: []DefField{{Name: "username", Type: "string"}},
	}
	ui := &UIDef{
		Style: "dialog",
		Menu:  UIMenu{Title: "管理员", Icon: "Users"},
		Fields: map[string]UIField{
			"username": {},
			"nope":     {},
		},
	}
	err := ui.ValidateAgainstDef(def)
	if err == nil || !strings.Contains(err.Error(), `ui field "nope" does not exist`) {
		t.Errorf("error = %v, want unknown reference message", err)
	}
}

func TestUIDefPathFor(t *testing.T) {
	got := UIDefPathFor("defs/admin.yaml")
	if got != filepath.Join("defs", "admin.ui.yaml") {
		t.Errorf("UIDefPathFor = %q, want defs/admin.ui.yaml", got)
	}
}
