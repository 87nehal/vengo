package data

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// InstrumentedDB wraps database/sql calls with explicit slow query logging.
type InstrumentedDB struct {
	inner     *sql.DB
	logger    *slog.Logger
	threshold time.Duration
}

// NewInstrumentedDB creates a database wrapper that logs calls taking at least threshold.
func NewInstrumentedDB(inner *sql.DB, logger *slog.Logger, threshold time.Duration) *InstrumentedDB {
	if logger == nil {
		logger = slog.Default()
	}
	return &InstrumentedDB{inner: inner, logger: logger, threshold: threshold}
}

// SQLDB returns the underlying database/sql handle.
func (db *InstrumentedDB) SQLDB() *sql.DB {
	if db == nil {
		return nil
	}
	return db.inner
}

// ExecContext executes a query and logs it when it exceeds the configured threshold.
func (db *InstrumentedDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := db.inner.ExecContext(ctx, query, args...)
	db.log(ctx, "exec", query, len(args), time.Since(start), err)
	return result, err
}

// QueryContext runs a query and logs it when it exceeds the configured threshold.
func (db *InstrumentedDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.inner.QueryContext(ctx, query, args...)
	db.log(ctx, "query", query, len(args), time.Since(start), err)
	return rows, err
}

// QueryRowContext runs a single-row query and logs dispatch time when it exceeds the configured threshold.
func (db *InstrumentedDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := db.inner.QueryRowContext(ctx, query, args...)
	db.log(ctx, "query_row", query, len(args), time.Since(start), nil)
	return row
}

func (db *InstrumentedDB) log(ctx context.Context, operation, query string, argCount int, duration time.Duration, err error) {
	if db == nil || db.logger == nil || db.threshold == 0 || duration < db.threshold {
		return
	}
	attrs := []any{
		"operation", operation,
		"duration", duration.String(),
		"threshold", db.threshold.String(),
		"query", query,
		"args", argCount,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	db.logger.WarnContext(ctx, "slow database query", attrs...)
}
