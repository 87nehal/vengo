# CLI

The `cmd/vengo` binary is a small command-line tool for project scaffolding, configuration inspection, and diagnostics. The same `run` function is used by the entry point and the tests, so the documented commands are covered by step-by-step tests.

## Commands

### `vengo version`

Prints the framework version.

```bash
vengo version
# vengo 0.1.0-dev
```

### `vengo help`

Lists all available commands. Recognized under any of `help`, `-h`, `--help`.

### `vengo new <dir> [module] [--modules=web,data,auth]`

Scaffolds a new project directory:

- `go.mod` with the chosen module path (or the directory name as fallback).
- `main.go` that wires `core.App` with the requested modules and uses the typed config pattern from `config.Bind`.
- `application.toml` with a default `[app]` section.

The `--modules` flag accepts a comma-separated list. Supported modules: `web`, `data`, `auth`. If the flag is omitted, `web` is selected by default. The `auth` option adds the security module with development session config that should be changed before production use.

Generated `go.mod` includes a `replace github.com/87nehal/vengo => <path>` directive so generated projects build against the local checkout. Override the replace path with `VENGO_LOCAL_PATH`.

```bash
vengo new orders-api github.com/you/orders-api
vengo new orders-api --modules=web
vengo new orders-api --modules=web,data,auth
```

### `vengo config [profile]`

Loads the resolved configuration using the standard `application.toml` chain (`application-{profile}.toml` overrides, then `APP_`-prefixed env vars). Prints each key, value, and source, with sensitive keys redacted.

```bash
vengo config            # uses $APP_PROFILE / $VENGO_PROFILE
vengo config prod       # explicit profile
```

### `vengo deps`

Reads `vengo-deps.json` from the current directory and prints a human-readable dependency graph. The file is produced by an app via `app.Container().GraphJSON()`.

```go
data, _ := app.Container().GraphJSON()
os.WriteFile("vengo-deps.json", data, 0o644)
```

### `vengo routes`

Reads `vengo-routes.json` from the current directory and prints registered HTTP routes sorted by path/method. The file is produced by an app via `webServer.RoutesJSON()`.

```go
data, _ := webServer.RoutesJSON()
os.WriteFile("vengo-routes.json", data, 0o644)
```

### `vengo doctor`

Runs a set of sanity checks and prints one line per check:

- `go version` — runs `go version`
- `module path` — reads `go.mod` from the current directory
- `vengo version` — reports the CLI's compiled-in version
- `env` — shows the active profile (or notes that none is set)

Exits with status `1` if any check fails.

```bash
vengo doctor
# Vengo Doctor
# ==================================================
#   [OK]   go version           go version go1.22.0 linux/amd64
#   [OK]   module path          github.com/example/orders-api
#   [OK]   vengo version        0.1.0-dev
#   [OK]   env                  active profile = prod
# ==================================================
# all checks passed
```

### `vengo run [path] [--build=cmd] [--no-run] [--build-only] [--debounce=300ms]`

Chdirs to `path` (default `.`), runs the build command, then runs the produced binary. Watches `.go`, `.toml`, `.json`, `.yaml`, and `.yml` files; on change, kills the running process, rebuilds, and restarts. The build command is run through `cmd /c` on Windows and `sh -c` on other platforms, so shell features like pipes and env-var expansion work.

Flags:

- `--build` — the shell build command (default `go build -o vengo-app .`).
- `--no-run` — build but do not start the binary.
- `--build-only` — same as `--no-run`, kept for symmetry with other tools.
- `--debounce` — debounce window for the file watcher (default `300ms`).

`Ctrl+C` stops the watcher and the running process.

```bash
vengo run .
vengo run ./cmd/api --build='go build -o ./bin/api ./cmd/api'
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | Operational error (file not found, build failed, doctor check failed, etc.) |
| 2    | Usage error (missing arguments, unknown command, unknown module flag value) |
