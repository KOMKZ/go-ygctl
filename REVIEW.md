# REVIEW: CLI generator workspace support

## 准入结论

PASS。`go-ygctl new cli` 已支持向现有 multi-app workspace 追加 CLI app，并生成 `command/module/service` 三层 CLI 骨架；app Makefile 的 build 产物进入应用自身 `build/` 目录。

## 门禁证据

- `go test ./internal/generator/...`：PASS。
- `go build -o go-ygctl .`：PASS。
- 临时 workspace 执行 `go-ygctl new cli tmp-cli --workspace . --org github.com/KOMKZ`：PASS。
- 真实 workspace 执行 `go-ygctl new cli hrise-cli --workspace . --org github.com/KOMKZ`：PASS。
- 生成物 `cd apps/hrise-cli && make build`：PASS，输出到 `apps/hrise-cli/build/hrise-cli`。

## 阻断项

无。

## 说明

`go test ./...` 通过，但 `main`、`cmd`、`internal/component` 仍显示 `[no test files]`；本次生成器交付门禁按规范聚焦 `internal/generator`。任务开始前已存在的 API DSL 相关未提交修改未纳入本次 REVIEW 结论。
