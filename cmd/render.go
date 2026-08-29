package cmd

import (
	"fmt"
	"strings"

	"github.com/KOMKZ/go-ygctl/internal/generator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	renderTemplateName         string
	renderTemplateComposition  string
	renderTemplateContractsDir string
	renderTemplateStudioDir    string
	renderTemplateWorkerDir    string
	renderTemplateGoDir        string
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "Manage Remotion render templates",
	Long:  "Manage Remotion render template skeletons shared by contracts, Studio, render worker, and Go workflow.",
}

var renderTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage render template skeletons",
}

var renderTemplateInitCmd = &cobra.Command{
	Use:   "init --name <template>",
	Short: "Generate render template contract and handoff skeletons",
	Long: `Generate deterministic render template skeleton files.

Run this from the happy-rise-devpro directory or pass explicit paths:

  ygctl render template init --name bookquote --composition-id BookquoteV1`,
	Args: cobra.NoArgs,
	RunE: runRenderTemplateInit,
}

func init() {
	rootCmd.AddCommand(renderCmd)
	renderCmd.AddCommand(renderTemplateCmd)
	renderTemplateCmd.AddCommand(renderTemplateInitCmd)

	renderTemplateInitCmd.Flags().StringVar(&renderTemplateName, "name", "", "Template key, e.g. bookquote")
	renderTemplateInitCmd.Flags().StringVar(&renderTemplateComposition, "composition-id", "", "Remotion composition ID, defaults to PascalCase(name)+V1")
	renderTemplateInitCmd.Flags().StringVar(&renderTemplateContractsDir, "contracts-dir", "hrise-rm-contracts", "Contracts project directory")
	renderTemplateInitCmd.Flags().StringVar(&renderTemplateStudioDir, "studio-dir", "hrise-rm-studio", "Remotion Studio project directory")
	renderTemplateInitCmd.Flags().StringVar(&renderTemplateWorkerDir, "worker-dir", "hrise-rm-render-server", "Render worker project directory")
	renderTemplateInitCmd.Flags().StringVar(&renderTemplateGoDir, "go-dir", "hrise-server-app", "Go workflow project directory")
}

func runRenderTemplateInit(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(renderTemplateName) == "" {
		return fmt.Errorf("--name is required")
	}

	config := &generator.RenderTemplateConfig{
		Name:          renderTemplateName,
		CompositionID: renderTemplateComposition,
		ContractsDir:  renderTemplateContractsDir,
		StudioDir:     renderTemplateStudioDir,
		WorkerDir:     renderTemplateWorkerDir,
		GoDir:         renderTemplateGoDir,
	}

	color.Cyan("\n🚀 Generating render template: %s", config.Name)
	result, err := generator.NewRenderTemplateGenerator(config).Generate()
	if err != nil {
		return err
	}

	color.Green("✅ Render template skeleton generated successfully!")
	fmt.Printf("   Template:     %s\n", result.TemplateVersion)
	fmt.Printf("   Composition:  %s\n", result.CompositionID)
	fmt.Printf("   Files:        %d\n", len(result.Files))
	for _, file := range result.Files {
		fmt.Printf("     - %s\n", file)
	}
	return nil
}
