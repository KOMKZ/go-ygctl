package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ComponentInfo 组件信息
type ComponentInfo struct {
	Name         string   // 组件名称
	Description  string   // 组件描述
	Dependencies []string // 依赖的组件
	External     string   // 外部服务依赖
	ConfigKey    string   // 配置键名
}

// 支持的组件列表
var supportedComponents = []ComponentInfo{
	{
		Name:         "database",
		Description:  "数据库组件 (MySQL/PostgreSQL/SQLite)",
		Dependencies: []string{"config", "logger"},
		External:     "MySQL/PostgreSQL/SQLite",
		ConfigKey:    "database.connections",
	},
	{
		Name:         "redis",
		Description:  "Redis 缓存组件",
		Dependencies: []string{"config", "logger"},
		External:     "Redis Server",
		ConfigKey:    "redis.instances",
	},
	{
		Name:         "jwt",
		Description:  "JWT 认证组件",
		Dependencies: []string{"config", "logger"},
		External:     "无 (可选 Redis)",
		ConfigKey:    "jwt",
	},
	{
		Name:         "kafka",
		Description:  "Kafka 消息队列组件",
		Dependencies: []string{"config", "logger"},
		External:     "Kafka Broker",
		ConfigKey:    "kafka",
	},
	{
		Name:         "telemetry",
		Description:  "OpenTelemetry 可观测性组件",
		Dependencies: []string{"config", "logger"},
		External:     "OTLP Collector",
		ConfigKey:    "telemetry",
	},
	{
		Name:         "auth",
		Description:  "认证授权组件",
		Dependencies: []string{"config", "logger"},
		External:     "无 (可选 Redis)",
		ConfigKey:    "auth",
	},
	{
		Name:         "limiter",
		Description:  "限流组件",
		Dependencies: []string{"config", "logger", "redis"},
		External:     "Redis Server",
		ConfigKey:    "limiter",
	},
	{
		Name:         "event",
		Description:  "事件总线组件",
		Dependencies: []string{"config", "logger"},
		External:     "无",
		ConfigKey:    "event",
	},
}

var componentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available components",
	Long:  `List all available Yogan Framework components with their descriptions and dependencies.`,
	Run:   runComponentList,
}

func init() {
	componentCmd.AddCommand(componentListCmd)
}

func runComponentList(cmd *cobra.Command, args []string) {
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	fmt.Println()
	bold.Println("═══════════════════════════════════════════════════════════════")
	bold.Println("                  Yogan Framework 组件列表")
	bold.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	for i, comp := range supportedComponents {
		// 组件名称和描述
		cyan.Printf("  %d. ", i+1)
		bold.Printf("%s\n", comp.Name)
		fmt.Printf("     %s\n", comp.Description)

		// 依赖
		fmt.Print("     依赖: ")
		green.Printf("%v\n", comp.Dependencies)

		// 外部服务
		fmt.Print("     外部服务: ")
		yellow.Printf("%s\n", comp.External)

		// 配置键
		fmt.Printf("     配置键: %s\n", comp.ConfigKey)
		fmt.Println()
	}

	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println("使用方法:")
	cyan.Println("  go-ygctl component add <component-name>")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go-ygctl component add database")
	fmt.Println("  go-ygctl component add redis")
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()
}

// GetComponentInfo 获取组件信息
func GetComponentInfo(name string) *ComponentInfo {
	for _, comp := range supportedComponents {
		if comp.Name == name {
			return &comp
		}
	}
	return nil
}

// GetComponentNames 获取所有组件名称
func GetComponentNames() []string {
	names := make([]string, len(supportedComponents))
	for i, comp := range supportedComponents {
		names[i] = comp.Name
	}
	return names
}
