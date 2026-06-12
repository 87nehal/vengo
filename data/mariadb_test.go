package data

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"testing/fstest"

	"github.com/87nehal/vengo/core"
	_ "github.com/go-sql-driver/mysql"
)

func mariadbDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MARIADB_TEST_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/vengo_test"
	}
	return dsn
}

func openMariaDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := mariadbDSN(t)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mariadb: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping mariadb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func cleanupMariaDB(t *testing.T, db *sql.DB, tables ...string) {
	t.Helper()
	for _, table := range tables {
		_, _ = db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	}
}

func TestMariaDB_DialectDetection(t *testing.T) {
	d, err := DialectForDriver("mysql")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name() != "mysql" {
		t.Fatalf("dialect name = %q, want mysql", d.Name())
	}
	sql := d.CreateMigrationsTableSQL("schema_migrations")
	if sql == "" {
		t.Fatal("empty CREATE TABLE SQL")
	}
	t.Logf("MySQL dialect SQL: %s", sql)
}

func TestMariaDB_CreateMigrationsTable(t *testing.T) {
	if os.Getenv("MARIADB_TEST_DSN") == "" {
		dsn := "root:root@tcp(127.0.0.1:3306)/"
		adminDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Skipf("cannot connect to MariaDB: %v", err)
		}
		if err := adminDB.PingContext(context.Background()); err != nil {
			t.Skipf("cannot ping MariaDB: %v", err)
		}
		_, _ = adminDB.ExecContext(context.Background(), "CREATE DATABASE IF NOT EXISTS vengo_test")
		_ = adminDB.Close()
	}

	db := openMariaDB(t)
	cleanupMariaDB(t, db, "schema_migrations", "test_migrations", "widgets")

	dialect := MySQLDialect{}

	t.Run("default_table", func(t *testing.T) {
		sql := dialect.CreateMigrationsTableSQL("schema_migrations")
		if _, err := db.ExecContext(context.Background(), sql); err != nil {
			t.Fatalf("create migrations table: %v", err)
		}
	})

	t.Run("custom_table", func(t *testing.T) {
		sql := dialect.CreateMigrationsTableSQL("test_migrations")
		if _, err := db.ExecContext(context.Background(), sql); err != nil {
			t.Fatalf("create custom migrations table: %v", err)
		}
	})
}

func TestMariaDB_ApplyMigrations(t *testing.T) {
	if os.Getenv("MARIADB_TEST_DSN") == "" {
		dsn := "root:root@tcp(127.0.0.1:3306)/"
		adminDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Skipf("cannot connect to MariaDB: %v", err)
		}
		if err := adminDB.PingContext(context.Background()); err != nil {
			t.Skipf("cannot ping MariaDB: %v", err)
		}
		_, _ = adminDB.ExecContext(context.Background(), "CREATE DATABASE IF NOT EXISTS vengo_test")
		_ = adminDB.Close()
	}

	db := openMariaDB(t)
	cleanupMariaDB(t, db, "schema_migrations", "widgets")

	dialect := MySQLDialect{}

	fsys := fstest.MapFS{
		"migrations/001_create_widgets.sql": {Data: []byte(`
			CREATE TABLE IF NOT EXISTS widgets (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
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

	var migrationCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 2 {
		t.Fatalf("migration count = %d, want 2", migrationCount)
	}
}

func TestMariaDB_HealthIndicator(t *testing.T) {
	if os.Getenv("MARIADB_TEST_DSN") == "" {
		dsn := "root:root@tcp(127.0.0.1:3306)/"
		adminDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Skipf("cannot connect to MariaDB: %v", err)
		}
		if err := adminDB.PingContext(context.Background()); err != nil {
			t.Skipf("cannot ping MariaDB: %v", err)
		}
		_ = adminDB.Close()
	}

	db := openMariaDB(t)
	ind := NewHealthIndicator(db)
	ctx := context.Background()
	h := ind.Health(ctx)
	if h.Status != "UP" {
		t.Fatalf("health status = %q, want UP", h.Status)
	}
}

func TestMariaDB_PoolStats(t *testing.T) {
	if os.Getenv("MARIADB_TEST_DSN") == "" {
		dsn := "root:root@tcp(127.0.0.1:3306)/"
		adminDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Skipf("cannot connect to MariaDB: %v", err)
		}
		if err := adminDB.PingContext(context.Background()); err != nil {
			t.Skipf("cannot ping MariaDB: %v", err)
		}
		_ = adminDB.Close()
	}

	db := openMariaDB(t)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	stats := DBStats(db)
	if stats.MaxOpenConnections != 10 {
		t.Fatalf("MaxOpenConnections = %d, want 10", stats.MaxOpenConnections)
	}

	ind := NewPoolStatsIndicator(db)
	ctx := context.Background()
	h := ind.Health(ctx)
	if h.Status != "UP" {
		t.Fatalf("pool health status = %q, want UP", h.Status)
	}
	if _, ok := h.Details["max_open_connections"]; !ok {
		t.Fatal("missing max_open_connections in pool stats")
	}
	if _, ok := h.Details["open_connections"]; !ok {
		t.Fatal("missing open_connections in pool stats")
	}
}

func TestMariaDB_WithModule(t *testing.T) {
	if os.Getenv("MARIADB_TEST_DSN") == "" {
		dsn := "root:root@tcp(127.0.0.1:3306)/"
		adminDB, err := sql.Open("mysql", dsn)
		if err != nil {
			t.Skipf("cannot connect to MariaDB: %v", err)
		}
		if err := adminDB.PingContext(context.Background()); err != nil {
			t.Skipf("cannot ping MariaDB: %v", err)
		}
		_, _ = adminDB.ExecContext(context.Background(), "CREATE DATABASE IF NOT EXISTS vengo_test")
		_ = adminDB.Close()
	}

	dsn := mariadbDSN(t)
	dialect := MySQLDialect{}

	fsys := fstest.MapFS{
		"migrations/001_create_widgets.sql": {Data: []byte(`
			CREATE TABLE IF NOT EXISTS widgets (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`)},
	}

	app := core.New("mariadb-module", New(
		WithConfig(Config{
			Driver:               "mysql",
			DSN:                  dsn,
			MaxOpenConns:         5,
			MaxIdleConns:         2,
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
	if d.Name() != "mysql" {
		t.Fatalf("registered dialect = %q, want mysql", d.Name())
	}
}
