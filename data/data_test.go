package data

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/87nehal/vengo/actuator"
	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
)

var (
	testDriverMu sync.Mutex
	testDriver   = &memoryDriver{dbs: map[string]*memoryState{}}
)

func init() { sql.Register("vengo_memory", testDriver) }

type memoryDriver struct{ dbs map[string]*memoryState }
type memoryState struct {
	mu         sync.Mutex
	pingErr    error
	closed     bool
	widgets    []string
	migrations map[string]bool
	queries    []string
}
type memoryConn struct{ state *memoryState }
type memoryTx struct{ state *memoryState }
type memoryRows struct {
	cols []string
	vals [][]driver.Value
	idx  int
}
type memoryResult int64

func memoryDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	testDriverMu.Lock()
	testDriver.dbs[name] = &memoryState{migrations: map[string]bool{}}
	testDriverMu.Unlock()
	db, err := sql.Open("vengo_memory", name)
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
func memoryStateFor(name string) *memoryState {
	testDriverMu.Lock()
	defer testDriverMu.Unlock()
	return testDriver.dbs[name]
}
func (d *memoryDriver) Open(name string) (driver.Conn, error) {
	testDriverMu.Lock()
	s := d.dbs[name]
	if s == nil {
		s = &memoryState{migrations: map[string]bool{}}
		d.dbs[name] = s
	}
	testDriverMu.Unlock()
	return &memoryConn{state: s}, nil
}
func (c *memoryConn) Prepare(q string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (c *memoryConn) Close() error {
	c.state.mu.Lock()
	c.state.closed = true
	c.state.mu.Unlock()
	return nil
}
func (c *memoryConn) Begin() (driver.Tx, error) { return &memoryTx{state: c.state}, nil }
func (c *memoryConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &memoryTx{state: c.state}, nil
}
func (c *memoryConn) Ping(context.Context) error {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	return c.state.pingErr
}
func (c *memoryConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	return execMemory(c.state, q, args)
}
func (c *memoryConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	return queryMemory(c.state, q, args)
}
func (t *memoryTx) Commit() error       { return nil }
func (t *memoryTx) Rollback() error     { return nil }
func (r *memoryRows) Columns() []string { return r.cols }
func (r *memoryRows) Close() error      { return nil }
func (r *memoryRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.vals) {
		return io.EOF
	}
	copy(dest, r.vals[r.idx])
	r.idx++
	return nil
}
func (r memoryResult) LastInsertId() (int64, error) { return int64(r), nil }
func (r memoryResult) RowsAffected() (int64, error) { return int64(r), nil }

func execMemory(s *memoryState, q string, args []driver.NamedValue) (driver.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, q)
	uq := strings.ToUpper(strings.TrimSpace(q))
	switch {
	case strings.HasPrefix(uq, "CREATE TABLE") || uq == "-- NOOP" || uq == "SELECT 1":
		return memoryResult(1), nil
	case strings.HasPrefix(uq, "INSERT INTO SCHEMA_MIGRATIONS") || strings.HasPrefix(uq, "INSERT INTO MIGRATIONS_TEST"):
		if len(args) > 0 {
			s.migrations[args[0].Value.(string)] = true
		}
		return memoryResult(1), nil
	case strings.HasPrefix(uq, "INSERT INTO WIDGETS"):
		if len(args) > 0 {
			s.widgets = append(s.widgets, args[0].Value.(string))
		} else {
			s.widgets = append(s.widgets, "literal")
		}
		return memoryResult(1), nil
	case strings.Contains(uq, "FAIL"):
		return nil, errors.New("forced failure")
	default:
		return memoryResult(1), nil
	}
}
func queryMemory(s *memoryState, q string, args []driver.NamedValue) (driver.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queries = append(s.queries, q)
	uq := strings.ToUpper(strings.TrimSpace(q))
	switch {
	case strings.HasPrefix(uq, "SELECT VERSION FROM"):
		if len(args) > 0 && s.migrations[args[0].Value.(string)] {
			return &memoryRows{cols: []string{"version"}, vals: [][]driver.Value{{args[0].Value}}}, nil
		}
		return &memoryRows{cols: []string{"version"}}, nil
	case strings.HasPrefix(uq, "SELECT COUNT(*) FROM WIDGETS"):
		return &memoryRows{cols: []string{"count"}, vals: [][]driver.Value{{int64(len(s.widgets))}}}, nil
	case strings.HasPrefix(uq, "SELECT 1"):
		return &memoryRows{cols: []string{"one"}, vals: [][]driver.Value{{int64(1)}}}, nil
	default:
		return &memoryRows{cols: []string{"ok"}, vals: [][]driver.Value{{"ok"}}}, nil
	}
}

func TestConfigBindingDefaults(t *testing.T) {
	cfg, err := config.Load(context.Background(), config.NewMapSource("test", map[string]string{"database.driver": "vengo_memory", "database.dsn": "cfg", "database.max-open-conns": "7", "database.slow-query-threshold": "1ms"}))
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := config.Bind(cfg, &c); err != nil {
		t.Fatal(err)
	}
	if c.Driver != "vengo_memory" || c.DSN != "cfg" || c.MaxOpenConns != 7 || c.MaxIdleConns != 2 || c.SlowQueryThreshold != time.Millisecond || c.MigrationsTable != "schema_migrations" {
		t.Fatalf("unexpected config: %+v", c)
	}
}
func TestModuleRegistrationAndPoolSettings(t *testing.T) {
	db := memoryDB(t, "module")
	app := core.New("test", New(WithDB(db), WithConfig(Config{MaxOpenConns: 3, MaxIdleConns: 1, SlowQueryThreshold: time.Nanosecond})))
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := core.Get[*sql.DB](app, DBServiceName)
	if err != nil || got != db {
		t.Fatalf("db service: %v", err)
	}
	inst, err := core.Get[*InstrumentedDB](app, InstrumentedDBServiceName)
	if err != nil || inst.SQLDB() != db {
		t.Fatalf("instrumented service: %v", err)
	}
	st := db.Stats()
	if st.MaxOpenConnections != 3 {
		t.Fatalf("max open=%d", st.MaxOpenConnections)
	}
}
func TestWithTxCommit(t *testing.T) {
	db := memoryDB(t, "txcommit")
	mgr := NewTxManager(db)
	if err := mgr.WithTx(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatal("missing tx in context")
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO widgets (name) VALUES (?)", "a")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if got := len(memoryStateFor("txcommit").widgets); got != 1 {
		t.Fatalf("widgets=%d", got)
	}
}
func TestWithTxRollback(t *testing.T) {
	db := memoryDB(t, "txrollback")
	err := NewTxManager(db).WithTx(context.Background(), func(context.Context, *sql.Tx) error { return errors.New("boom") })
	if err == nil {
		t.Fatal("want error")
	}
}
func TestGenericWithTx(t *testing.T) {
	db := memoryDB(t, "gentx")
	res, err := WithTx(context.Background(), db, func(ctx context.Context, tx *sql.Tx) (string, error) {
		if _, ok := TxFromContext(ctx); !ok {
			t.Fatal("missing tx in context")
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO widgets (name) VALUES (?)", "a")
		return "hello-result", err
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != "hello-result" {
		t.Fatalf("unexpected result: %q", res)
	}
	if got := len(memoryStateFor("gentx").widgets); got != 1 {
		t.Fatalf("widgets=%d", got)
	}
}
func TestApplyMigrations(t *testing.T) {
	db := memoryDB(t, "migrate")
	fsys := fstest.MapFS{"migrations/002_second.sql": {Data: []byte("INSERT INTO widgets (name) VALUES ('b')")}, "migrations/001_first.sql": {Data: []byte("INSERT INTO widgets (name) VALUES ('a')")}, "migrations/readme.txt": {Data: []byte("skip")}}
	if err := ApplyMigrations(context.Background(), db, fsys, MigrationOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(context.Background(), db, fsys, MigrationOptions{}); err != nil {
		t.Fatal(err)
	}
	s := memoryStateFor("migrate")
	if got := len(s.widgets); got != 2 {
		t.Fatalf("widgets=%d", got)
	}
	if !s.migrations["001_first.sql"] || !s.migrations["002_second.sql"] {
		t.Fatalf("migrations not recorded: %#v", s.migrations)
	}
}
func TestHealthIndicator(t *testing.T) {
	down := NewHealthIndicator(nil).Health(context.Background())
	if down.Status != actuator.StatusDown {
		t.Fatalf("nil status=%s", down.Status)
	}
	db := memoryDB(t, "health")
	up := NewHealthIndicator(db).Health(context.Background())
	if up.Status != actuator.StatusUp {
		t.Fatalf("up status=%s", up.Status)
	}
	memoryStateFor("health").pingErr = errors.New("down")
	down = NewHealthIndicator(db).Health(context.Background())
	if down.Status != actuator.StatusDown {
		t.Fatalf("down status=%s", down.Status)
	}
}
func TestSlowQueryLogging(t *testing.T) {
	db := memoryDB(t, "slow")
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	inst := NewInstrumentedDB(db, logger, -time.Nanosecond)
	if _, err := inst.ExecContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "slow database query") || !strings.Contains(buf.String(), "operation=exec") {
		t.Fatalf("missing slow log: %s", buf.String())
	}
}
