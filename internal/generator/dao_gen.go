package generator

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

//go:embed all:templates/dao
var daoTemplates embed.FS

// DAOGenConfig holds the configuration for DAO layer generation.
type DAOGenConfig struct {
	WorkspacePath string
	DefFile       string
}

// DAOGenResult reports the DAO generation result.
type DAOGenResult struct {
	DomainDir string
}

// defFieldData is a render-ready field.
type defFieldData struct {
	Name            string
	PascalName      string
	GoType          string
	StructTag       string // complete backtick-delimited tag (gorm + json)
	Comment         string
	Unique          bool
	MySQLCol        string   // MySQL column type for the migration skeleton
	ValidationRules []string // ozzo-validation expressions (may be empty)
}

// defQueryData is a render-ready custom query.
type defQueryData struct {
	MethodName string
	ParamName  string
	ParamType  string
	ColumnName string
}

// defTemplateData is the template context for dao/api generation.
type defTemplateData struct {
	DomainKey    string
	DomainSnake  string
	DomainPascal string
	EntityName   string
	EntityPascal string
	EntitySnake  string
	TableName    string
	DomainImport string
	ErrModule    int
	NeedsTimePkg bool
	Fields       []defFieldData
	UniqueFields []defFieldData
	Queries      []defQueryData
	Endpoints    []EndpointDef
	RootModule   string
	AppModule    string // target app module path (api gen only)
	AppName      string // target app name (api gen only)
	HasEmailRule bool   // any field declares the email validation rule
	HasValidate  bool   // any request DTO carries validation rules
}

// Generate runs the DAO layer generation from the def.
func (c *DAOGenConfig) Generate() (*DAOGenResult, error) {
	def, err := LoadDef(c.DefFile)
	if err != nil {
		return nil, err
	}
	workspace, err := resolveWorkspacePath(c.WorkspacePath)
	if err != nil {
		return nil, err
	}

	domainDir := filepath.Join(workspace, "domains", def.Domain)
	// Initialize the domain skeleton if missing (table-backed domain).
	if _, err := os.Stat(filepath.Join(domainDir, "go.mod")); err != nil {
		initGen := NewDomainGenerator(&DomainConfig{
			DomainKey:     def.Domain,
			EntityName:    def.Entity,
			WorkspacePath: workspace,
			Pure:          false,
		})
		if _, err := initGen.Generate(); err != nil {
			return nil, err
		}
	}

	data, err := buildDefTemplateData(workspace, domainDir, def)
	if err != nil {
		return nil, err
	}

	files := []struct {
		tmpl string
		out  string
	}{
		{"model/model.go.tmpl", "model/" + data.EntitySnake + ".go"},
		{"errors/errors.go.tmpl", "errors/errors.go"},
		{"repository/repository.go.tmpl", "repository/repository.go"},
		{"repository/repository_mysql.go.tmpl", "repository/repository_mysql.go"},
	}

	var goFiles []string
	for _, f := range files {
		outPath := filepath.Join(domainDir, f.out)
		if err := renderDefTemplate(daoTemplates, "templates/dao", outPath, f.tmpl, data); err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", f.out, err)
		}
		goFiles = append(goFiles, outPath)
	}

	// gofmt + tidy + build verification.
	gofmtArgs := append([]string{"-w"}, goFiles...)
	if out, err := exec.Command("gofmt", gofmtArgs...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gofmt failed: %v\n%s", err, out)
	}
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = domainDir
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %v\n%s", err, out)
	}
	buildCmd := exec.Command("go", "build", "./domains/"+def.Domain+"/...")
	buildCmd.Dir = workspace
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go build failed: %v\n%s", err, out)
	}

	return &DAOGenResult{DomainDir: domainDir}, nil
}

func resolveWorkspacePath(workspacePath string) (string, error) {
	if workspacePath != "" {
		return filepath.Abs(workspacePath)
	}
	return FindWorkspaceRoot("")
}

// buildDefTemplateData converts a Def into render-ready template data.
func buildDefTemplateData(workspace, domainDir string, def *Def) (*defTemplateData, error) {
	rootModule, err := deriveModuleBase(workspace)
	if err != nil {
		return nil, err
	}

	entityPascal := ToPascalCase(def.Entity)
	entitySnake := ToSnakeCase(entityPascal)

	data := &defTemplateData{
		DomainKey:    def.Domain,
		DomainSnake:  strings.ReplaceAll(def.Domain, "-", "_"),
		DomainPascal: ToPascalCase(def.Domain),
		EntityName:   def.Entity,
		EntityPascal: entityPascal,
		EntitySnake:  entitySnake,
		TableName:    def.Table,
		DomainImport: rootModule + "/domains/" + def.Domain,
		RootModule:   rootModule,
		Endpoints:    def.Endpoints,
		ErrModule:    readDomainErrModule(domainDir),
	}

	needsTime := false
	for _, f := range def.Fields {
		if dslTypes[f.Type].NeedsTimePkg {
			needsTime = true
		}
		fd := defFieldData{
			Name:            f.Name,
			PascalName:      GoFieldName(f.Name),
			GoType:          FieldGoType(f),
			StructTag:       FieldStructTag(f),
			Comment:         f.Comment,
			Unique:          f.Unique,
			MySQLCol:        MySQLColumnType(f),
			ValidationRules: ValidationRules(f),
		}
		data.Fields = append(data.Fields, fd)
		if f.Unique {
			data.UniqueFields = append(data.UniqueFields, fd)
		}
		if len(fd.ValidationRules) > 0 {
			data.HasValidate = true
			for _, r := range fd.ValidationRules {
				if strings.Contains(r, "is.Email") {
					data.HasEmailRule = true
				}
			}
		}
	}
	data.NeedsTimePkg = needsTime

	for _, q := range def.Queries {
		col := strings.TrimPrefix(q, "by_")
		for _, fd := range data.Fields {
			if fd.Name == col {
				data.Queries = append(data.Queries, defQueryData{
					MethodName: "FindBy" + fd.PascalName,
					ParamName:  col,
					ParamType:  fd.GoType,
					ColumnName: col,
				})
				break
			}
		}
	}
	return data, nil
}

var errModuleDeclRe = regexp.MustCompile(`const\s+Module\w+\s*=\s*(\d+)`)

// readDomainErrModule returns the domain's current module number, or allocates
// a new one if the domain has no errors declared yet.
func readDomainErrModule(domainDir string) int {
	matches, _ := filepath.Glob(filepath.Join(domainDir, "errors", "*.go"))
	for _, f := range matches {
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if mm := errModuleDeclRe.FindSubmatch(content); mm != nil {
			if n, err := strconv.Atoi(string(mm[1])); err == nil {
				return n
			}
		}
	}
	return allocateErrModule(filepath.Dir(domainDir))
}

// renderDefTemplate renders one embedded def-family template.
func renderDefTemplate(fs embed.FS, tmplRoot, outPath, tmplName string, data *defTemplateData) error {
	tmplPath := filepath.Join(tmplRoot, tmplName)
	content, err := fs.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplName, err)
	}

	funcMap := template.FuncMap{
		"inc": func(base, i int) int { return base + i },
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
	}
	tmpl, err := template.New(tmplName).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplName, err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

// deriveModuleBase reuses the domain generator's base resolution logic.
func deriveModuleBase(workspace string) (string, error) {
	g := &DomainGenerator{config: &DomainConfig{WorkspacePath: workspace}}
	if err := g.loadModuleBase(); err != nil {
		return "", err
	}
	return g.rootModule, nil
}
