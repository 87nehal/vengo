# Architecture

The first implementation uses a small set of packages instead of a multi-module workspace. That keeps local development simple while the public API is still moving.

## Packages

- `core`: application lifecycle, module registration, hooks, and named services.
- `config`: configuration sources (TOML, JSON, env, embed.FS), precedence, typed struct binding, validation, and redacted reporting.
- `autoconfigure`: conditional configuration registry.
- `web`: `net/http` based server module.
- `actuator`: health endpoint module.
- `starter/web`: convenience package for creating the web module.
- `cmd/vengo`: CLI entry point for version checks and project generation.
- `examples/hello`: runnable example application.

## Design Rules

1. Keep the core independent of HTTP, database, cloud, and security packages.
2. Prefer explicit module registration over package scanning.
3. Prefer named services and typed lookup until a fuller dependency graph exists.
4. Keep production behavior observable through logs, reports, and future explain commands.
5. Add code generation only after benchmarks show a real need.

## Near-Term Next Steps

1. Add route listing diagnostics.
2. Add dependency graph validation.
3. Add CLI `doctor` and `explain` commands.
4. Add middleware and structured error helpers to the web module.

## Current Verification

Run these commands before handing off scaffold changes:

```bash
go test ./...
go vet ./...
```