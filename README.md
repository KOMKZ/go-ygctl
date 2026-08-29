# go-ygctl

Yogan Framework CLI tool for generating applications.

## Installation

```bash
go install github.com/KOMKZ/go-ygctl@latest
```

Or build from source:

```bash
git clone https://github.com/KOMKZ/go-ygctl.git
cd go-ygctl
go build -o go-ygctl .
```

## Usage

### Render Template Skeleton

```bash
go-ygctl render template init --name bookquote --composition-id BookquoteV1
```

This command generates deterministic skeleton files across contracts, Studio,
render worker, and Go workflow handoff directories. It does not generate
business copy, credentials, assets, or per-video Remotion source.

### Create HTTP Application

**Interactive mode:**

```bash
go-ygctl new http --interactive
```

**Quick mode:**

```bash
# Local framework (with go.work)
go-ygctl new http my-api --module github.com/myorg/my-api --output ./apps

# Remote framework
go-ygctl new http my-api --module github.com/myorg/my-api --local-framework=false
```

### Create CLI Application

**New multi-app project:**

```bash
go-ygctl new cli demo-cli --project demo-proj --org github.com/KOMKZ --output .
```

**Existing multi-app workspace:**

```bash
cd hrise-server-app
go-ygctl new cli hrse-cli --workspace . --org github.com/KOMKZ
go run ./apps/hrse-cli hello
```

The generated CLI app includes a `hello` subcommand that prints `hello world`
and test coverage for the generated `main`, `app`, and `command` packages.

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interactive` | `-i` | false | Interactive mode |
| `--module` | `-m` | - | Go module name |
| `--output` | `-o` | `.` | Output directory |
| `--workspace` | - | - | Existing multi-app workspace path for `new cli` |
| `--port` | `-p` | 8080 | Server port |
| `--local-framework` | - | true | Use local framework with replace |
| `--framework-path` | - | `../../go-yogan-framework` | Local framework path |

## Generated Structure

```
my-api/
├── main.go
├── go.mod
├── configs/
│   └── config.yaml
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── callbacks.go
│   │   ├── components.go
│   │   └── router.go
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   └── demo/
│   │       ├── model/
│   │       ├── repository.go
│   │       ├── repository_memory.go
│   │       └── service.go
│   ├── module/
│   │   └── demo/
│   │       ├── handler.go
│   │       ├── request.go
│   │       └── response.go
│   └── router/
│       ├── demo.go
│       └── health.go
└── pkg/
    └── util/
```

## License

MIT
