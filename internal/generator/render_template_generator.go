package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type RenderTemplateConfig struct {
	Name           string
	CompositionID  string
	ContractsDir   string
	StudioDir      string
	WorkerDir      string
	GoComponentDir string
}

type RenderTemplateResult struct {
	TemplateVersion string
	CompositionID   string
	Files           []string
}

type RenderTemplateGenerator struct {
	config *RenderTemplateConfig
}

func NewRenderTemplateGenerator(config *RenderTemplateConfig) *RenderTemplateGenerator {
	return &RenderTemplateGenerator{config: config}
}

func (g *RenderTemplateGenerator) Generate() (*RenderTemplateResult, error) {
	if err := g.prepareConfig(); err != nil {
		return nil, err
	}

	data := g.templateData()
	files := []struct {
		path    string
		content string
	}{
		{filepath.Join(g.config.ContractsDir, "schemas", data.Version+".input.schema.json"), renderTemplateInputSchema(data)},
		{filepath.Join(g.config.ContractsDir, "manifests", data.Version+".manifest.json"), renderTemplateManifest(data)},
		{filepath.Join(g.config.ContractsDir, "fixtures", data.Version+".sample.json"), renderTemplateFixture(data)},
		{filepath.Join(g.config.ContractsDir, "docs", data.Version+".md"), renderTemplateDoc(data)},
		{filepath.Join(g.config.StudioDir, "src", "render-contract", data.Version, data.Version+".input.ts"), renderTemplateTSType(data, "studio")},
		{filepath.Join(g.config.StudioDir, "src", "render-contract", data.Version, data.Version+".manifest.ts"), renderTemplateTSManifest(data)},
		{filepath.Join(g.config.WorkerDir, "src", "render-contract", data.Version, data.Version+".input.ts"), renderTemplateTSType(data, "worker")},
		{filepath.Join(g.config.WorkerDir, "src", "render-contract", data.Version, data.Version+".validate.ts"), renderTemplateWorkerValidator(data)},
		{filepath.Join(g.config.GoComponentDir, "rendercontract", data.GoPackage, data.Snake+"_v1_input.go"), renderTemplateGoType(data)},
		{filepath.Join(g.config.GoComponentDir, "renderbuild", data.GoPackage, data.Snake+"_v1_builder.go"), renderTemplateGoBuilder(data)},
	}

	var written []string
	for _, file := range files {
		if err := writeNewFile(file.path, file.content); err != nil {
			return nil, err
		}
		written = append(written, file.path)
	}

	return &RenderTemplateResult{
		TemplateVersion: data.Version,
		CompositionID:   data.CompositionID,
		Files:           written,
	}, nil
}

func (g *RenderTemplateGenerator) prepareConfig() error {
	name := strings.TrimSpace(g.config.Name)
	if name == "" {
		return fmt.Errorf("render template name is required")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
		return fmt.Errorf("render template name must match ^[a-z][a-z0-9-]*$: %s", name)
	}

	g.config.Name = name
	if strings.TrimSpace(g.config.CompositionID) == "" {
		g.config.CompositionID = ToPascalCase(name) + "V1"
	}
	if g.config.ContractsDir == "" || g.config.StudioDir == "" || g.config.WorkerDir == "" || g.config.GoComponentDir == "" {
		return fmt.Errorf("contracts-dir, studio-dir, worker-dir, and go-component-dir are required")
	}
	return nil
}

type renderTemplateData struct {
	Name          string
	Version       string
	Pascal        string
	Snake         string
	JSIdent       string
	GoPackage     string
	CompositionID string
}

func (g *RenderTemplateGenerator) templateData() renderTemplateData {
	return renderTemplateData{
		Name:          g.config.Name,
		Version:       g.config.Name + "-v1",
		Pascal:        ToPascalCase(g.config.Name),
		Snake:         strings.ReplaceAll(g.config.Name, "-", "_"),
		JSIdent:       lowerFirst(ToPascalCase(g.config.Name)),
		GoPackage:     strings.ReplaceAll(g.config.Name, "-", "") + "v1",
		CompositionID: g.config.CompositionID,
	}
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func writeNewFile(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrPathExists, path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func renderTemplateInputSchema(data renderTemplateData) string {
	return prettyJSON(map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"title":                data.Pascal + "V1Input",
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"canvas", "assets"},
		"properties": map[string]any{
			"canvas": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"width", "height", "fps", "durationFrames"},
				"properties": map[string]any{
					"width":          map[string]any{"type": "integer", "minimum": 1},
					"height":         map[string]any{"type": "integer", "minimum": 1},
					"fps":            map[string]any{"type": "integer", "minimum": 1},
					"durationFrames": map[string]any{"type": "integer", "minimum": 1},
				},
			},
			"assets": map[string]any{
				"type":  "array",
				"items": map[string]any{"$ref": "#/$defs/renderAssetRef"},
			},
			"style": map[string]any{"type": "object", "additionalProperties": true},
		},
		"$defs": map[string]any{
			"renderAssetRef": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"kind", "ossKey", "contentType"},
				"properties": map[string]any{
					"kind":        map[string]any{"enum": []string{"image", "audio", "subtitle", "data"}},
					"ossKey":      map[string]any{"type": "string", "minLength": 1},
					"contentType": map[string]any{"type": "string", "minLength": 1},
				},
			},
		},
	})
}

func renderTemplateManifest(data renderTemplateData) string {
	return prettyJSON(map[string]any{
		"manifestVersion":    "render-manifest-v1",
		"compositionId":      data.CompositionID,
		"compositionVersion": data.Version,
		"inputSchema":        "../schemas/" + data.Version + ".input.schema.json",
		"jobSchema":          "../schemas/render-job-v1.schema.json",
		"resultSchema":       "../schemas/render-result-v1.schema.json",
		"fixtures":           []string{"../fixtures/" + data.Version + ".sample.json"},
		"output": map[string]any{
			"contentType": "video/mp4",
			"width":       1080,
			"height":      1920,
			"fps":         30,
		},
		"owner": "hrise-rm-contracts",
		"notes": "Go prepares all assets and timeline data. Remotion consumes inputProps only.",
	})
}

func renderTemplateFixture(data renderTemplateData) string {
	requestID := "render_req_" + data.Snake + "_001"
	return prettyJSON(map[string]any{
		"schemaVersion":      "render-job-v1",
		"requestId":          requestID,
		"idempotencyKey":     requestID + ":" + data.Version,
		"compositionId":      data.CompositionID,
		"compositionVersion": data.Version,
		"output": map[string]any{
			"ossKey":      "renders/v1/" + requestID + "/output.mp4",
			"contentType": "video/mp4",
		},
		"callback": map[string]any{
			"url":          "http://hrise-server-app.internal/api/render/callback",
			"authTokenRef": "render-callback-token",
			"maxAttempts":  3,
			"retryDelayMs": 3000,
		},
		"inputProps": map[string]any{
			"canvas": map[string]any{
				"width":          1080,
				"height":         1920,
				"fps":            30,
				"durationFrames": 300,
			},
			"assets": []any{
				map[string]any{
					"kind":        "image",
					"ossKey":      "assets/v1/" + requestID + "/image-01.png",
					"contentType": "image/png",
				},
			},
			"style": map[string]any{},
		},
	})
}

func renderTemplateDoc(data renderTemplateData) string {
	return fmt.Sprintf(`# %s

## 契约

| 项 | 值 |
|----|----|
| compositionId | %s |
| compositionVersion | %s |
| input schema | schemas/%s.input.schema.json |
| sample fixture | fixtures/%s.sample.json |

## 边界

- Go 准备业务数据、素材、TTS/ASR、幂等键和 callback URL。
- Redis 只承载 RenderJobV1 队列消息。
- OSS 存输入素材、输出视频和 diagnostics。
- Remotion 只消费 inputProps 并渲染。
`, data.Version, data.CompositionID, data.Version, data.Version, data.Version)
}

func renderTemplateTSType(data renderTemplateData, target string) string {
	importPath := "../envelope/render-job-v1"
	if target == "worker" {
		importPath = "../envelope/render-job-v1.js"
	}
	return fmt.Sprintf(`// Code generated from hrise-rm-contracts/schemas/%s.input.schema.json; DO NOT EDIT.

import type {RenderJobV1} from '%s';

export const %sCompositionId = '%s' as const;
export const %sCompositionVersion = '%s' as const;

export type %sV1Input = {
  canvas: {
    width: number;
    height: number;
    fps: number;
    durationFrames: number;
  };
  assets: RenderAssetRef[];
  style?: Record<string, unknown>;
};

export type RenderAssetRef = {
  kind: 'image' | 'audio' | 'subtitle' | 'data';
  ossKey: string;
  contentType: string;
};

export type %sV1RenderJob = RenderJobV1<%sV1Input>;
`, data.Version, importPath, data.JSIdent, data.CompositionID, data.JSIdent, data.Version, data.Pascal, data.Pascal, data.Pascal)
}

func renderTemplateTSManifest(data renderTemplateData) string {
	return fmt.Sprintf(`// Code generated from hrise-rm-contracts/manifests/%s.manifest.json; DO NOT EDIT.

export const %sManifest = {
  manifestVersion: 'render-manifest-v1',
  compositionId: '%s',
  compositionVersion: '%s',
  inputSchema: '../schemas/%s.input.schema.json',
  jobSchema: '../schemas/render-job-v1.schema.json',
  resultSchema: '../schemas/render-result-v1.schema.json',
  fixtures: ['../fixtures/%s.sample.json'],
  output: {
    contentType: 'video/mp4',
    width: 1080,
    height: 1920,
    fps: 30,
  },
} as const;
`, data.Version, data.JSIdent, data.CompositionID, data.Version, data.Version, data.Version)
}

func renderTemplateGoType(data renderTemplateData) string {
	return fmt.Sprintf("// Code generated from hrise-rm-contracts/schemas/%s.input.schema.json; DO NOT EDIT.\n\npackage %s\n\nimport \"github.com/KOMKZ/go-yogan-component-render/rendercontract/envelope\"\n\nconst (\n\tCompositionID      = %q\n\tCompositionVersion = %q\n)\n\ntype Input struct {\n\tCanvas Canvas         `json:\"canvas\"`\n\tAssets []RenderAssetRef `json:\"assets\"`\n\tStyle  map[string]any   `json:\"style,omitempty\"`\n}\n\ntype Canvas struct {\n\tWidth          int `json:\"width\"`\n\tHeight         int `json:\"height\"`\n\tFPS            int `json:\"fps\"`\n\tDurationFrames int `json:\"durationFrames\"`\n}\n\ntype RenderAssetRef struct {\n\tKind        string `json:\"kind\"`\n\tOssKey      string `json:\"ossKey\"`\n\tContentType string `json:\"contentType\"`\n}\n\ntype RenderJob = envelope.RenderJobV1[Input]\n",
		data.Version, data.GoPackage, data.CompositionID, data.Version)
}

func renderTemplateWorkerValidator(data renderTemplateData) string {
	return fmt.Sprintf(`// Code generated from hrise-rm-contracts/schemas/%s.input.schema.json; DO NOT EDIT.

import {z} from 'zod';
import {%sCompositionId, %sCompositionVersion} from './%s.input.js';

export const %sInputSchema = z.object({
  canvas: z.object({
    width: z.number().int().positive(),
    height: z.number().int().positive(),
    fps: z.number().int().positive(),
    durationFrames: z.number().int().positive(),
  }),
  assets: z.array(z.object({
    kind: z.enum(['image', 'audio', 'subtitle', 'data']),
    ossKey: z.string().min(1),
    contentType: z.string().min(1),
  })),
  style: z.record(z.unknown()).optional(),
});

export const %sRenderJobSchema = z.object({
  schemaVersion: z.literal('render-job-v1'),
  requestId: z.string().min(1),
  idempotencyKey: z.string().min(1),
  compositionId: z.literal(%sCompositionId),
  compositionVersion: z.literal(%sCompositionVersion),
  output: z.object({
    ossKey: z.string().min(1),
    contentType: z.literal('video/mp4'),
  }),
  callback: z.object({
    url: z.string().url(),
    authTokenRef: z.string().min(1).optional(),
    maxAttempts: z.number().int().min(1).max(5),
    retryDelayMs: z.number().int().min(100).max(60000),
  }),
  inputProps: %sInputSchema,
});
`, data.Version, data.JSIdent, data.JSIdent, data.Version, data.JSIdent, data.JSIdent, data.JSIdent, data.JSIdent, data.JSIdent)
}

func renderTemplateGoBuilder(data renderTemplateData) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/KOMKZ/go-yogan-component-render/renderbuild/common"
	contract "github.com/KOMKZ/go-yogan-component-render/rendercontract/%s"
	"github.com/KOMKZ/go-yogan-component-render/rendercontract/envelope"
)

func NewJob(input contract.Input, options common.Options) contract.RenderJob {
	return contract.RenderJob{
		SchemaVersion:      envelope.RenderJobV1SchemaVersion,
		RequestID:          options.RequestID,
		IdempotencyKey:     options.IdempotencyKey,
		CompositionID:      contract.CompositionID,
		CompositionVersion: contract.CompositionVersion,
		Output: envelope.RenderJobOutput{
			OssKey:      options.OutputOssKey,
			ContentType: "video/mp4",
		},
		Callback: envelope.RenderCallbackPolicy{
			URL:          options.CallbackURL,
			AuthTokenRef: options.CallbackAuthRef,
			MaxAttempts:  envelope.CallbackMaxAttempts,
			RetryDelayMs: envelope.CallbackRetryDelayMs,
		},
		InputProps: input,
		TimeoutMs:  options.TimeoutMs,
		Trace:      options.Trace,
	}
}
`, data.GoPackage, data.GoPackage)
}

func prettyJSON(value any) string {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(content) + "\n"
}
