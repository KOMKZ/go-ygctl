package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplateGeneratorGenerate(t *testing.T) {
	root := t.TempDir()
	cfg := &RenderTemplateConfig{
		Name:           "quote-card",
		ContractsDir:   filepath.Join(root, "hrise-rm-contracts"),
		StudioDir:      filepath.Join(root, "hrise-rm-studio"),
		WorkerDir:      filepath.Join(root, "hrise-rm-render-server"),
		GoComponentDir: filepath.Join(root, "hrise-server-app", "components", "go-yogan-component-render"),
	}

	result, err := NewRenderTemplateGenerator(cfg).Generate()
	if err != nil {
		t.Fatalf("Generate() err = %v", err)
	}

	if result.TemplateVersion != "quote-card-v1" {
		t.Fatalf("TemplateVersion = %q", result.TemplateVersion)
	}
	if result.CompositionID != "QuoteCardV1" {
		t.Fatalf("CompositionID = %q", result.CompositionID)
	}
	if len(result.Files) != 10 {
		t.Fatalf("generated %d files", len(result.Files))
	}

	assertFilesExist(t, root, []string{
		"hrise-rm-contracts/schemas/quote-card-v1.input.schema.json",
		"hrise-rm-contracts/manifests/quote-card-v1.manifest.json",
		"hrise-rm-contracts/fixtures/quote-card-v1.sample.json",
		"hrise-rm-contracts/docs/quote-card-v1.md",
		"hrise-rm-studio/src/render-contract/quote-card-v1/quote-card-v1.input.ts",
		"hrise-rm-studio/src/render-contract/quote-card-v1/quote-card-v1.manifest.ts",
		"hrise-rm-render-server/src/render-contract/quote-card-v1/quote-card-v1.input.ts",
		"hrise-rm-render-server/src/render-contract/quote-card-v1/quote-card-v1.validate.ts",
		"hrise-server-app/components/go-yogan-component-render/rendercontract/quotecardv1/quote_card_v1_input.go",
		"hrise-server-app/components/go-yogan-component-render/renderbuild/quotecardv1/quote_card_v1_builder.go",
	})
	assertFileContains(t, filepath.Join(root, "hrise-rm-contracts/manifests/quote-card-v1.manifest.json"), `"compositionId": "QuoteCardV1"`)
	assertFileContains(t, filepath.Join(root, "hrise-rm-contracts/fixtures/quote-card-v1.sample.json"), `"schemaVersion": "render-job-v1"`)
	assertFileContains(t, filepath.Join(root, "hrise-rm-render-server/src/render-contract/quote-card-v1/quote-card-v1.validate.ts"), "quoteCardRenderJobSchema")
}

func TestRenderTemplateGeneratorRejectsExistingFile(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "contracts", "schemas", "bookquote-v1.input.schema.json")
	if err := os.MkdirAll(filepath.Dir(existing), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &RenderTemplateConfig{
		Name:           "bookquote",
		ContractsDir:   filepath.Join(root, "contracts"),
		StudioDir:      filepath.Join(root, "studio"),
		WorkerDir:      filepath.Join(root, "worker"),
		GoComponentDir: filepath.Join(root, "go-component-render"),
	}

	_, err := NewRenderTemplateGenerator(cfg).Generate()
	if !errors.Is(err, ErrPathExists) {
		t.Fatalf("Generate() err = %v, want ErrPathExists", err)
	}
}

func TestRenderTemplateGeneratorRejectsInvalidName(t *testing.T) {
	cfg := &RenderTemplateConfig{
		Name:           "Bad_Name",
		ContractsDir:   "contracts",
		StudioDir:      "studio",
		WorkerDir:      "worker",
		GoComponentDir: "go-component-render",
	}

	_, err := NewRenderTemplateGenerator(cfg).Generate()
	if err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("Generate() err = %v, want invalid name error", err)
	}
}
