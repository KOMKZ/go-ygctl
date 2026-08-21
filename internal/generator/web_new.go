package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// webNewNameRe validates the app name: lowercase kebab-case.
var webNewNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// WebNewConfig configures ygctl web new.
type WebNewConfig struct {
	AppPath        string // output directory (created if missing)
	AppName        string // kebab-case, e.g. hrise-admin-web-demo
	AppTitle       string // optional display title; default derived from name
	UILink         string // @rong/admin-ui link path, e.g. link:../rong-admin-ui
	AppPort        int    // dev server port; default 3100
	APIProxyTarget string // vite /api proxy target; default http://localhost:9201
	StoragePrefix  string // localStorage key prefix; default first segment of name
	BackendAppDir  string // startall/stopall 后端目录名; default ../hrise-admin-api
}

// WebNewResult reports the generation result.
type WebNewResult struct {
	AppPath string
	Files   int
}

// webNewData is the template data for web new (and web ensure).
type webNewData struct {
	AppName        string
	AppTitle       string
	UILink         string
	AppPort        int
	APIProxyTarget string
	StoragePrefix  string
	BackendAppDir  string
	HomePath       string
	LoginPath      string
}

// Generate renders the admin app skeleton from the golden-app templates
// (login/auth/permission skeleton + layout + dashboard + @ygctl-web-gen markers).
func (c *WebNewConfig) Generate() (*WebNewResult, error) {
	if !webNewNameRe.MatchString(c.AppName) {
		return nil, fmt.Errorf("app name %q must be lowercase kebab-case", c.AppName)
	}
	if c.AppPath == "" {
		c.AppPath = c.AppName
	}
	if _, err := os.Stat(filepath.Join(c.AppPath, "package.json")); err == nil {
		return nil, fmt.Errorf("%w: %s already contains a package.json", ErrPathExists, c.AppPath)
	}
	if c.AppTitle == "" {
		c.AppTitle = kebabToTitle(c.AppName)
	}
	if c.UILink == "" {
		return nil, fmt.Errorf("ui link path is required (--ui-link), e.g. link:../rong-admin-ui")
	}
	if c.AppPort == 0 {
		c.AppPort = 3100
	}
	if c.APIProxyTarget == "" {
		c.APIProxyTarget = "http://localhost:9201"
	}
	if c.StoragePrefix == "" {
		seg := strings.Split(c.AppName, "-")
		c.StoragePrefix = seg[0]
	}
	if c.BackendAppDir == "" {
		c.BackendAppDir = "hrise-admin-api"
	}

	data := &webNewData{
		AppName:        c.AppName,
		AppTitle:       c.AppTitle,
		UILink:         c.UILink,
		AppPort:        c.AppPort,
		APIProxyTarget: c.APIProxyTarget,
		StoragePrefix:  c.StoragePrefix,
		BackendAppDir:  c.BackendAppDir,
		HomePath:       "/dashboard",
		LoginPath:      "/login",
	}

	count := 0
	err := fs.WalkDir(webTemplates, "templates/web/new", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "templates/web/new/")
		out := filepath.Join(c.AppPath, filepath.FromSlash(rel))
		if strings.HasSuffix(rel, ".tmpl") {
			out = strings.TrimSuffix(out, ".tmpl")
			if err := renderNewTemplate(path, out, data); err != nil {
				return err
			}
		} else {
			// 非模板文件（shell 脚本等）原样复制：bash [[ ]] 与模板定界符冲突
			content, err := webTemplates.ReadFile(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(out, content, 0644); err != nil {
				return err
			}
		}
		count++
		return nil
	})
	if err != nil {
		return nil, err
	}
	// shell 脚本恢复执行位（embed FS 不保留文件模式）
	shFiles, err := filepath.Glob(filepath.Join(c.AppPath, "scripts", "*.sh"))
	if err == nil {
		for _, f := range shFiles {
			if err := os.Chmod(f, 0755); err != nil {
				return nil, fmt.Errorf("chmod %s: %w", f, err)
			}
		}
	}
	return &WebNewResult{AppPath: c.AppPath, Files: count}, nil
}

// kebabToTitle converts a kebab-case name to a title: hrise-admin-web-demo
// -> "Hrise Admin Web Demo".
func kebabToTitle(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// renderNewTemplate renders one web/new template with [[ ]] delimiters.
func renderNewTemplate(tmplPath, outPath string, data *webNewData) error {
	content, err := webTemplates.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", tmplPath, err)
	}
	tmpl, err := template.New(filepath.Base(tmplPath)).Delims("[[", "]]").Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplPath, err)
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
