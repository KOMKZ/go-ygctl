package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/KOMKZ/go-ygctl/internal/component"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var componentAddCmd = &cobra.Command{
	Use:   "add [component-name]",
	Short: "Generate integration guide for a component",
	Long: `Generate a step-by-step integration guide document for a Yogan Framework component.

This command will:
1. Ask you some questions about your setup
2. Generate a detailed integration guide document
3. Include code examples, configuration samples, and troubleshooting tips

The guide is beginner-friendly and walks you through each step.

Example:
  go-ygctl component add database
  go-ygctl component add redis
  go-ygctl component add jwt`,
	Args: cobra.MaximumNArgs(1),
	Run:  runComponentAdd,
}

func init() {
	componentCmd.AddCommand(componentAddCmd)
}

func runComponentAdd(cmd *cobra.Command, args []string) {
	var componentName string

	// 如果没有提供组件名，交互式选择
	if len(args) == 0 {
		prompt := &survey.Select{
			Message: "请选择要添加的组件:",
			Options: GetComponentNames(),
		}
		if err := survey.AskOne(prompt, &componentName); err != nil {
			color.Red("❌ 操作已取消")
			return
		}
	} else {
		componentName = args[0]
	}

	// 验证组件名
	compInfo := GetComponentInfo(componentName)
	if compInfo == nil {
		color.Red("❌ 未知组件: %s", componentName)
		fmt.Println("\n可用组件:")
		for _, name := range GetComponentNames() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// 显示组件信息
	fmt.Println()
	color.Cyan("📦 组件: %s", compInfo.Name)
	fmt.Printf("   %s\n", compInfo.Description)
	fmt.Printf("   依赖: %v\n", compInfo.Dependencies)
	fmt.Printf("   外部服务: %s\n", compInfo.External)
	fmt.Println()

	// 根据组件类型收集配置
	config, err := collectComponentConfig(componentName)
	if err != nil {
		color.Red("❌ 配置收集失败: %v", err)
		return
	}

	// 询问输出目录
	var outputDir string
	outputPrompt := &survey.Input{
		Message: "文档输出目录:",
		Default: "./docs/components",
		Help:    "生成的集成指南将保存到此目录",
	}
	if err := survey.AskOne(outputPrompt, &outputDir); err != nil {
		color.Red("❌ 操作已取消")
		return
	}

	// 确认生成
	var confirm bool
	confirmPrompt := &survey.Confirm{
		Message: fmt.Sprintf("确认生成 %s 组件集成指南到 %s?", componentName, outputDir),
		Default: true,
	}
	if err := survey.AskOne(confirmPrompt, &confirm); err != nil || !confirm {
		color.Yellow("⏹ 已取消")
		return
	}

	// 生成文档
	generator := component.NewDocGenerator(componentName, config, outputDir)
	if err := generator.Generate(); err != nil {
		color.Red("❌ 生成失败: %v", err)
		return
	}

	// 输出成功信息
	fmt.Println()
	color.Green("✅ 集成指南生成成功!")
	fmt.Println()
	fmt.Println("📄 生成的文件:")

	absPath, _ := filepath.Abs(outputDir)
	docFile := filepath.Join(absPath, fmt.Sprintf("%s-integration-guide.md", componentName))
	color.Cyan("   %s", docFile)

	fmt.Println()
	fmt.Println("📖 下一步:")
	fmt.Printf("   1. 打开文档: cat %s\n", docFile)
	fmt.Println("   2. 按照文档步骤逐步操作")
	fmt.Println("   3. 遇到问题参考文档末尾的故障排查")
	fmt.Println()
}

// collectComponentConfig 根据组件类型收集配置
func collectComponentConfig(componentName string) (map[string]interface{}, error) {
	config := make(map[string]interface{})

	switch componentName {
	case "database":
		return collectDatabaseConfig()
	case "redis":
		return collectRedisConfig()
	case "jwt":
		return collectJWTConfig()
	case "kafka":
		return collectKafkaConfig()
	case "telemetry":
		return collectTelemetryConfig()
	case "auth":
		return collectAuthConfig()
	case "limiter":
		return collectLimiterConfig()
	case "event":
		return collectEventConfig()
	default:
		return config, nil
	}
}

// ================= 各组件配置收集 =================

func collectDatabaseConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 数据库类型
	var dbType string
	dbTypePrompt := &survey.Select{
		Message: "数据库类型:",
		Options: []string{"mysql", "postgres", "sqlite"},
		Default: "mysql",
	}
	if err := survey.AskOne(dbTypePrompt, &dbType); err != nil {
		return nil, err
	}
	config["db_type"] = dbType

	// 连接名称
	var connName string
	connNamePrompt := &survey.Input{
		Message: "连接名称:",
		Default: "master",
		Help:    "用于在代码中获取此数据库连接",
	}
	if err := survey.AskOne(connNamePrompt, &connName); err != nil {
		return nil, err
	}
	config["conn_name"] = connName

	// 是否启用 ORM 日志
	var enableLog bool
	logPrompt := &survey.Confirm{
		Message: "启用 SQL 日志?",
		Default: true,
	}
	if err := survey.AskOne(logPrompt, &enableLog); err != nil {
		return nil, err
	}
	config["enable_log"] = enableLog

	// 是否启用 OpenTelemetry
	var enableOtel bool
	otelPrompt := &survey.Confirm{
		Message: "启用 OpenTelemetry 追踪?",
		Default: false,
		Help:    "需要同时配置 telemetry 组件",
	}
	if err := survey.AskOne(otelPrompt, &enableOtel); err != nil {
		return nil, err
	}
	config["enable_otel"] = enableOtel

	return config, nil
}

func collectRedisConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Redis 模式
	var mode string
	modePrompt := &survey.Select{
		Message: "Redis 模式:",
		Options: []string{"standalone", "cluster"},
		Default: "standalone",
	}
	if err := survey.AskOne(modePrompt, &mode); err != nil {
		return nil, err
	}
	config["mode"] = mode

	// 实例名称
	var instanceName string
	instancePrompt := &survey.Input{
		Message: "实例名称:",
		Default: "main",
	}
	if err := survey.AskOne(instancePrompt, &instanceName); err != nil {
		return nil, err
	}
	config["instance_name"] = instanceName

	// 是否需要密码
	var needPassword bool
	pwdPrompt := &survey.Confirm{
		Message: "Redis 需要密码?",
		Default: false,
	}
	if err := survey.AskOne(pwdPrompt, &needPassword); err != nil {
		return nil, err
	}
	config["need_password"] = needPassword

	return config, nil
}

func collectJWTConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 签名算法
	var algorithm string
	algoPrompt := &survey.Select{
		Message: "签名算法:",
		Options: []string{"HS256", "HS384", "HS512", "RS256"},
		Default: "HS256",
	}
	if err := survey.AskOne(algoPrompt, &algorithm); err != nil {
		return nil, err
	}
	config["algorithm"] = algorithm

	// Token 有效期
	var ttl string
	ttlPrompt := &survey.Select{
		Message: "Access Token 有效期:",
		Options: []string{"1h", "2h", "4h", "8h", "24h"},
		Default: "2h",
	}
	if err := survey.AskOne(ttlPrompt, &ttl); err != nil {
		return nil, err
	}
	config["access_token_ttl"] = ttl

	// 是否启用 Refresh Token
	var enableRefresh bool
	refreshPrompt := &survey.Confirm{
		Message: "启用 Refresh Token?",
		Default: true,
	}
	if err := survey.AskOne(refreshPrompt, &enableRefresh); err != nil {
		return nil, err
	}
	config["enable_refresh_token"] = enableRefresh

	// 黑名单存储
	var blacklistStorage string
	blacklistPrompt := &survey.Select{
		Message: "Token 黑名单存储:",
		Options: []string{"memory", "redis", "disabled"},
		Default: "memory",
		Help:    "用于 Token 注销功能",
	}
	if err := survey.AskOne(blacklistPrompt, &blacklistStorage); err != nil {
		return nil, err
	}
	config["blacklist_storage"] = blacklistStorage

	return config, nil
}

func collectKafkaConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 使用场景
	var useCase string
	useCasePrompt := &survey.Select{
		Message: "使用场景:",
		Options: []string{"producer", "consumer", "both"},
		Default: "both",
	}
	if err := survey.AskOne(useCasePrompt, &useCase); err != nil {
		return nil, err
	}
	config["use_case"] = useCase

	// 是否需要认证
	var needAuth bool
	authPrompt := &survey.Confirm{
		Message: "Kafka 需要 SASL 认证?",
		Default: false,
	}
	if err := survey.AskOne(authPrompt, &needAuth); err != nil {
		return nil, err
	}
	config["need_auth"] = needAuth

	// 示例 Topic
	var topicName string
	topicPrompt := &survey.Input{
		Message: "示例 Topic 名称:",
		Default: "my-topic",
	}
	if err := survey.AskOne(topicPrompt, &topicName); err != nil {
		return nil, err
	}
	config["topic_name"] = topicName

	return config, nil
}

func collectTelemetryConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 服务名称
	var serviceName string
	servicePrompt := &survey.Input{
		Message: "服务名称:",
		Default: "my-service",
	}
	if err := survey.AskOne(servicePrompt, &serviceName); err != nil {
		return nil, err
	}
	config["service_name"] = serviceName

	// Exporter 类型
	var exporterType string
	exporterPrompt := &survey.Select{
		Message: "Exporter 类型:",
		Options: []string{"otlp-grpc", "otlp-http", "jaeger", "stdout"},
		Default: "otlp-grpc",
	}
	if err := survey.AskOne(exporterPrompt, &exporterType); err != nil {
		return nil, err
	}
	config["exporter_type"] = exporterType

	// 是否启用 Metrics
	var enableMetrics bool
	metricsPrompt := &survey.Confirm{
		Message: "启用 Metrics?",
		Default: true,
	}
	if err := survey.AskOne(metricsPrompt, &enableMetrics); err != nil {
		return nil, err
	}
	config["enable_metrics"] = enableMetrics

	return config, nil
}

func collectAuthConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 认证方式
	var authMethods []string
	methodPrompt := &survey.MultiSelect{
		Message: "启用的认证方式:",
		Options: []string{"password", "oauth2", "api_key"},
		Default: []string{"password"},
	}
	if err := survey.AskOne(methodPrompt, &authMethods); err != nil {
		return nil, err
	}
	config["auth_methods"] = authMethods

	// 登录尝试限制
	var enableAttemptLimit bool
	attemptPrompt := &survey.Confirm{
		Message: "启用登录尝试限制?",
		Default: true,
		Help:    "防止暴力破解",
	}
	if err := survey.AskOne(attemptPrompt, &enableAttemptLimit); err != nil {
		return nil, err
	}
	config["enable_attempt_limit"] = enableAttemptLimit

	return config, nil
}

func collectLimiterConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 存储类型
	var storeType string
	storePrompt := &survey.Select{
		Message: "限流器存储:",
		Options: []string{"redis", "memory"},
		Default: "redis",
		Help:    "分布式场景推荐 redis",
	}
	if err := survey.AskOne(storePrompt, &storeType); err != nil {
		return nil, err
	}
	config["store_type"] = storeType

	// 算法
	var algorithm string
	algoPrompt := &survey.Select{
		Message: "限流算法:",
		Options: []string{"sliding_window", "token_bucket", "concurrency"},
		Default: "sliding_window",
	}
	if err := survey.AskOne(algoPrompt, &algorithm); err != nil {
		return nil, err
	}
	config["algorithm"] = algorithm

	return config, nil
}

func collectEventConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// 协程池大小
	var poolSize string
	poolPrompt := &survey.Select{
		Message: "事件处理协程池大小:",
		Options: []string{"50", "100", "200", "500"},
		Default: "100",
	}
	if err := survey.AskOne(poolPrompt, &poolSize); err != nil {
		return nil, err
	}
	config["pool_size"] = poolSize

	return config, nil
}
