package data

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
)

type dialectContextKey struct{}

// ContextWithDialect returns a child context containing the dialect name.
func ContextWithDialect(ctx context.Context, dialectName string) context.Context {
	return context.WithValue(ctx, dialectContextKey{}, dialectName)
}

// DialectFromContext returns the dialect name from ctx, when present.
func DialectFromContext(ctx context.Context) string {
	if name, ok := ctx.Value(dialectContextKey{}).(string); ok {
		return name
	}
	return ""
}

// ExtContext represents the common query interface of *sql.DB, *sql.Tx, and InstrumentedDB.
type ExtContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// QueryOne executes a query and scans a single row into T.
func QueryOne[T any](ctx context.Context, db ExtContext, query string, args ...any) (T, error) {
	var result T
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return result, err
		}
		return result, sql.ErrNoRows
	}

	if err := scanInto(rows, &result); err != nil {
		return result, err
	}

	return result, rows.Err()
}

// QueryMany executes a query and scans all rows into a slice of T.
func QueryMany[T any](ctx context.Context, db ExtContext, query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var result T
		if err := scanInto(rows, &result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Exec executes a raw SQL statement and returns the result.
func Exec(ctx context.Context, db ExtContext, query string, args ...any) (sql.Result, error) {
	return db.ExecContext(ctx, query, args...)
}

// ExecNamed executes a raw SQL statement with named parameters.
func ExecNamed(ctx context.Context, db ExtContext, query string, arg any) (sql.Result, error) {
	parsedQuery, args, err := sqlx.Named(query, arg)
	if err != nil {
		return nil, err
	}

	dialectName := DialectFromContext(ctx)
	bindType := 1 // sqlx.QUESTION
	if dialectName == "postgres" {
		bindType = 2 // sqlx.DOLLAR
	}

	reboundQuery := sqlx.Rebind(bindType, parsedQuery)
	return db.ExecContext(ctx, reboundQuery, args...)
}

func scanInto(rows *sql.Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}
	base := reflect.Indirect(v)

	isStruct := base.Kind() == reflect.Struct
	if isStruct {
		if _, ok := base.Interface().(time.Time); ok {
			isStruct = false
		} else if _, ok := dest.(sql.Scanner); ok {
			isStruct = false
		}
	}

	if isStruct {
		rowsX := &sqlx.Rows{Rows: rows, Mapper: sqlx.NewDb(nil, "default").Mapper}
		return rowsX.StructScan(dest)
	}
	return rows.Scan(dest)
}
