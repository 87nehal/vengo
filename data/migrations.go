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
	Table  string
	Prefix string
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

	if err := ensureMigrationsTable(ctx, db, table); err != nil {
		return err
	}

	for _, version := range versions {
		applied, err := migrationApplied(ctx, db, table, version)
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
		if err := applyMigration(ctx, db, table, version, string(contents)); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB, table string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (version TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`, table)
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, table, version string) (bool, error) {
	query := fmt.Sprintf(`SELECT version FROM %s WHERE version = ?`, table)
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

func applyMigration(ctx context.Context, db *sql.DB, table, version, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, sqlText); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	insert := fmt.Sprintf(`INSERT INTO %s (version) VALUES (?)`, table)
	if _, err = tx.ExecContext(ctx, insert, version); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
