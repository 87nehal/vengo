# Actuator

The `actuator` package provides production-readiness modules for health checks, application info, metrics, structured logging, and distributed tracing hooks. All modules register as `core.Module` implementations and hook into the web server automatically.

## Health

### Basic Usage

```go
health := actuator.NewHealth()
app := core.New("myapp", server, health)
```

Registers `GET /actuator/health` returning `{"status":"UP"}` or `{"status":"DOWN","checks":{...}}`.

### Custom Checks

```go
health := actuator.NewHealth(
    actuator.WithChecks(actuator.Check{
        Name: "database",
        Check: func(ctx context.Context) error {
            return db.PingContext(ctx)
        },
    }),
)
```

### Health Indicators

For structured responses with details, implement `HealthIndicator`:

```go
type HealthIndicator interface {
    HealthIndicatorName() string
    Health(ctx context.Context) Health
}
```

```go
type DBIndicator struct{ db *sql.DB }

func (d *DBIndicator) HealthIndicatorName() string { return "database" }
func (d *DBIndicator) Health(ctx context.Context) actuator.Health {
    if err := d.db.PingContext(ctx); err != nil {
        return actuator.Health{
            Status:  actuator.StatusDown,
            Details: map[string]any{"error": err.Error()},
        }
    }
    return actuator.Health{Status: actuator.StatusUp}
}

health := actuator.NewHealth(
    actuator.WithIndicators(&DBIndicator{db: db}),
)
```

### Liveness and Readiness Probes

Separate endpoints for Kubernetes-style probes:

- `GET /actuator/health/liveness` — only liveness indicators
- `GET /actuator/health/readiness` — only readiness indicators

Implement `ProbedIndicator` to control which probe an indicator participates in:

```go
type ProbedIndicator interface {
    HealthIndicator
    ProbeType() ProbeType // ProbeLiveness, ProbeReadiness, or ProbeBoth
}
```

Indicators that don't implement `ProbedIndicator` appear in all probe endpoints.

### Health Options

| Option | Description |
|--------|-------------|
| `WithPath(path)` | Custom endpoint path |
| `WithChecks(checks...)` | Add closure-based checks |
| `WithIndicators(indicators...)` | Add structured health indicators |
| `WithEnabled(bool)` | Enable/disable the endpoint |

## Info

Application metadata endpoint:

```go
info := actuator.NewInfo(
    actuator.WithVersion("1.2.3"),
    actuator.WithCommit("abc123"),
    actuator.WithBuild("2024-01-15"),
    actuator.WithInfoExtra("environment", "production"),
)
```

`GET /actuator/info` returns:

```json
{
  "name": "myapp",
  "version": "1.2.3",
  "commit": "abc123",
  "build": "2024-01-15",
  "environment": "production"
}
```

The app name is read from `app.Name()` automatically.

### Info Options

| Option | Description |
|--------|-------------|
| `WithInfoPath(path)` | Custom endpoint path |
| `WithVersion(v)` | Application version |
| `WithCommit(hash)` | Git commit hash |
| `WithBuild(timestamp)` | Build timestamp |
| `WithInfoExtra(key, value)` | Arbitrary extra field |
| `WithInfoEnabled(bool)` | Enable/disable the endpoint |

## Metrics

Thread-safe counters and gauges:

```go
metrics := actuator.NewMetrics()
reqs := actuator.NewCounter("http.requests")
open := actuator.NewGauge("connections.open")

metrics.RegisterCounter(reqs)
metrics.RegisterGauge(open)
```

`GET /actuator/metrics` returns a snapshot of all registered metrics:

```json
{"http.requests": 142, "connections.open": 7}
```

### Counter

Monotonically increasing value:

```go
c := actuator.NewCounter("requests")
c.Inc()
c.Add(5)
c.Value() // 6
```

### Gauge

Fluctuating value:

```go
g := actuator.NewGauge("temperature")
g.Set(72)
g.Inc()
g.Dec()
g.Add(3)
g.Value() // 75
```

Both `Counter` and `Gauge` are safe for concurrent use (`sync/atomic`).

### Metrics Options

| Option | Description |
|--------|-------------|
| `WithMetricsPath(path)` | Custom endpoint path |
| `WithMetricsEnabled(bool)` | Enable/disable the endpoint |

## Logging

Structured logging module using `log/slog`:

```go
logging := actuator.NewLogging()
app := core.New("myapp", server, logging)
```

Creates a JSON slog logger on stderr (default) and installs `web.RequestLogger` middleware on the server automatically.

### Custom Logger

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

logging := actuator.NewLogging(
    actuator.WithLogger(logger),
    actuator.WithRequestLogging(false), // disable auto-installed middleware
)
```

### Retrieving the Logger

```go
if logger, ok := actuator.LoggerFromApp(app); ok {
    logger.Info("app started", "port", 8080)
}
```

### Logging Options

| Option | Description |
|--------|-------------|
| `WithLogger(logger)` | Custom `*slog.Logger` |
| `WithLogLevel(level)` | Log level for the default logger |
| `WithLoggingEnabled(bool)` | Enable/disable the entire module |
| `WithRequestLogging(bool)` | Enable/disable auto-installed request logging middleware |

## Telemetry

OpenTelemetry-compatible tracing hooks with no-op defaults. No external dependencies required.

```go
telemetry := actuator.NewTelemetry()
app := core.New("myapp", server, telemetry)
```

Installs `TracingMiddleware` on the server, which creates a span per request with `http.method`, `http.url`, and `http.remote` attributes.

### Tracer and Span Interfaces

```go
type Tracer interface {
    Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
    End()
    SetAttribute(key string, value any)
    RecordError(err error)
    SetStatus(status SpanStatus, description string)
    SpanID() string
    TraceID() string
}
```

The default implementations are no-ops — zero cost when not configured.

### Plugging In a Real Tracer

```go
telemetry := actuator.NewTelemetry(
    actuator.WithTracer(myOTelTracer),
)
```

### Accessing Spans From Handlers

```go
server.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
    span := actuator.SpanFromContext(r.Context())
    span.SetAttribute("user.id", userID)
    // ...
})
```

### Telemetry Options

| Option | Description |
|--------|-------------|
| `WithTracer(tracer)` | Custom tracer implementation |
| `WithTelemetryEnabled(bool)` | Enable/disable the entire module |
| `WithTracingMiddleware(bool)` | Enable/disable auto-installed middleware |

## Configurable Exposure

Every actuator module supports disabling its endpoint:

```go
info := actuator.NewInfo(actuator.WithInfoEnabled(false))       // no /actuator/info
metrics := actuator.NewMetrics(actuator.WithMetricsEnabled(false)) // no /actuator/metrics
health := actuator.NewHealth(actuator.WithEnabled(false))        // no /actuator/health
logging := actuator.NewLogging(actuator.WithLoggingEnabled(false)) // no logging module
telemetry := actuator.NewTelemetry(actuator.WithTelemetryEnabled(false)) // no tracing
```

All endpoints are enabled by default. When disabled, `Configure()` returns early and no routes are registered.

## Complete Example

```go
server := web.New(":8080")
server.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
    web.WriteJSON(w, 200, map[string]string{"message": "hello"})
})

app := core.New("myapp", server,
    actuator.NewHealth(
        actuator.WithChecks(actuator.Check{
            Name:  "self",
            Check: func(context.Context) error { return nil },
        }),
    ),
    actuator.NewInfo(
        actuator.WithVersion("1.0.0"),
        actuator.WithCommit("abc123"),
    ),
    actuator.NewMetrics(),
    actuator.NewLogging(),
    actuator.NewTelemetry(),
)

if err := app.Start(context.Background()); err != nil {
    log.Fatal(err)
}
```
