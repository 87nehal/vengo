package data

import "fmt"

const (
	MySQL    = "mysql"
	MariaDB  = "mariadb"
	Postgres = "postgres"
	SQLite   = "sqlite"
)

// Dialect abstracts database-specific SQL differences across SQLite, MySQL/MariaDB, and PostgreSQL.
type Dialect interface {
	Name() string
	CreateMigrationsTableSQL(table string) string
	Placeholder(n int) string
}

// DialectForDriver returns the appropriate Dialect for the given database/sql driver name.
func DialectForDriver(driver string) (Dialect, error) {
	switch driver {
	case "sqlite", "sqlite3":
		return SQLiteDialect{}, nil
	case "mysql", "mariadb":
		return MySQLDialect{}, nil
	case "postgres", "pgx":
		return PostgresDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q: supported drivers are sqlite, mysql, mariadb, postgres, pgx", driver)
	}
}

// SQLiteDialect generates SQL compatible with SQLite.
type SQLiteDialect struct{}

func (SQLiteDialect) Name() string { return "sqlite" }

func (SQLiteDialect) CreateMigrationsTableSQL(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`, table)
}

func (SQLiteDialect) Placeholder(n int) string { return "?" }

// MySQLDialect generates SQL compatible with MySQL and MariaDB.
type MySQLDialect struct{}

func (MySQLDialect) Name() string { return "mysql" }

func (MySQLDialect) CreateMigrationsTableSQL(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`, table)
}

func (MySQLDialect) Placeholder(n int) string { return "?" }

// MariaDBDialect is an alias for MySQLDialect.
type MariaDBDialect = MySQLDialect

// PostgresDialect generates SQL compatible with PostgreSQL.
type PostgresDialect struct{}

func (PostgresDialect) Name() string { return "postgres" }

func (PostgresDialect) CreateMigrationsTableSQL(table string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version VARCHAR(255) PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`, table)
}

func (PostgresDialect) Placeholder(n int) string { return fmt.Sprintf("$%d", n) }
