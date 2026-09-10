# Structure Gate Report

- mode: `changed`
- files scanned: `62`
- functions scanned: `307`
- issues: `11`
- blocking: `0`

## Issues

| severity | kind | path | name | value | threshold | changed | reason |
|---|---|---|---|---:|---:|---|---|
| block | naming | `internal/generator/config.go` | `` | 1 | 0 | false | weak file name; split/new business files should use business_role.go |
| block | naming | `cmd/model.go` | `` | 1 | 0 | false | weak file name; split/new business files should use business_role.go |
| warn | func | `internal/generator/domain_migrator.go` | `*DomainMigrator.Migrate` | 159 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/http_generator.go` | `*HTTPGenerator.Generate` | 157 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/api_gen.go` | `*APIGenConfig.Generate` | 155 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/make_sync.go` | `syncOneApp` | 146 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/prompt.go` | `PromptHTTPConfig` | 138 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/prompt.go` | `PromptRPCConfig` | 134 | 130 | false | function line count exceeds threshold |
| warn | func | `internal/generator/web_gen.go` | `resolveWebFields` | 131 | 130 | false | function line count exceeds threshold |
| warn | package | `internal/generator` | `` | 27 | 18 | false | package file count exceeds threshold; report only, no automatic subpackage move |
| warn | package | `cmd` | `` | 20 | 18 | false | package file count exceeds threshold; report only, no automatic subpackage move |

## Remediation Template

- split plan: describe the target responsibility split, such as service/repository/model/policy/adapter/generator template/fixture/test scenario.
- impact: list changed packages, public APIs, imports, generated outputs, and callers that need review.
- verification: list exact commands, including `make structure-gate` plus focused build/test commands.
- baseline rule: refresh baseline only inside a governance ticket with `STRUCTURE_GATE_BASELINE_TICKET=<ticket-id>`.
- goal: do not chase zero warnings; keep new code from worsening, protect critical paths, and maintain a stock issue map.
