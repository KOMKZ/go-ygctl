package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	workerName      string
	workerDriver    string
	workerProfiles  string
	workerWorkspace string
	workerOrg       string
	workerModule    string
	workerFWPath    string
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage queue worker applications",
	Long:  "Manage long-running queue worker applications in a Yogan workspace.",
}

var workerInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a queue worker application",
	Long: `Generate an engineering-grade worker application under apps/<name>.

Example:
  ygctl worker init --name hrise-worker --driver asynq --profiles default,media`,
	Args: cobra.NoArgs,
	RunE: runWorkerInit,
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.AddCommand(workerInitCmd)

	workerInitCmd.Flags().StringVar(&workerName, "name", "", "Worker app name (required)")
	workerInitCmd.Flags().StringVar(&workerDriver, "driver", "asynq", "Queue driver")
	workerInitCmd.Flags().StringVar(&workerProfiles, "profiles", "default", "Comma-separated worker profiles")
	workerInitCmd.Flags().StringVar(&workerWorkspace, "workspace", "", "Workspace root (default: search upward for go.work)")
	workerInitCmd.Flags().StringVar(&workerOrg, "org", "github.com/KOMKZ", "Organization module prefix")
	workerInitCmd.Flags().StringVarP(&workerModule, "module", "m", "", "App Go module name (auto-generated if empty)")
	workerInitCmd.Flags().StringVar(&workerFWPath, "framework-path", "../../../go-yogan-framework", "Local framework path (relative to apps/<worker>)")
}

func runWorkerInit(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(workerName) == "" {
		return fmt.Errorf("--name is required")
	}

	workspace, err := resolveWorkerWorkspace(workerWorkspace)
	if err != nil {
		return err
	}

	config := generator.NewDefaultWorkerConfig()
	config.AppName = workerName
	config.ProjectName = filepath.Base(workspace)
	config.OrgName = workerOrg
	config.ModuleName = workerModule
	config.OutputPath = "."
	config.WorkspacePath = workspace
	config.Driver = workerDriver
	config.Profiles = splitCSV(workerProfiles)
	config.FrameworkPath = workerFWPath
	config.Description = fmt.Sprintf("%s worker", generator.ToPascalCase(workerName))

	color.Cyan("\n🚀 Generating worker application: %s", config.AppName)
	if err := generator.NewWorkerGenerator(config).Generate(); err != nil {
		return err
	}

	appPath := filepath.Join(workspace, "apps", config.AppName)
	color.Green("✅ Worker generated successfully!")
	fmt.Printf("   Path:     %s\n", appPath)
	fmt.Printf("   Driver:   %s\n", config.Driver)
	fmt.Printf("   Profiles: %s\n", strings.Join(config.Profiles, ","))
	return nil
}

func resolveWorkerWorkspace(workspace string) (string, error) {
	if workspace != "" {
		return filepath.Abs(workspace)
	}
	return generator.FindWorkspaceRoot("")
}

func splitCSV(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{"default"}
	}
	return result
}
