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

var (
	webNewAppName string
	webNewTitle   string
	webNewUILink  string
	webNewPort    int
	webNewProxy   string
	webNewPrefix  string
	webNewBackend string
	webNewOut     string
	webNewBase    string
	webNewHome    string
	webNewLogin   string
)

var webNewCmd = &cobra.Command{
	Use:   "new <app-name>",
	Short: "Generate a complete admin web app skeleton (login + layout + markers)",
	Long: `从金样反推的骨架模板生成完整 admin 前端（确定性渲染）：

  ygctl web new hrise-admin-web-demo --ui-link link:../rong-admin-ui

包含：登录/登出/续期/守卫封装、layout、dashboard、@ygctl-web-gen marker。
生成后：pnpm install && make dev，再 ygctl web gen <entity> 生成 CRUD 模块。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.WebNewConfig{
			AppPath:        webNewOut,
			AppName:        args[0],
			AppTitle:       webNewTitle,
			UILink:         webNewUILink,
			AppPort:        webNewPort,
			APIProxyTarget: webNewProxy,
			StoragePrefix:  webNewPrefix,
			BackendAppDir:  webNewBackend,
			AppBasePath:    webNewBase,
			HomePath:       webNewHome,
			LoginPath:      webNewLogin,
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ Admin web app generated: %s（%d files）", result.AppPath, result.Files)
		fmt.Println()
		color.Yellow("Next steps:")
		fmt.Println("  cd " + result.AppPath + " && pnpm install")
		fmt.Println("  make dev-bg        # dev server")
		fmt.Println("  ygctl web gen <entity> -f defs/<entity>.yaml --app " + result.AppPath)
		return nil
	},
}

var (
	ensureAppPath    string
	ensureProject    string
	ensurePort       int
	ensureBackendApp string
)

var webEnsureCmd = &cobra.Command{
	Use:   "ensure --app <path>",
	Short: "Ensure Makefile + runtime scripts exist in an existing web app (idempotent)",
	Long: `为已有 web 应用补齐/刷新生命周期文件（参考 rong-admin-web）：

  Makefile + scripts/{app,runtime,toolchain}.sh

- 已存在且含 ygctl 标记的文件会被刷新；无标记的手写文件拒绝覆盖
- 适合存量应用（如 rong-admin-webdemo）补齐生命周期`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &generator.EnsureWebLifecycleConfig{
			AppPath:       ensureAppPath,
			Project:       ensureProject,
			AppPort:       ensurePort,
			BackendAppDir: ensureBackendApp,
		}
		result, err := cfg.Generate()
		if err != nil {
			return err
		}
		color.Green("✅ Lifecycle ensured for %s:", ensureAppPath)
		for _, f := range result.Files {
			fmt.Println("  -", f)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.AddCommand(webNewCmd)
	webCmd.AddCommand(webGenCmd)
	webCmd.AddCommand(webEnsureCmd)
	webNewCmd.Flags().StringVar(&webNewOut, "out", "", "Output directory (default: current dir/<app-name>)")
	webNewCmd.Flags().StringVar(&webNewTitle, "title", "", "Display title (default: derived from app name)")
	webNewCmd.Flags().StringVar(&webNewUILink, "ui-link", "", "UI framework link path (required), e.g. link:../rong-admin-ui")
	webNewCmd.Flags().IntVar(&webNewPort, "port", 0, "Dev server port (default 3100)")
	webNewCmd.Flags().StringVar(&webNewProxy, "api-proxy", "", "Vite /api proxy target (default http://localhost:9201)")
	webNewCmd.Flags().StringVar(&webNewPrefix, "storage-prefix", "", "localStorage key prefix (default: first segment of app name)")
	webNewCmd.Flags().StringVar(&webNewBackend, "backend-app", "", "startall/stopall backend dir name (default hrise-admin-api)")
	webNewCmd.Flags().StringVar(&webNewBase, "app-base-path", "", "Authenticated layout base path (default /)")
	webNewCmd.Flags().StringVar(&webNewHome, "home-path", "", "Authenticated home path (default /dashboard)")
	webNewCmd.Flags().StringVar(&webNewLogin, "login-path", "", "Login route path (default /login)")
	webEnsureCmd.Flags().StringVar(&ensureAppPath, "app", "", "Existing web app root (required)")
	webEnsureCmd.Flags().StringVar(&ensureProject, "project", "", "Makefile PROJECT (default: base name of app path)")
	webEnsureCmd.Flags().IntVar(&ensurePort, "port", 0, "Dev server port (default 3100)")
	webEnsureCmd.Flags().StringVar(&ensureBackendApp, "backend-app", "", "startall/stopall backend dir name (default hrise-admin-api)")
	webGenCmd.Flags().StringVar(&webAppPath, "app", "", "Target admin app root (contains src/)")
	webGenCmd.Flags().StringVarP(&webDefFile, "file", "f", "", "Backend def file (defs/<entity>.yaml)")
	webGenCmd.Flags().StringVar(&webUIFile, "ui", "", "Frontend ui def file (default: defs/<entity>.ui.yaml)")
	webGenCmd.Flags().StringVar(&webStyle, "style", "", "Style override (dialog | page; default: ui def style)")
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
