package data

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type widgetRow struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func TestQueryOneAndMany(t *testing.T) {
	db, err := sql.Open("sqlite", "file:TestQueryOneAndMany?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Setup table
	_, err = db.ExecContext(ctx, `CREATE TABLE widgets (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert data
	_, err = db.ExecContext(ctx, `INSERT INTO widgets (name) VALUES (?), (?)`, "widget-a", "widget-b")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Test QueryOne for struct
	row, err := QueryOne[widgetRow](ctx, db, `SELECT id, name FROM widgets WHERE name = ?`, "widget-a")
	if err != nil {
		t.Fatalf("QueryOne struct: %v", err)
	}
	if row.Name != "widget-a" || row.ID != 1 {
		t.Errorf("unexpected struct result: %+v", row)
	}

	// Test QueryOne for primitive (int)
	id, err := QueryOne[int](ctx, db, `SELECT id FROM widgets WHERE name = ?`, "widget-b")
	if err != nil {
		t.Fatalf("QueryOne primitive: %v", err)
	}
	if id != 2 {
		t.Errorf("unexpected primitive result: %d", id)
	}

	// Test QueryOne no rows
	_, err = QueryOne[widgetRow](ctx, db, `SELECT id, name FROM widgets WHERE name = ?`, "non-existent")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	// Test QueryMany for struct
	rows, err := QueryMany[widgetRow](ctx, db, `SELECT id, name FROM widgets ORDER BY id ASC`)
	if err != nil {
		t.Fatalf("QueryMany: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Name != "widget-a" || rows[1].Name != "widget-b" {
		t.Errorf("unexpected rows: %+v", rows)
	}
}

func TestExecNamed(t *testing.T) {
	db, err := sql.Open("sqlite", "file:TestExecNamed?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `CREATE TABLE widgets (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert named
	w := widgetRow{Name: "named-widget"}
	_, err = ExecNamed(ctx, db, `INSERT INTO widgets (name) VALUES (:name)`, w)
	if err != nil {
		t.Fatalf("ExecNamed: %v", err)
	}

	// Check insert
	name, err := QueryOne[string](ctx, db, `SELECT name FROM widgets WHERE id = 1`)
	if err != nil {
		t.Fatalf("QueryOne: %v", err)
	}
	if name != "named-widget" {
		t.Errorf("expected 'named-widget', got %q", name)
	}
}

type scanStruct struct {
	TimeVal time.Time `db:"t_val"`
}

func TestScanIntoSpecialTypes(t *testing.T) {
	db, err := sql.Open("sqlite", "file:TestScanIntoSpecialTypes?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `CREATE TABLE test_table (t_val TIMESTAMP)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	_, err = db.ExecContext(ctx, `INSERT INTO test_table (t_val) VALUES (?)`, now)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Scan into struct with time.Time field
	row, err := QueryOne[scanStruct](ctx, db, `SELECT t_val FROM test_table`)
	if err != nil {
		t.Fatalf("QueryOne with struct containing time.Time: %v", err)
	}
	if !row.TimeVal.Equal(now) {
		t.Errorf("expected time %v, got %v", now, row.TimeVal)
	}

	// Scan directly into time.Time
	directTime, err := QueryOne[time.Time](ctx, db, `SELECT t_val FROM test_table`)
	if err != nil {
		t.Fatalf("QueryOne direct time.Time: %v", err)
	}
	if !directTime.Equal(now) {
		t.Errorf("expected time %v, got %v", now, directTime)
	}
}
