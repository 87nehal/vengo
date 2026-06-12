package data

import (
	"testing"
)

func TestDialectForDriver(t *testing.T) {
	tests := []struct {
		driver      string
		wantName    string
		wantErr     bool
		placeholder string
	}{
		{"sqlite", "sqlite", false, "?"},
		{"sqlite3", "sqlite", false, "?"},
		{"mysql", "mysql", false, "?"},
		{"mariadb", "mysql", false, "?"},
		{"postgres", "postgres", false, "$1"},
		{"pgx", "postgres", false, "$1"},
		{"oracle", "", true, ""},
		{"sqlserver", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			d, err := DialectForDriver(tt.driver)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("DialectForDriver(%q) expected error", tt.driver)
				}
				return
			}
			if err != nil {
				t.Fatalf("DialectForDriver(%q) unexpected error: %v", tt.driver, err)
			}
			if d.Name() != tt.wantName {
				t.Fatalf("dialect name = %q, want %q", d.Name(), tt.wantName)
			}
			if got := d.Placeholder(1); got != tt.placeholder {
				t.Fatalf("placeholder(1) = %q, want %q", got, tt.placeholder)
			}
		})
	}
}

func TestDialectCreateMigrationsTable(t *testing.T) {
	tests := []struct {
		dialect  Dialect
		table    string
		contains string
	}{
		{SQLiteDialect{}, "schema_migrations", "version TEXT PRIMARY KEY"},
		{MySQLDialect{}, "schema_migrations", "version VARCHAR(255) PRIMARY KEY"},
		{PostgresDialect{}, "schema_migrations", "version VARCHAR(255) PRIMARY KEY"},
		{MySQLDialect{}, "my_migrations", "my_migrations"},
	}
	for _, tt := range tests {
		t.Run(tt.dialect.Name()+"/"+tt.table, func(t *testing.T) {
			sql := tt.dialect.CreateMigrationsTableSQL(tt.table)
			if sql == "" {
				t.Fatal("CreateMigrationsTableSQL returned empty string")
			}
			if !containsString(sql, tt.contains) {
				t.Fatalf("SQL %q does not contain %q", sql, tt.contains)
			}
		})
	}
}

func TestDialectPlaceholders(t *testing.T) {
	mysql := MySQLDialect{}
	pg := PostgresDialect{}

	if mysql.Placeholder(1) != "?" {
		t.Fatalf("mysql placeholder(1) = %q, want ?", mysql.Placeholder(1))
	}
	if mysql.Placeholder(5) != "?" {
		t.Fatalf("mysql placeholder(5) = %q, want ?", mysql.Placeholder(5))
	}
	if pg.Placeholder(1) != "$1" {
		t.Fatalf("postgres placeholder(1) = %q, want $1", pg.Placeholder(1))
	}
	if pg.Placeholder(3) != "$3" {
		t.Fatalf("postgres placeholder(3) = %q, want $3", pg.Placeholder(3))
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
