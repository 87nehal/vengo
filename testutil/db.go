package testutil

import (
	"context"
	"database/sql"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/87nehal/vengo/data"
)

// OpenTestDB opens a database connection and registers its closing in t.Cleanup.
func OpenTestDB(t testing.TB, driver, dsn string) *sql.DB {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// RunMigrations runs all Vengo migrations from the given filesystem on the database.
func RunMigrations(t testing.TB, db *sql.DB, fsys fs.FS) {
	driver := getDriverName(db)
	dialect, err := data.DialectForDriver(driver)
	if err != nil {
		t.Fatalf("determine dialect: %v", err)
	}
	ctx := context.Background()
	opts := data.MigrationOptions{
		Table:   "schema_migrations",
		Prefix:  "migrations",
		Dialect: dialect,
	}
	if err := data.ApplyMigrations(ctx, db, fsys, opts); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
}

func getDriverName(db *sql.DB) string {
	if db == nil || db.Driver() == nil {
		return "sqlite"
	}
	tName := reflect.TypeOf(db.Driver()).String()
	tName = strings.ToLower(tName)
	switch {
	case strings.Contains(tName, "mysql") || strings.Contains(tName, "mariadb"):
		return "mysql"
	case strings.Contains(tName, "pq") || strings.Contains(tName, "pgx") || strings.Contains(tName, "postgres"):
		return "postgres"
	case strings.Contains(tName, "sqlite"):
		return "sqlite"
	default:
		return "sqlite"
	}
}
