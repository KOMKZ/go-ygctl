package component

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.md.tmpl
var templateFS embed.FS

// DocGenerator 文档生成器
type DocGenerator struct {
	componentName string
	config        map[string]interface{}
	outputDir     string
}

// NewDocGenerator 创建文档生成器
func NewDocGenerator(componentName string, config map[string]interface{}, outputDir string) *DocGenerator {
	return &DocGenerator{
		componentName: componentName,
		config:        config,
		outputDir:     outputDir,
	}
}

// Generate 生成文档
func (g *DocGenerator) Generate() error {
	// 创建输出目录
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 读取模板
	templatePath := fmt.Sprintf("templates/%s.md.tmpl", g.componentName)
	tmplContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("读取模板失败: %w", err)
	}

	// 解析模板
	tmpl, err := template.New(g.componentName).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}

	// 创建输出文件
	outputFile := filepath.Join(g.outputDir, fmt.Sprintf("%s-integration-guide.md", g.componentName))
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	// 准备模板数据
	data := map[string]interface{}{
		"ComponentName": g.componentName,
		"Config":        g.config,
	}

	// 执行模板
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("执行模板失败: %w", err)
	}

	return nil
}
