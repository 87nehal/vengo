package actuator

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func TestNewLogging_Defaults(t *testing.T) {
	mod := NewLogging()
	if mod.Logger() == nil {
		t.Fatal("expected non-nil logger")
	}
	if mod.Name() != "actuator.logging" {
		t.Errorf("expected name 'actuator.logging', got %q", mod.Name())
	}
}

func TestNewLogging_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	mod := NewLogging(WithLogger(logger))
	if mod.Logger() != logger {
		t.Error("expected custom logger to be used")
	}
}

func TestNewLogging_WithLogLevel(t *testing.T) {
	mod := NewLogging(WithLogLevel(slog.LevelDebug))
	if mod.level != slog.LevelDebug {
		t.Errorf("expected level %v, got %v", slog.LevelDebug, mod.level)
	}
}

func TestNewLogging_Disabled(t *testing.T) {
	mod := NewLogging(WithLoggingEnabled(false))
	if mod.enabled {
		t.Error("expected module to be disabled")
	}
	if mod.logger != nil {
		t.Error("expected no logger when disabled")
	}
}

func TestNewLogging_DisableMiddleware(t *testing.T) {
	mod := NewLogging(WithRequestLogging(false))
	if mod.useMiddleware {
		t.Error("expected middleware to be disabled")
	}
}

func TestLoggingModule_Configure_RegistersService(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewLogging())

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	if !app.Has(LoggingServiceName) {
		t.Error("expected logging service to be registered")
	}
}

func TestLoggingModule_Configure_Disabled(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewLogging(WithLoggingEnabled(false)))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	if app.Has(LoggingServiceName) {
		t.Error("expected logging service NOT to be registered when disabled")
	}
}

func TestLoggingModule_InstallsMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server := web.New(":0")
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, NewLogging(WithLogger(logger)))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if buf.Len() == 0 {
		t.Fatal("expected middleware to log request")
	}

	var logData map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logData); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}

	if logData["method"] != "GET" {
		t.Errorf("expected method 'GET', got %v", logData["method"])
	}
	if logData["path"] != "/test" {
		t.Errorf("expected path '/test', got %v", logData["path"])
	}
}

func TestLoggingModule_MiddlewareDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server := web.New(":0")
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, NewLogging(
		WithLogger(logger),
		WithRequestLogging(false),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if buf.Len() > 0 {
		t.Error("expected no logging when middleware is disabled")
	}
}

func TestLoggerFromApp(t *testing.T) {
	var buf bytes.Buffer
	customLogger := slog.New(slog.NewJSONHandler(&buf, nil))

	app := core.New("test", web.New(":0"), NewLogging(WithLogger(customLogger)))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	logger, ok := LoggerFromApp(app)
	if !ok {
		t.Fatal("expected logger to be found")
	}
	if logger != customLogger {
		t.Error("expected same logger instance")
	}
}

func TestLoggerFromApp_NotRegistered(t *testing.T) {
	app := core.New("test", web.New(":0"))

	_, ok := LoggerFromApp(app)
	if ok {
		t.Error("expected logger not to be found")
	}
}
