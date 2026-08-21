package cmd

import (
	"fmt"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	webAppPath string
	webDefFile string
	webUIFile  string
	webStyle   string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Generate frontend admin CRUD modules from defs (DSL)",
	Long: `前端 CRUD 模块生成（金样反推模板，确定性渲染，无 AI）：

  web gen <entity> -f defs/<entity>.yaml  生成 CRUD 模块到前端应用

DSL 分文件约定：
  defs/<entity>.yaml     后端 def（api gen / web gen 共用）
  defs/<entity>.ui.yaml  前端呈现 def（菜单/列/搜索/表单/风格），api gen 不消费

CRUD 模板多风格（默认 dialog，可扩展）：
  dialog  列表页 + 新建/编辑弹窗（RFilterBarPro + RDataTable + RCrudFormDialog）
  page    列表页 + 独立新建/编辑页面路由（create/edit）`,
}

var webGenCmd = &cobra.Command{
	Use:   "gen <entity> -f <def>",
	Short: "Generate a CRUD module (api + views + route + menu) into a web app",
	Long: `在应用内生成 src/api/<entity>.ts、src/views/<entity>/（按风格）并注入路由/菜单片段。

- ui def 自动发现：defs/<entity>.ui.yaml（缺失时默认规则兜底）
- --style 覆盖 ui def 中的 style（dialog | page）
- 路由/菜单注入幂等：应用需预置 @ygctl-web-gen marker（web new 模板自带）
- 重复运行覆盖生成文件为相同内容（确定性渲染）`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.WebGenConfig{
			AppPath: webAppPath,
			DefFile: webDefFile,
			UIFile:  webUIFile,
			Style:   webStyle,
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ CRUD 模块已生成（entity=%s style=%s）", result.Entity, result.Style)
		for _, f := range result.Files {
			fmt.Println("  -", f)
		}
		if result.RouteInjected {
			fmt.Println("  - src/router/index.ts（路由片段已注入）")
		}
		if result.IconInjected {
			fmt.Println("  - src/layouts/components/SidebarMenu.vue（图标片段已注入）")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.AddCommand(webGenCmd)
	webGenCmd.Flags().StringVar(&webAppPath, "app", "", "Target admin app root (contains src/)")
	webGenCmd.Flags().StringVarP(&webDefFile, "file", "f", "", "Backend def file (defs/<entity>.yaml)")
	webGenCmd.Flags().StringVar(&webUIFile, "ui", "", "Frontend ui def file (default: defs/<entity>.ui.yaml)")
	webGenCmd.Flags().StringVar(&webStyle, "style", "", "Style override (dialog | page; default: ui def style)")
}
