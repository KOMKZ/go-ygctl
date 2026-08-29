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
		Name:         "quote-card",
		ContractsDir: filepath.Join(root, "hrise-rm-contracts"),
		StudioDir:    filepath.Join(root, "hrise-rm-studio"),
		WorkerDir:    filepath.Join(root, "hrise-rm-render-server"),
		GoDir:        filepath.Join(root, "hrise-server-app"),
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
	if len(result.Files) != 11 {
		t.Fatalf("generated %d files", len(result.Files))
	}

	assertFilesExist(t, root, []string{
		"hrise-rm-contracts/schemas/quote-card-v1.input.schema.json",
		"hrise-rm-contracts/manifests/quote-card-v1.manifest.json",
		"hrise-rm-contracts/fixtures/quote-card-v1.sample.json",
		"hrise-rm-contracts/docs/quote-card-v1.md",
		"hrise-rm-contracts/packages/ts/src/quote-card-v1.ts",
		"hrise-rm-contracts/packages/go/rendercontracts/quote_card_v1.go",
		"hrise-rm-studio/src/render-templates/quote-card-v1/QuoteCardV1.tsx",
		"hrise-rm-studio/src/render-templates/quote-card-v1/fixture.ts",
		"hrise-rm-render-server/src/render/templates/quote-card-v1/adapter.ts",
		"hrise-server-app/internal/render/quote_card_job_builder.go",
	})
	assertFileContains(t, filepath.Join(root, "hrise-rm-contracts/manifests/quote-card-v1.manifest.json"), `"compositionId": "QuoteCardV1"`)
	assertFileContains(t, filepath.Join(root, "hrise-rm-contracts/fixtures/quote-card-v1.sample.json"), `"schemaVersion": "render-job-v1"`)
	assertFileContains(t, filepath.Join(root, "hrise-rm-render-server/src/render/templates/quote-card-v1/adapter.ts"), "prepareQuoteCardInput")
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
		Name:         "bookquote",
		ContractsDir: filepath.Join(root, "contracts"),
		StudioDir:    filepath.Join(root, "studio"),
		WorkerDir:    filepath.Join(root, "worker"),
		GoDir:        filepath.Join(root, "go"),
	}

	_, err := NewRenderTemplateGenerator(cfg).Generate()
	if !errors.Is(err, ErrPathExists) {
		t.Fatalf("Generate() err = %v, want ErrPathExists", err)
	}
}

func TestRenderTemplateGeneratorRejectsInvalidName(t *testing.T) {
	cfg := &RenderTemplateConfig{
		Name:         "Bad_Name",
		ContractsDir: "contracts",
		StudioDir:    "studio",
		WorkerDir:    "worker",
		GoDir:        "go",
	}

	_, err := NewRenderTemplateGenerator(cfg).Generate()
	if err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("Generate() err = %v, want invalid name error", err)
	}
}
