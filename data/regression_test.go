package data_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/87nehal/vengo/cmd"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/data"
	"github.com/87nehal/vengo/testutil"
	"github.com/87nehal/vengo/web"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type TestUser struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func TestRegression_Issue1_MultiStatementMigrations(t *testing.T) {
	db := testutil.OpenTestDB(t, "sqlite", ":memory:")
	
	fsys := fstest.MapFS{
		"migrations/0001_init.sql": {Data: []byte(`
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE posts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				title TEXT NOT NULL
			);
		`)},
	}
	
	testutil.RunMigrations(t, db, fsys)
	
	_, err := db.Exec("INSERT INTO users (name) VALUES ('Alice')")
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	_, err = db.Exec("INSERT INTO posts (user_id, title) VALUES (1, 'Hello World')")
	if err != nil {
		t.Fatalf("failed to insert post: %v", err)
	}
}

func TestRegression_Issue2_DateTimeScanning_MySQL_MariaDB_Postgres_SQLite(t *testing.T) {
	databases := []struct {
		name   string
		driver string
		dsn    string
	}{
		{"sqlite", "sqlite", ":memory:"},
	}

	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		databases = append(databases, struct{ name, driver, dsn string }{"mysql", "mysql", dsn})
	} else {
		databases = append(databases, struct{ name, driver, dsn string }{"mysql", "mysql", "root:root@tcp(127.0.0.1:3306)/vengo_test"})
	}

	if dsn := os.Getenv("MARIADB_TEST_DSN"); dsn != "" {
		databases = append(databases, struct{ name, driver, dsn string }{"mariadb", "mysql", dsn})
	} else {
		databases = append(databases, struct{ name, driver, dsn string }{"mariadb", "mysql", "root:root@tcp(127.0.0.1:3307)/vengo_test"})
	}

	if dsn := os.Getenv("POSTGRES_TEST_DSN"); dsn != "" {
		databases = append(databases, struct{ name, driver, dsn string }{"postgres", "postgres", dsn})
	} else {
		databases = append(databases, struct{ name, driver, dsn string }{"postgres", "postgres", "postgres://root:root@127.0.0.1:5432/vengo_test?sslmode=disable"})
	}

	for _, dbInfo := range databases {
		t.Run(dbInfo.name, func(t *testing.T) {
			if dbInfo.driver == "mysql" {
				adminDB, err := sql.Open(dbInfo.driver, strings.Replace(dbInfo.dsn, "/vengo_test", "/", 1))
				if err == nil {
					_, _ = adminDB.Exec("CREATE DATABASE IF NOT EXISTS vengo_test")
					_ = adminDB.Close()
				}
			}

			dsn := dbInfo.dsn
			if dbInfo.name == "mysql" || dbInfo.name == "mariadb" {
				if !strings.Contains(dsn, "parseTime=") {
					if strings.Contains(dsn, "?") {
						dsn += "&parseTime=true"
					} else {
						dsn += "?parseTime=true"
					}
				}
			}

			db := testutil.OpenTestDB(t, dbInfo.driver, dsn)
			
			_, _ = db.Exec("DROP TABLE IF EXISTS test_users")
			
			dialect, err := data.DialectForDriver(dbInfo.driver)
			if err != nil {
				t.Fatalf("unsupported driver: %v", err)
			}
			
			var createSQL string
			switch dialect.Name() {
			case "postgres":
				createSQL = `CREATE TABLE test_users (
					id SERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`
			case "mysql", "mariadb":
				createSQL = `CREATE TABLE test_users (
					id INT AUTO_INCREMENT PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`
			default:
				createSQL = `CREATE TABLE test_users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				)`
			}
			
			if _, err := db.Exec(createSQL); err != nil {
				t.Fatalf("failed to create table: %v", err)
			}
			
			now := time.Now().UTC().Round(time.Second)
			var insertSQL string
			var insertArgs []any
			if dialect.Name() == "postgres" {
				insertSQL = `INSERT INTO test_users (name, created_at) VALUES ($1, $2)`
				insertArgs = []any{"Bob", now}
			} else {
				insertSQL = `INSERT INTO test_users (name, created_at) VALUES (?, ?)`
				insertArgs = []any{"Bob", now}
			}
			
			if _, err := db.Exec(insertSQL, insertArgs...); err != nil {
				t.Fatalf("failed to insert row: %v", err)
			}
			
			var scannedUser TestUser
			querySQL := `SELECT id, name, created_at FROM test_users LIMIT 1`
			err = db.QueryRow(querySQL).Scan(&scannedUser.ID, &scannedUser.Name, &scannedUser.CreatedAt)
			if err != nil {
				t.Fatalf("failed to scan into time.Time: %v", err)
			}
			
			if scannedUser.Name != "Bob" {
				t.Errorf("got name %q, want Bob", scannedUser.Name)
			}
			
			diff := scannedUser.CreatedAt.Sub(now)
			if diff < 0 {
				diff = -diff
			}
			if diff > 5*time.Second {
				t.Errorf("scanned time %v too far from original time %v", scannedUser.CreatedAt, now)
			}
		})
	}
}

func TestRegression_Issue3_BindJSON_LenientAndStrict(t *testing.T) {
	type TestRequest struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	
	payload := `{"name":"Charlie","age":30,"extra_field":"hello","another_extra":true}`
	
	t.Run("lenient by default", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/test", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		
		var target TestRequest
		err := web.BindJSON(req, &target)
		if err != nil {
			t.Fatalf("BindJSON failed on unknown fields: %v", err)
		}
		
		if target.Name != "Charlie" || target.Age != 30 {
			t.Errorf("expected parsed values, got %+v", target)
		}
	})
	
	t.Run("strict mode fails on unknown fields", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/test", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		
		var target TestRequest
		err := web.BindJSONStrict(req, &target)
		if err == nil {
			t.Fatalf("expected BindJSONStrict to fail on unknown fields, but it succeeded")
		}
		
		if err.Code != http.StatusBadRequest {
			t.Errorf("expected StatusBadRequest (400), got %d", err.Code)
		}
	})
}

func TestRegression_Issue5_SignalHelper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	shutdownCtx := cmd.NotifyShutdown(ctx)
	if shutdownCtx == nil {
		t.Fatal("expected non-nil shutdown context")
	}
}

func TestRegression_Issue6_WithIPv4Only(t *testing.T) {
	server := web.New("127.0.0.1:0", web.WithIPv4Only())
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Stop(ctx) }()
	
	addr := server.Addr()
	if addr == "" {
		t.Fatal("bound address is empty")
	}
}

func TestRegression_Issue7_OnReadyHook(t *testing.T) {
	server := web.New("127.0.0.1:0")
	
	readyCh := make(chan string, 1)
	server.OnReady(func(addr string) {
		readyCh <- addr
	})
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() { _ = server.Stop(ctx) }()
	
	addr := server.Addr()
	
	client := &http.Client{Timeout: 1 * time.Second}
	_, _ = client.Get("http://" + addr + "/")
	
	select {
	case boundAddr := <-readyCh:
		if boundAddr != addr {
			t.Errorf("ready address %s does not match bound address %s", boundAddr, addr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OnReady hook to be called")
	}
}

func TestRegression_Issue8_BackgroundServerErrors(t *testing.T) {
	server1 := web.New("127.0.0.1:0")
	ctx := context.Background()
	err := server1.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server 1: %v", err)
	}
	defer func() { _ = server1.Stop(ctx) }()
	
	addr := server1.Addr()
	
	server2 := web.New(addr)
	errCh := server2.ErrChan()
	
	err = server2.Start(ctx)
	if err == nil {
		select {
		case bgErr := <-errCh:
			if bgErr == nil {
				t.Fatal("expected background error, got nil")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("expected server2 to fail due to port already in use")
		}
	} else {
		t.Logf("server2 failed synchronously as expected: %v", err)
	}
}

func TestRegression_Issue9_MariaDBDialectAndBuilders(t *testing.T) {
	d, err := data.DialectForDriver("mariadb")
	if err != nil {
		t.Fatalf("failed to get mariadb dialect: %v", err)
	}
	if d.Name() != "mysql" {
		t.Errorf("got dialect name %q, want mysql", d.Name())
	}
	
	mysqlDSN := data.NewMySQLDSN("localhost", 3306, "mydb", "user", "pass")
	if !strings.Contains(mysqlDSN, "parseTime=true") || !strings.Contains(mysqlDSN, "multiStatements=true") {
		t.Errorf("mysql DSN missing defaults: %s", mysqlDSN)
	}
	
	mariaDSN := data.NewMariaDBDSN("localhost", 3307, "mydb", "user", "pass", "charset=utf8mb4")
	if !strings.Contains(mariaDSN, "parseTime=true") || !strings.Contains(mariaDSN, "charset=utf8mb4") {
		t.Errorf("mariadb DSN missing defaults or extra option: %s", mariaDSN)
	}
}

func TestRegression_Issue10_IntegrationTestUtilities(t *testing.T) {
	app := core.New("integration-test-app", web.New("127.0.0.1:0"))
	
	ts := testutil.NewTestServer(t, app)
	if ts == nil {
		t.Fatal("expected non-nil test server")
	}
	
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("failed to make request to test server: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unregistered path, got %d", resp.StatusCode)
	}
}
