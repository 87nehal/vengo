# Architecture

The first implementation uses a small set of packages instead of a multi-module workspace. That keeps local development simple while the public API is still moving.

## Packages

- `core`: application lifecycle, module registration, hooks, named services, and dependency injection container.
- `config`: configuration sources (TOML, JSON, env, embed.FS), precedence, typed struct binding, validation, and redacted reporting.
- `autoconfigure`: conditional configuration registry.
- `web`: `net/http` based server module with middleware chains, route groups, structured errors, JSON binding, route registry, and request logging.
- `actuator`: production-readiness modules — health checks (liveness/readiness), info endpoint, metrics (counters/gauges), structured slog logging, and OpenTelemetry-compatible tracing hooks.
- `data`: `database/sql` integration, connection pool config, named DB service registration, transactions, SQL migrations, DB health checks, and explicit slow query logging helpers.
- `starter/web`: convenience package for creating the web module.
- `cmd/vengo`: CLI entry point for version checks, project generation, config inspection, dependency graph display, route listing, environment doctor checks, and dev-mode (`vengo run`) file watching.
- `examples`: runnable example applications for hello-world web usage, data/sql, security, testing, and CLI-worker patterns.

## Design Rules

1. Keep the core independent of HTTP, database, cloud, and security packages.
2. Prefer explicit module registration over package scanning.
3. Use typed providers and constructor injection for dependency wiring, with named services as a fallback.
4. Keep production behavior observable through logs, reports, and future explain commands.
5. Add code generation only after benchmarks show a real need.

## Dependency Injection

The `core` package provides a reflection-based dependency injection container layered on top of the named service registry.

### Providers

Register constructor functions with `core.Provide`:

```go
core.Provide(app, NewUserRepo)
core.Provide(app, NewUserService)
core.Provide(app, NewUserHandler)
```

The container inspects each constructor's parameter types to discover dependencies automatically.

### Resolution

Resolve instances by type with `core.Resolve`:

```go
handler, err := core.Resolve[*UserHandler](app)
```

Resolved instances are cached as singletons. The container resolves the full dependency graph recursively.

### Field Injection

Use the `inject:""` struct tag for field injection after construction:

```go
type Handler struct {
    Service *UserService `inject:""`
}
```

### Lazy Resolution

Use `core.Lazy[T]` for deferred resolution:

```go
wire := core.Lazy[*UserService](app)
svc, err := wire.Get()
```

### Graph Introspection

The container exposes its dependency graph for diagnostics:

```go
core.WriteGraph(app, os.Stdout)
data, _ := app.Container().GraphJSON()
```

### Safety

- Duplicate provider registration is rejected.
- Circular dependencies are detected at resolution time.
- Missing dependencies produce clear error messages naming the provider and the missing type.

## Near-Term Next Steps

1. Keep docs and examples aligned with the beta API.
2. Add optional data examples for GORM and sqlc if they become useful to users.
3. Continue release hardening with compatibility and CI checks.

## Current Verification

Run these commands before handing off scaffold changes:

```bash
go test ./... -count=1
go vet ./...
go test ./... -shuffle=on -count=3
```