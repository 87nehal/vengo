package data

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
)

var migrationTableNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// MigrationOptions controls migration discovery and tracking.
type MigrationOptions struct {
	Table   string
	Prefix  string
	Dialect Dialect
}

// ApplyMigrations applies unapplied .sql files from fsys in lexicographic order.
func ApplyMigrations(ctx context.Context, db *sql.DB, fsys fs.FS, opts MigrationOptions) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if fsys == nil {
		return fmt.Errorf("migrations filesystem is nil")
	}

	table := opts.Table
	if table == "" {
		table = "schema_migrations"
	}
	if !migrationTableNamePattern.MatchString(table) {
		return fmt.Errorf("invalid migrations table name %q", table)
	}

	dialect := opts.Dialect
	if dialect == nil {
		dialect = SQLiteDialect{}
	}

	prefix := strings.Trim(strings.ReplaceAll(opts.Prefix, `\\`, `/`), `/`)
	if prefix == "" {
		prefix = "migrations"
	}

	entries, err := fs.ReadDir(fsys, prefix)
	if err != nil {
		if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("read migrations: %w", err)
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Strings(versions)

	if err := ensureMigrationsTable(ctx, db, table, dialect); err != nil {
		return err
	}

	for _, version := range versions {
		applied, err := migrationApplied(ctx, db, table, version, dialect)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		contents, err := fs.ReadFile(fsys, path.Join(prefix, version))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if err := applyMigration(ctx, db, table, version, string(contents), dialect); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB, table string, dialect Dialect) error {
	query := dialect.CreateMigrationsTableSQL(table)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, table, version string, dialect Dialect) (bool, error) {
	query := fmt.Sprintf(`SELECT version FROM %s WHERE version = %s`, table, dialect.Placeholder(1))
	var existing string
	err := db.QueryRowContext(ctx, query, version).Scan(&existing)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, fmt.Errorf("check migration %s: %w", version, err)
}

func applyMigration(ctx context.Context, db *sql.DB, table, version, sqlText string, dialect Dialect) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	statements := splitSQL(sqlText)
	for _, stmt := range statements {
		if _, err = tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
	}

	insert := fmt.Sprintf(`INSERT INTO %s (version) VALUES (%s)`, table, dialect.Placeholder(1))
	if _, err = tx.ExecContext(ctx, insert, version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}

func splitSQL(sqlText string) []string {
	var statements []string
	var current strings.Builder
	var inSingleQuote bool
	var inDoubleQuote bool
	var inBacktick bool
	var inDollarQuote bool
	var inLineComment bool
	var inBlockComment bool

	runes := []rune(sqlText)
	length := len(runes)
	for i := 0; i < length; i++ {
		r := runes[i]
		next := rune(0)
		if i+1 < length {
			next = runes[i+1]
		}

		if inLineComment {
			if r == '\n' {
				inLineComment = false
			}
			current.WriteRune(r)
			continue
		}

		if inBlockComment {
			if r == '*' && next == '/' {
				inBlockComment = false
				current.WriteRune(r)
				current.WriteRune(next)
				i++
				continue
			}
			current.WriteRune(r)
			continue
		}

		// Check for comment start
		if r == '-' && next == '-' && !inSingleQuote && !inDoubleQuote && !inBacktick && !inDollarQuote {
			inLineComment = true
			current.WriteRune(r)
			current.WriteRune(next)
			i++
			continue
		}
		if r == '/' && next == '*' && !inSingleQuote && !inDoubleQuote && !inBacktick && !inDollarQuote {
			inBlockComment = true
			current.WriteRune(r)
			current.WriteRune(next)
			i++
			continue
		}

		// Check backslash escape inside quotes
		if r == '\\' && (inSingleQuote || inDoubleQuote) {
			current.WriteRune(r)
			if i+1 < length {
				current.WriteRune(runes[i+1])
				i++
			}
			continue
		}

		// Check dollar quote $$
		if r == '$' && next == '$' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			inDollarQuote = !inDollarQuote
			current.WriteRune(r)
			current.WriteRune(next)
			i++
			continue
		}

		// check quotes
		if r == '\'' && !inDoubleQuote && !inBacktick && !inDollarQuote {
			inSingleQuote = !inSingleQuote
		} else if r == '"' && !inSingleQuote && !inBacktick && !inDollarQuote {
			inDoubleQuote = !inDoubleQuote
		} else if r == '`' && !inSingleQuote && !inDoubleQuote && !inDollarQuote {
			inBacktick = !inBacktick
		}

		if r == ';' && !inSingleQuote && !inDoubleQuote && !inBacktick && !inDollarQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		} else {
			current.WriteRune(r)
		}
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}
	return statements
}

