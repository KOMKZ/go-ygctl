# Structure Gate Report

- mode: `changed`
- files scanned: `65`
- functions scanned: `330`
- issues: `19`
- blocking: `0`

## Issues

| severity | kind | path | name | value | threshold | changed | reason |
|---|---|---|---|---:|---:|---|---|
| block | naming | `internal/generator/config.go` | `` | 1 | 0 | false | weak file name; split/new business files should use business_role.go |
| block | naming | `cmd/model.go` | `` | 1 | 0 | false | weak file name; split/new business files should use business_role.go |
| warn | func | `internal/generator/api_gen.go` | `*APIGenConfig.Generate` | 160 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/domain_migrator.go` | `*DomainMigrator.Migrate` | 159 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/http_generator.go` | `*HTTPGenerator.Generate` | 157 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/make_sync.go` | `syncOneApp` | 146 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/prompt.go` | `PromptHTTPConfig` | 138 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/prompt.go` | `PromptRPCConfig` | 134 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/web_gen.go` | `resolveWebFields` | 131 | 130 | false | function line count exceeds threshold |
| warn | package | `internal/generator` | `` | 28 | 18 | true | package file count exceeds threshold; report only, no automatic subpackage move |
| warn | package | `cmd` | `` | 21 | 18 | false | package file count exceeds threshold; report only, no automatic subpackage move |
| warn | errlint | `internal/generator/api_gen.go` | `LINT-ERR-006` | 4 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 97,98,99,144) |
| warn | errlint | `internal/generator/domain_migrator.go` | `LINT-ERR-006` | 2 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 181,336) |
| warn | errlint | `cmd/api.go` | `LINT-ERR-006` | 1 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 195) |
| warn | errlint | `cmd/domain_migrate.go` | `LINT-ERR-006` | 1 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 39) |
| warn | errlint | `cmd/migrate.go` | `LINT-ERR-006` | 1 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 88) |
| warn | errlint | `cmd/model.go` | `LINT-ERR-006` | 1 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 67) |
| warn | errlint | `cmd/permission.go` | `LINT-ERR-006` | 1 | 0 | false | 禁止裸 _ = call() 吞错。修复：可传播→errcode.Capture 后返回；无法返回的旁路→logger.GetLogger(域).Warn 或 gin c.Error 显式告警；解码容错→if err != nil 显式降级零值；装配期错误→fail-fast panic (lines 69) |
| warn | errlint | `internal/generator/migrate.go` | `LINT-ERR-004` | 1 | 0 | false | err.Error() 不得进响应/DTO/持久化。修复：用户面 errcode.SafeMessage(err).PublicMessage 或 PublicMessageOf；运维/日志诊断面 errcode.Redact(err.Error())；内部分类改 errors.As 类型化判别 (lines 217) |

## Remediation Template

- split plan: describe the target responsibility split, such as service/repository/model/policy/adapter/generator template/fixture/test scenario.
- impact: list changed packages, public APIs, imports, generated outputs, and callers that need review.
- verification: list exact commands, including `make structure-gate` plus focused build/test commands.
- baseline rule: refresh baseline only inside a governance ticket with `STRUCTURE_GATE_BASELINE_TICKET=<ticket-id>`.
- goal: do not chase zero warnings; keep new code from worsening, protect critical paths, and maintain a stock issue map.
