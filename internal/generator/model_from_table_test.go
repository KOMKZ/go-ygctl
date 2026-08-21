package generator

import (
	"database/sql"
	"strings"
	"testing"
)

func TestFieldName(t *testing.T) {
	cases := map[string]string{
		"id":            "ID",
		"user_id":       "UserID",
		"avatar_url":    "AvatarURL",
		"created_at":    "CreatedAt",
		"is_active":     "IsActive",
		"phone":         "Phone",
		"a_b_c":         "ABC",
		"mcp_server_id": "MCPServerID",
	}
	for in, want := range cases {
		if got := fieldName(in); got != want {
			t.Errorf("fieldName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoType(t *testing.T) {
	cases := []struct {
		col  TableColumn
		want string
	}{
		{TableColumn{DataType: "bigint", ColumnType: "bigint unsigned"}, "uint64"},
		{TableColumn{DataType: "bigint", ColumnType: "bigint"}, "int64"},
		{TableColumn{DataType: "int", ColumnType: "int unsigned"}, "uint32"},
		{TableColumn{DataType: "int", ColumnType: "int"}, "int"},
		{TableColumn{DataType: "tinyint", ColumnType: "tinyint"}, "int8"},
		{TableColumn{DataType: "tinyint", ColumnType: "tinyint(1)"}, "bool"},
		{TableColumn{DataType: "tinyint", ColumnType: "tinyint unsigned"}, "uint8"},
		{TableColumn{DataType: "smallint", ColumnType: "smallint"}, "int16"},
		{TableColumn{DataType: "varchar", ColumnType: "varchar(50)"}, "string"},
		{TableColumn{DataType: "json", ColumnType: "json"}, "string"},
		{TableColumn{DataType: "decimal", ColumnType: "decimal(10,2)"}, "float64"},
		{TableColumn{DataType: "datetime", ColumnType: "datetime"}, "time.Time"},
		{TableColumn{DataType: "blob", ColumnType: "longblob"}, "[]byte"},
	}
	for _, c := range cases {
		if got := goType(c.col); got != c.want {
			t.Errorf("goType(%+v) = %q, want %q", c.col, got, c.want)
		}
	}
}

func TestGormTags(t *testing.T) {
	cases := []struct {
		col  TableColumn
		want string
	}{
		{TableColumn{ColumnKey: "PRI"}, `gorm:"primarykey" `},
		{TableColumn{ColumnKey: "UNI"}, `gorm:"uniqueIndex" `},
		{TableColumn{DataType: "varchar", MaxLength: sql.NullInt64{Int64: 50, Valid: true}}, `gorm:"size:50" `},
		{TableColumn{IsNullable: "NO"}, `gorm:"not null" `},
		{TableColumn{ColumnDefault: sql.NullString{String: "1", Valid: true}}, `gorm:"default:1" `},
		{TableColumn{ColumnDefault: sql.NullString{String: "CURRENT_TIMESTAMP", Valid: true}}, ""},
		{TableColumn{IsNullable: "NO", ColumnKey: "PRI"}, `gorm:"primarykey" `},
		{TableColumn{}, ""},
	}
	for _, c := range cases {
		if got := gormTags(c.col); got != c.want {
			t.Errorf("gormTags(%+v) = %q, want %q", c.col, got, c.want)
		}
	}
}

func TestRenderModel(t *testing.T) {
	cols := []TableColumn{
		{Name: "id", DataType: "bigint", ColumnType: "bigint unsigned", IsNullable: "NO", ColumnKey: "PRI"},
		{Name: "username", DataType: "varchar", ColumnType: "varchar(50)", IsNullable: "NO", ColumnKey: "UNI", MaxLength: sql.NullInt64{Int64: 50, Valid: true}},
		{Name: "last_login_at", DataType: "datetime", ColumnType: "datetime", IsNullable: "YES"},
	}
	src := renderModel("admins", cols)

	checks := []string{
		"type Admins struct {",
		"ID uint64 `gorm:\"primarykey\" json:\"id\"`",
		"Username string `gorm:\"uniqueIndex;size:50;not null\" json:\"username\"`",
		"LastLoginAt *time.Time `json:\"last_login_at\"`",
		`func (Admins) TableName() string {`,
		`return "admins"`,
	}
	for _, want := range checks {
		if !strings.Contains(src, want) {
			t.Errorf("renderModel output missing %q\n---\n%s", want, src)
		}
	}
}
