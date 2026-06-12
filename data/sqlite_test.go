package data

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"

	"github.com/87nehal/vengo/core"
	_ "modernc.org/sqlite"
)

func TestSQLite_DialectDetection(t *testing.T) {
	d, err := DialectForDriver("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "sqlite" {
		t.Fatalf("dialect name = %q, want sqlite", d.Name())
	}
}

func TestSQLite_CreateMigrationsTable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:TestSQLite_CreateMigrationsTable?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	dialect := SQLiteDialect{}
	sql := dialect.CreateMigrationsTableSQL("schema_migrations")
	if _, err := db.ExecContext(context.Background(), sql); err != nil {
		t.Fatalf("create migrations table: %v", err)
	}
}

func TestSQLite_ApplyMigrations(t *testing.T) {
	db, err := sql.Open("sqlite", "file:TestSQLite_ApplyMigrations?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	dialect := SQLiteDialect{}

	fsys := fstest.MapFS{
		"migrations/001_create_widgets.sql": {Data: []byte(`
			CREATE TABLE IF NOT EXISTS widgets (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)},
		"migrations/002_insert_widget.sql": {Data: []byte(`
			INSERT INTO widgets (name) VALUES ('test_widget')
		`)},
	}

	ctx := context.Background()

	if err := ApplyMigrations(ctx, db, fsys, MigrationOptions{
		Table:   "schema_migrations",
		Prefix:  "migrations",
		Dialect: dialect,
	}); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&count); err != nil {
		t.Fatalf("count widgets: %v", err)
	}
	if count != 1 {
		t.Fatalf("widget count = %d, want 1", count)
	}

	if err := ApplyMigrations(ctx, db, fsys, MigrationOptions{
		Table:   "schema_migrations",
		Prefix:  "migrations",
		Dialect: dialect,
	}); err != nil {
		t.Fatalf("second apply (idempotent): %v", err)
	}

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&count); err != nil {
		t.Fatalf("count widgets after second apply: %v", err)
	}
	if count != 1 {
		t.Fatalf("widget count after second apply = %d, want 1", count)
	}
}

func TestSQLite_WithModule(t *testing.T) {
	dialect := SQLiteDialect{}

	fsys := fstest.MapFS{
		"migrations/001_create_widgets.sql": {Data: []byte(`
			CREATE TABLE IF NOT EXISTS widgets (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)},
	}

	app := core.New("sqlite-module", New(
		WithConfig(Config{
			Driver:               "sqlite",
			DSN:                  "file:TestSQLite_WithModule?mode=memory&cache=shared",
			MaxOpenConns:         1,
			MaxIdleConns:         1,
			MigrationsTable:      "schema_migrations",
			MigrationsPathPrefix: "migrations",
			Dialect:              dialect,
		}),
		WithMigrations(fsys),
	))

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer func() {
		_ = app.Stop(ctx)
	}()

	db, err := core.Get[*sql.DB](app, DBServiceName)
	if err != nil {
		t.Fatalf("get db: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	d, err := core.Get[Dialect](app, DialectServiceName)
	if err != nil {
		t.Fatalf("get dialect: %v", err)
	}
	if d.Name() != "sqlite" {
		t.Fatalf("registered dialect = %q, want sqlite", d.Name())
	}
}

func TestSQLite_AutoDetectDialect(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/001_create_widgets.sql": {Data: []byte(`
			CREATE TABLE IF NOT EXISTS widgets (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)},
	}

	app := core.New("sqlite-autodetect", New(
		WithConfig(Config{
			Driver:               "sqlite",
			DSN:                  "file:TestSQLite_AutoDetectDialect?mode=memory&cache=shared",
			MaxOpenConns:         1,
			MaxIdleConns:         1,
			MigrationsTable:      "schema_migrations",
			MigrationsPathPrefix: "migrations",
		}),
		WithMigrations(fsys),
	))

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start app: %v", err)
	}
	defer func() {
		_ = app.Stop(ctx)
	}()

	d, err := core.Get[Dialect](app, DialectServiceName)
	if err != nil {
		t.Fatalf("get dialect: %v", err)
	}
	if d.Name() != "sqlite" {
		t.Fatalf("auto-detected dialect = %q, want sqlite", d.Name())
	}
}
