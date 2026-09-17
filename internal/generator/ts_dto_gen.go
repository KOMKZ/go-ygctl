// ts_dto_gen: Go struct → TypeScript interface 生成器。
//
// 上游入口：cmd/ts_dto.go（ygctl ts-dto）。
// 下游依赖：go/parser + go/ast 解析 Go 源文件；纯 stdlib，无外部依赖。
// 维护：加新类型映射时改 tsTypeForAST；Go → TS 命名规则改 tsIdent。
package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// TsDtoGenConfig 生成器配置（exported，cmd 包可调用）。
type TsDtoGenConfig struct {
	WorkspaceRoot string
	DomainKey     string
	OutputDir     string
	TSPackageName string
	Log           *log.Logger
}

// tsStructInfo 单个 struct 的解析结果。
type tsStructInfo struct {
	Name    string
	Fields  []tsField
	FileRef string // 来源文件相对路径（用于注释）
}

// generated_tsPackage 整个 Go package 的输出。
type tsGeneratedPackage struct {
	PackageName string
	Structs     []tsStructInfo
}

// generated_tsField 字段 + TS 类型映射。
type tsField struct {
	GoName   string
	TSName   string
	TSOptional bool
	TSType   string
}

// Generate 扫描 domain 的 service/ + read/model/ 子包生成 TypeScript contract.ts。
func (c *TsDtoGenConfig) Generate() error {
	pkgs, err := c.scanPackages()
	if err != nil {
		return err
	}
	merged := c.mergePackages(pkgs)
	output := c.renderTS(merged)
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		return fmt.Errorf("mkdir output: %w", err)
	}
	outPath := filepath.Join(c.OutputDir, c.TSPackageName+".ts")
	return os.WriteFile(outPath, output, 0644)
}

// scanPackages 扫描 domain 下的 service/ 和 read/model/ 子包。
func (c *TsDtoGenConfig) scanPackages() (map[string]*tsGeneratedPackage, error) {
	result := make(map[string]*tsGeneratedPackage)
	subPkgs := []string{"service", "read/model"}
	for _, sub := range subPkgs {
		dir := filepath.Join(c.WorkspaceRoot, "domains", c.DomainKey, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
		pkgName := filepath.Base(sub) // service / model
		pkg := &tsGeneratedPackage{PackageName: pkgName}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			structs, err := c.parseStructsFromFile(filepath.Join(dir, e.Name()), pkgName)
			if err != nil {
				c.Log.Printf("parse %s: %v", e.Name(), err)
				continue
			}
			pkg.Structs = append(pkg.Structs, structs...)
		}
		if len(pkg.Structs) > 0 {
			sort.Slice(pkg.Structs, func(i, j int) bool {
				return pkg.Structs[i].Name < pkg.Structs[j].Name
			})
			result[pkgName] = pkg
		}
	}
	return result, nil
}

// parseStructsFromFile 解析单个 Go 源文件中的所有 exported struct。
func (c *TsDtoGenConfig) parseStructsFromFile(path, pkgName string) ([]tsStructInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	rel, _ := filepath.Rel(c.WorkspaceRoot, path)
	var out []tsStructInfo
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if !ast.IsExported(ts.Name.Name) {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			info := tsStructInfo{
				Name:    ts.Name.Name,
				FileRef: rel,
				Fields:  c.fieldsFromStruct(st),
			}
			if len(info.Fields) > 0 {
				out = append(out, info)
			}
		}
	}
	return out, nil
}

// fieldsFromStruct 提取 struct 的所有字段（含 json tag 映射）。
func (c *TsDtoGenConfig) fieldsFromStruct(st *ast.StructType) []tsField {
	var out []tsField
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue // embedded
		}
		tsType, isOptional := c.tsTypeForExpr(f.Type)
		jsonTag := extractJSONTag(f.Tag)
		for _, name := range f.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			tsName := toCamelTS(name.Name)
			if jsonTag != nil {
				if jsonTag.name != "" {
					tsName = jsonTag.name
				}
				if jsonTag.omitEmpty || jsonTag.omitempty {
					isOptional = true
				}
			}
			out = append(out, tsField{
				GoName:    name.Name,
				TSName:    tsName,
				TSType:    tsType,
				TSOptional: isOptional,
			})
		}
	}
	return out
}

// tsTypeForExpr Go AST expr → TypeScript 类型字符串。
// 支持：基础类型 / 指针（optional）/ 切片 / map / 命名 struct（递归）。
func (c *TsDtoGenConfig) tsTypeForExpr(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return "string", false
		case "bool":
			return "boolean", false
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64":
			return "number", false
		case "byte":
			return "number", false
		case "Time":
			// time.Time → ISO 8601 RFC3339 字符串
			return "string", false
		}
		// 其他 ident 当作 interface 引用；保留原名（首字母大写按惯例）
		return t.Name, false
	case *ast.StarExpr:
		inner, _ := c.tsTypeForExpr(t.X)
		return inner, true
	case *ast.ArrayType:
		// 注意：array len 为 nil 表示 slice）
		inner, _ := c.tsTypeForExpr(t.Elt)
		return inner + "[]", false
	case *ast.MapType:
		k, _ := c.tsTypeForExpr(t.Key)
		v, _ := c.tsTypeForExpr(t.Value)
		return fmt.Sprintf("{ [k: %s]: %s }", k, v), false
	case *ast.SelectorExpr:
		// pkg.Type → 简化为 Type（命名空间由 TS namespace 承担）
		if x, ok := t.X.(*ast.Ident); ok {
			// time.Time → string (RFC3339 ISO 8601)
			if x.Name == "time" && t.Sel.Name == "Time" {
				return "string", false
			}
			return x.Name + "." + t.Sel.Name, false
		}
		return t.Sel.Name, false
	}
	return "unknown", false
}

// mergePackages 合并多个 package 的 structs；按 package 分组。
func (c *TsDtoGenConfig) mergePackages(pkgs map[string]*tsGeneratedPackage) []*tsGeneratedPackage {
	keys := make([]string, 0, len(pkgs))
	for k := range pkgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*tsGeneratedPackage, 0, len(keys))
	for _, k := range keys {
		out = append(out, pkgs[k])
	}
	return out
}

// renderTS 把 merged packages 输出为 TypeScript 文件。
func (c *TsDtoGenConfig) renderTS(pkgs []*tsGeneratedPackage) []byte {
	var buf bytes.Buffer
	buf.WriteString("// AUTOGENERATED FILE — DO NOT EDIT.\n")
	buf.WriteString("// Generated by: ygctl gen ts-dto --domain " + c.DomainKey + "\n")
	buf.WriteString("// Source: domains/" + c.DomainKey + "/{service,read/model}/*.go\n\n")
	for _, pkg := range pkgs {
		buf.WriteString("export namespace " + pkg.PackageName + " {\n")
		for _, s := range pkg.Structs {
			buf.WriteString(fmt.Sprintf("  /** %s (from %s) */\n", s.Name, s.FileRef))
			buf.WriteString(fmt.Sprintf("  export interface %s {\n", s.Name))
			for _, f := range s.Fields {
				opt := ""
				if f.TSOptional {
					opt = "?"
				}
				buf.WriteString(fmt.Sprintf("    %s%s: %s;\n", f.TSName, opt, f.TSType))
			}
			buf.WriteString("  }\n\n")
		}
		buf.WriteString("}\n\n")
	}
	return buf.Bytes()
}

// ---------- JSON tag 解析 ----------

type jsonTagInfo struct {
	name       string
	omitempty  bool
	omitEmpty  bool
}

var jsonTagRe = regexp.MustCompile(`json:"([^"]*)"`)

// extractJSONTag 从 ast.Field.Tag 提取 json tag 信息。
func extractJSONTag(tag *ast.BasicLit) *jsonTagInfo {
	if tag == nil {
		return nil
	}
	m := jsonTagRe.FindStringSubmatch(tag.Value)
	if len(m) < 2 {
		return nil
	}
	parts := strings.Split(m[1], ",")
	info := &jsonTagInfo{name: parts[0]}
	for _, opt := range parts[1:] {
		switch opt {
		case "omitempty":
			info.omitempty = true
		case "omitzero":
			info.omitEmpty = true
		}
	}
	if info.omitempty {
		info.omitEmpty = true
	}
	return info
}

// ---------- 命名转换 ----------

// toCamelTS 把 Go PascalCase 转为 TypeScript camelCase。
func toCamelTS(goName string) string {
	if goName == "" {
		return goName
	}
	first := strings.ToLower(string(goName[0]))
	return first + goName[1:]
}