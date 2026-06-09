# vengo

`vengo` is an early Go-native application framework inspired by Spring Boot's developer experience, with a smaller and more explicit runtime model.

This repository is currently a starter skeleton. The first implementation includes:

- core application lifecycle and module registration
- named service registry with typed lookup helpers
- TOML/JSON configuration with typed struct binding, profile support, and redacted reports
- small auto-configuration registry
- `net/http` based web module
- actuator-style health endpoint module
- CLI scaffold for version and project generation
- runnable hello example

The project is intentionally one Go module for now. Splitting packages into separate modules can happen once the public API is more stable.

## Quick Start

```bash
go test ./...
go run ./examples/hello
```

Then open:

```text
http://localhost:8080/
http://localhost:8080/actuator/health
```

## CLI

```bash
go run ./cmd/vengo version
go run ./cmd/vengo new orders-api github.com/you/orders-api
```

If the module path is omitted, `vengo new` uses the project directory name as the module path.

## Health Endpoint

The actuator module exposes `/actuator/health` by default:

```go
app := core.New("hello", web.New(":8080"), actuator.NewHealth())
```

Use options when you need a custom path or explicit checks:

```go
health := actuator.NewHealth(
	actuator.WithPath("/healthz"),
	actuator.WithChecks(actuator.Check{Name: "self", Check: func(context.Context) error { return nil }}),
)
```

## Configuration

Load `application.toml` (or `.json`) with profile overrides and bind into typed structs:

```go
type ServerConfig struct {
    Port int    `config:"server.port" default:"8080"`
    Host string `config:"server.host" default:"localhost"`
}

cfg, _ := config.LoadDefaults(ctx, config.ActiveProfile())

var serverCfg ServerConfig
config.Bind(cfg, &serverCfg)
```

Source precedence (later overrides earlier):

1. `application.toml` (from `.` or `./config/`)
2. `application-{profile}.toml`
3. Environment variables with `APP_` prefix (`APP_SERVER_PORT` → `server.port`)

Set the active profile via `APP_PROFILE` or `VENGO_PROFILE` environment variables.

## Design Direction

- Prefer explicit registration over package scanning and hidden auto-wiring.
- Keep the core dependency-light.
- Build diagnostics into every major subsystem.
- Make integrations optional modules rather than framework requirements.