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
	Name          string
	CompositionID string
	ContractsDir  string
	StudioDir     string
	WorkerDir     string
	GoDir         string
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
		{filepath.Join(g.config.ContractsDir, "packages", "ts", "src", data.Version+".ts"), renderTemplateTSType(data)},
		{filepath.Join(g.config.ContractsDir, "packages", "go", "rendercontracts", data.Snake+"_v1.go"), renderTemplateGoType(data)},
		{filepath.Join(g.config.StudioDir, "src", "render-templates", data.Version, "README.md"), renderTemplateStudioReadme(data)},
		{filepath.Join(g.config.StudioDir, "src", "render-templates", data.Version, data.CompositionID+".tsx"), renderTemplateStudioComposition(data)},
		{filepath.Join(g.config.StudioDir, "src", "render-templates", data.Version, "fixture.ts"), renderTemplateStudioFixture(data)},
		{filepath.Join(g.config.WorkerDir, "src", "render", "templates", data.Version, "adapter.ts"), renderTemplateWorkerAdapter(data)},
		{filepath.Join(g.config.GoDir, "internal", "render", data.Snake+"_job_builder.go"), renderTemplateGoBuilder(data)},
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
	if g.config.ContractsDir == "" || g.config.StudioDir == "" || g.config.WorkerDir == "" || g.config.GoDir == "" {
		return fmt.Errorf("contracts-dir, studio-dir, worker-dir, and go-dir are required")
	}
	return nil
}

type renderTemplateData struct {
	Name          string
	Version       string
	Pascal        string
	Snake         string
	CompositionID string
}

func (g *RenderTemplateGenerator) templateData() renderTemplateData {
	return renderTemplateData{
		Name:          g.config.Name,
		Version:       g.config.Name + "-v1",
		Pascal:        ToPascalCase(g.config.Name),
		Snake:         strings.ReplaceAll(g.config.Name, "-", "_"),
		CompositionID: g.config.CompositionID,
	}
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

func renderTemplateTSType(data renderTemplateData) string {
	return fmt.Sprintf(`import type {RenderAssetRef, RenderJobV1} from './index.js';

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

export type %sV1RenderJob = RenderJobV1<%sV1Input>;
`, data.Snake, data.CompositionID, data.Snake, data.Version, data.Pascal, data.Pascal, data.Pascal)
}

func renderTemplateGoType(data renderTemplateData) string {
	return fmt.Sprintf("package rendercontracts\n\nconst (\n\t%sCompositionID = %q\n\t%sVersion       = %q\n)\n\ntype %sV1Input struct {\n\tCanvas %sCanvas        `json:\"canvas\"`\n\tAssets []RenderAssetRef `json:\"assets\"`\n\tStyle  map[string]any   `json:\"style,omitempty\"`\n}\n\ntype %sCanvas struct {\n\tWidth          int `json:\"width\"`\n\tHeight         int `json:\"height\"`\n\tFPS            int `json:\"fps\"`\n\tDurationFrames int `json:\"durationFrames\"`\n}\n\ntype %sV1RenderJob = RenderJobV1[%sV1Input]\n",
		data.Pascal, data.CompositionID, data.Pascal, data.Version, data.Pascal, data.Pascal, data.Pascal, data.Pascal, data.Pascal)
}

func renderTemplateStudioReadme(data renderTemplateData) string {
	return fmt.Sprintf(`# %s Studio Template

## 规则

- 从 contracts fixture 派生 preview props。
- 不复制长期维护的私有 schema。
- 不生成每个视频任务的 Remotion 源码。
- Composition ID 保持为 %s。
`, data.Version, data.CompositionID)
}

func renderTemplateStudioComposition(data renderTemplateData) string {
	return fmt.Sprintf(`import React from 'react';
import {AbsoluteFill} from 'remotion';
import type {%sV1Input} from '@happy-rise/hrise-rm-contracts/%s';

export function %s(props: %sV1Input) {
  return (
    <AbsoluteFill style={{backgroundColor: '#111827', color: 'white', padding: 64}}>
      <h1>%s</h1>
      <pre>{JSON.stringify(props.canvas, null, 2)}</pre>
    </AbsoluteFill>
  );
}
`, data.Pascal, data.Version, data.CompositionID, data.Pascal, data.CompositionID)
}

func renderTemplateStudioFixture(data renderTemplateData) string {
	return fmt.Sprintf(`import fixture from '../../../../hrise-rm-contracts/fixtures/%s.sample.json';

export const %sPreviewProps = fixture.inputProps;
`, data.Version, data.Snake)
}

func renderTemplateWorkerAdapter(data renderTemplateData) string {
	return fmt.Sprintf(`import type {%sV1Input} from '@happy-rise/hrise-rm-contracts/%s';

export const compositionId = '%s';
export const compositionVersion = '%s';

export function prepare%sInput(inputProps: %sV1Input): %sV1Input {
  return inputProps;
}
`, data.Pascal, data.Version, data.CompositionID, data.Version, data.Pascal, data.Pascal, data.Pascal)
}

func renderTemplateGoBuilder(data renderTemplateData) string {
	return fmt.Sprintf(`package render

import contracts "github.com/KOMKZ/hrise-rm-contracts/packages/go/rendercontracts"

func New%sRenderJob(requestID string, input contracts.%sV1Input) contracts.%sV1RenderJob {
	return contracts.%sV1RenderJob{
		SchemaVersion:      contracts.RenderJobV1SchemaVersion,
		RequestID:          requestID,
		IdempotencyKey:     requestID + ":%s",
		CompositionID:      contracts.%sCompositionID,
		CompositionVersion: contracts.%sVersion,
		Output: contracts.RenderJobOutput{
			OssKey:      "renders/v1/" + requestID + "/output.mp4",
			ContentType: "video/mp4",
		},
		Callback: contracts.RenderCallbackPolicy{
			MaxAttempts:  contracts.CallbackMaxAttempts,
			RetryDelayMs: contracts.CallbackRetryDelayMs,
		},
		InputProps: input,
	}
}
`, data.Pascal, data.Pascal, data.Pascal, data.Pascal, data.Version, data.Pascal, data.Pascal)
}

func prettyJSON(value any) string {
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(content) + "\n"
}
