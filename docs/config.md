# Configuration

The `config` package provides a layered configuration system that loads values from multiple sources, merges them with last-source-wins precedence, and binds them into typed Go structs.

## Sources

Configuration values come from `Source` implementations. Each source has a name and produces a flat `map[string]string` of key-value pairs.

### Built-in Sources

| Source | Description |
|--------|-------------|
| `FileSource` | Reads `.toml` or `.json` files from disk |
| `EmbedSource` | Reads `.toml` or `.json` files from `embed.FS` or any `fs.FS` |
| `EnvSource` | Reads environment variables with an optional prefix |
| `MapSource` | In-memory map, useful for tests and defaults |

### File Source

```go
source := config.NewFileSource("application.toml")
```

Nested TOML tables are flattened into dot-separated keys:

```toml
[server]
port = 8080
host = "localhost"
```

Becomes:

```
server.port = "8080"
server.host = "localhost"
```

### Embed Source

Ship configuration inside your binary:

```go
//go:embed application.toml
var configFS embed.FS

source := config.NewEmbedSource(configFS, "application.toml")
```

### Environment Source

```go
source := config.NewEnvSource("APP_")
```

Environment variables are lowercased and underscores become dots:

```
APP_SERVER_PORT=9090  →  server.port = "9090"
```

## Loading and Precedence

Use `config.Load` with sources in order of increasing priority. Later sources override earlier ones:

```go
cfg, err := config.Load(ctx,
    config.NewMapSource("defaults", map[string]string{"server.port": "8080"}),
    config.NewFileSource("application.toml"),
    config.NewEnvSource("APP_"),
)
```

### Default Loading

`LoadDefaults` builds the standard source chain automatically:

```go
cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
```

This loads:

1. `application.toml` (from `.` or `./config/`)
2. `application-{profile}.toml` (if a profile is active)
3. Environment variables with `APP_` prefix

### Profiles

Set the active profile via environment variables:

```bash
APP_PROFILE=prod    # takes precedence
VENGO_PROFILE=prod  # fallback
```

Or pass it explicitly:

```go
cfg, err := config.LoadDefaults(ctx, "prod")
```

## Typed Binding

Bind configuration values into Go structs using struct tags:

```go
type ServerConfig struct {
    Port    int           `config:"server.port" default:"8080"`
    Host    string        `config:"server.host" default:"localhost"`
    Timeout time.Duration `config:"server.timeout" default:"30s"`
    Debug   bool          `config:"server.debug" default:"false"`
}

var serverCfg ServerConfig
if err := config.Bind(cfg, &serverCfg); err != nil {
    log.Fatal(err)
}
```

### Struct Tags

| Tag | Purpose |
|-----|---------|
| `config:"key"` | Explicit config key to look up |
| `default:"value"` | Fallback value when the key is not found |

If `config:"..."` is omitted, the lowercase field name is used as the key. For nested structs, the field name becomes a prefix:

```go
type AppConfig struct {
    Server ServerConfig
    DB     DatabaseConfig
}
```

`Server.Port` looks up `server.port`; `DB.Host` looks up `db.host` by default because the lowercase field name becomes the prefix. Use explicit `config:"database.host"` tags when you want a different key such as `database.host`.

### Supported Types

- `string`
- `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `time.Duration`

## Validation

After binding, validate the result:

```go
type ServerConfig struct {
    Port int    `config:"server.port" validate:"nonzero"`
    Host string `config:"server.host" validate:"required"`
}

var cfg ServerConfig
if err := config.BindAndValidate(appCfg, &cfg); err != nil {
    log.Fatal(err)
}
```

### Validation Tags

| Tag | Behavior |
|-----|----------|
| `required` | String must be non-empty; pointer/slice/map must be non-nil and non-empty |
| `nonzero` | Value must not be the zero value for its type |

### Custom Validation

Implement the `Validator` interface for complex rules:

```go
func (c *ServerConfig) Validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("port %d out of range", c.Port)
    }
    return nil
}
```

The `Validate()` method is called automatically by `config.Validate` and `config.BindAndValidate`.

## Core Integration

Register config on the app for use by modules:

```go
app := core.New("myapp", server, health)
app.SetConfig(cfg)
```

Retrieve it later:

```go
appCfg, err := config.FromApp(app)
// or bind directly:
var serverCfg ServerConfig
config.BindFromApp(app, &serverCfg)
```

## Introspection

### Report

Get all resolved values with their sources. Sensitive keys (containing `password`, `secret`, `token`, `credential`, `api.key`, `apikey`, `private.key`) are automatically redacted:

```go
for _, entry := range cfg.Report() {
    fmt.Printf("%s = %s [%s]\n", entry.Key, entry.Value, entry.Source)
}
```

### Source Lookup

Find where a specific value came from:

```go
source, ok := cfg.SourceOf("server.port")
// source = "env:APP_", ok = true
```

### Keys

List all resolved keys:

```go
keys := cfg.Keys()
```

## CLI

Dump the resolved configuration from the command line:

```bash
vengo config           # uses active profile from env
vengo config prod      # explicit profile
```

Secrets are redacted in the output.
