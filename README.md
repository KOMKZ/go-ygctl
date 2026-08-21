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

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--interactive` | `-i` | false | Interactive mode |
| `--module` | `-m` | - | Go module name |
| `--output` | `-o` | `.` | Output directory |
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
