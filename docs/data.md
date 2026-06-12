# Data Access

The `data` package adds explicit `database/sql` support while keeping `core` independent of database concerns.

## Driver setup

Vengo does not import a concrete database driver. Applications choose and import one:

```go
import _ "modernc.org/sqlite" // or a Postgres/MySQL driver
```

## Configuration

Bind database settings through the normal config system:

```toml
database.driver = "sqlite"
database.dsn = "file:app.db"
database.max-open-conns = 10
database.max-idle-conns = 2
database.conn-max-lifetime = "30m"
database.conn-max-idle-time = "5m"
database.slow-query-threshold = "200ms"
database.migrations-table = "schema_migrations"
database.migrations-path-prefix = "migrations"
```

## Registering the module

```go
app := core.New("app", data.New())
app.SetConfig(cfg)
if err := app.Start(ctx); err != nil {
    log.Fatal(err)
}
```

The module registers the named `*sql.DB` service as `data.DBServiceName` (`data.db`).

```go
db, err := core.Get[*sql.DB](app, data.DBServiceName)
```

## Health checks

Wire the database health indicator into actuator health checks when constructing the health module:

```go
health := actuator.NewHealth(actuator.WithIndicators(data.NewHealthIndicator(db)))
```

If the data module creates the DB, retrieve it after configuration/start and pass it where you build health checks in your app setup.

## Transactions

Use `TxManager.WithTx(ctx, fn)` for commit/rollback handling:

```go
manager := data.NewTxManager(db)
err := manager.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `INSERT INTO widgets (name) VALUES (?)`, "demo")
    return err
})
```

The transaction is committed when the callback returns nil. Callback errors roll back. Panics also roll back and are rethrown. `data.TxFromContext(ctx)` retrieves the active transaction from the callback context.

## Migrations

Embed SQL migrations and pass the filesystem to the module:

```go
//go:embed migrations/*.sql
var migrations embed.FS

app := core.New("app", data.New(data.WithMigrations(migrations)))
```

On startup, `.sql` files under `database.migrations-path-prefix` are applied lexicographically. Applied filenames are recorded in `schema_migrations` or the configured table. The migration table name is validated to prevent unsafe dynamic SQL.

## Slow query logging

Raw `*sql.DB` calls cannot be globally intercepted. When `database.slow-query-threshold` is greater than zero, the module registers `*data.InstrumentedDB` as `data.InstrumentedDBServiceName` (`data.instrumented-db`). Use this wrapper for explicit slow query logging:

```go
instrumented, _ := core.Get[*data.InstrumentedDB](app, data.InstrumentedDBServiceName)
_, err := instrumented.ExecContext(ctx, query, args...)
```

Direct calls to the named `*sql.DB` bypass slow query logging.

## Step-by-step testing

Run component tests in this order while developing data integrations:

```bash
go test ./data -run TestConfig
go test ./data -run TestModule
go test ./data -run TestWithTxCommit
go test ./data -run TestWithTxRollback
go test ./data -run TestApplyMigrations
go test ./data -run TestHealthIndicator
go test ./data -run TestSlowQueryLogging
go run ./examples/data-sql
go test ./...
go vet ./...
```
