// gen ts-dto: 扫描 Go struct（带 json tag）→ 输出 TypeScript interface。
// 用于前后端契约同步：后端改 DTO → 重跑命令 → 前端 TS 自动更新 → CI 校验漂移。
//
// 上游入口：微服务 / 小程序 / 管理后台 各自在 build 前跑 gen ts-dto。
// 下游依赖：Go AST 解析（go/parser + go/ast）+ JSON tag 解析。
// 维护：加新结构 / 新字段同步重跑本命令；CI 必须卡 0 drift。
//
// TypeScript 端约定：
//   - 时间用 `string`（RFC3339），前端用 dayjs 解析
//   - 二进制用 `string`（base64）
//   - nullable 字段加 `?`
//   - 数组用 `T[]`
//   - 嵌套对象用对应 interface 名
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/spf13/cobra"
)

var (
	tsDtoDomain    string
	tsDtoWorkspace string
	tsDtoOutput    string
	tsDtoPackage   string
)

var tsDtoCmd = &cobra.Command{
	Use:   "ts-dto",
	Short: "Generate TypeScript interfaces from Go DTO structs",
	Long: `Scan a Go package for exported structs with json tags and emit TypeScript interfaces.

Scans:
  <workspace>/domains/<domain>/{service,read/model}/**/*.go (recursive)

Generated output:
  <output>/contract.ts (TypeScript module with namespace per Go package)

Field type mapping (Go → TS):
  string        → string
  bool          → boolean
  int / int8... → number
  uint / uint8.. → number
  int64 / uint64 → number  (frontend 用 number 表达，业务层处理精度)
  float32/64    → number
  []byte        → string (base64)
  time.Time     → string (ISO 8601 RFC3339)
  *T (pointer)  → T | null (optional field)
  []T           → T[] (array)
  map[K]V       → { [k: K]: V }  (Record)
  other struct  → other interface name

Example:
  ygctl gen ts-dto --domain content-creation-v3 \
    --output ../hrise-user-micro-app/src/features/content-creation-v3 \
    --package service`,
	Args: cobra.NoArgs,
	RunE: runGenTsDto,
}

func init() {
	rootCmd.AddCommand(tsDtoCmd)
	tsDtoCmd.Flags().StringVar(&tsDtoDomain, "domain", "", "Domain key (e.g. content-creation-v3), required")
	tsDtoCmd.Flags().StringVar(&tsDtoWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	tsDtoCmd.Flags().StringVar(&tsDtoOutput, "output", "", "Output directory (required)")
	tsDtoCmd.Flags().StringVar(&tsDtoPackage, "package", "contract", "TypeScript namespace prefix (default: contract)")
}

func runGenTsDto(cmd *cobra.Command, _ []string) error {
	if tsDtoDomain == "" {
		return fmt.Errorf("--domain is required")
	}
	if tsDtoOutput == "" {
		return fmt.Errorf("--output is required")
	}
	workspace, err := findWorkspaceRoot(tsDtoWorkspace)
	if err != nil {
		return err
	}
	cfg := &generator.TsDtoGenConfig{
		WorkspaceRoot: workspace,
		DomainKey:     tsDtoDomain,
		OutputDir:     tsDtoOutput,
		TSPackageName: tsDtoPackage,
		Log:           log.New(os.Stderr, "[ygctl] ", log.LstdFlags),
	}
	if err := cfg.Generate(); err != nil {
		return err
	}
	fmt.Printf("✅ TS types generated: %s\n", filepath.Join(tsDtoOutput, tsDtoPackage+".ts"))
	return nil
}

// findWorkspaceRoot 解析 workspace 根（含 go.work）；沿用让提供的 `--workspace` 或向上搜。
func findWorkspaceRoot(p string) (string, error) {
	if p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(abs, "go.work")); err != nil {
			return "", fmt.Errorf("workspace %s has no go.work: %w", abs, err)
		}
		return abs, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.work found upward from %s", dir)
		}
		dir = parent
	}
}

// _ 引用字符串 utils。
var _ = strings.TrimSpace