package actuator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func TestInfoEndpointWithVersion(t *testing.T) {
	server := web.New(":0")
	info := NewInfo(
		WithVersion("1.2.3"),
		WithCommit("abc123"),
		WithBuild("2024-01-15"),
	)
	app := core.New("testapp", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result["name"] != "testapp" {
		t.Errorf("name = %v, want testapp", result["name"])
	}
	if result["version"] != "1.2.3" {
		t.Errorf("version = %v, want 1.2.3", result["version"])
	}
	if result["commit"] != "abc123" {
		t.Errorf("commit = %v, want abc123", result["commit"])
	}
	if result["build"] != "2024-01-15" {
		t.Errorf("build = %v, want 2024-01-15", result["build"])
	}
}

func TestInfoEndpointWithExtra(t *testing.T) {
	server := web.New(":0")
	info := NewInfo(
		WithVersion("1.0.0"),
		WithInfoExtra("environment", "production"),
		WithInfoExtra("region", "us-east-1"),
	)
	app := core.New("myapp", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result["environment"] != "production" {
		t.Errorf("environment = %v, want production", result["environment"])
	}
	if result["region"] != "us-east-1" {
		t.Errorf("region = %v, want us-east-1", result["region"])
	}
}

func TestInfoEndpointMinimalInfo(t *testing.T) {
	server := web.New(":0")
	info := NewInfo()
	app := core.New("simple", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result["name"] != "simple" {
		t.Errorf("name = %v, want simple", result["name"])
	}

	if len(result) != 1 {
		t.Errorf("result has %d fields, want 1", len(result))
	}
}

func TestInfoEndpointCustomPath(t *testing.T) {
	server := web.New(":0")
	info := NewInfo(
		WithInfoPath("/custom/info"),
		WithVersion("1.0.0"),
	)
	app := core.New("test", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/custom/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", result["version"])
	}
}

func TestInfoEndpointRejectsNonGet(t *testing.T) {
	server := web.New(":0")
	info := NewInfo()
	app := core.New("test", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("POST", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestInfoEndpointEmptyOptions(t *testing.T) {
	server := web.New(":0")
	info := NewInfo(
		WithInfoPath(""),
		WithInfoEnabled(true),
	)
	app := core.New("test", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestInfoEndpointDisabled(t *testing.T) {
	server := web.New(":0")
	info := NewInfo(WithInfoEnabled(false))
	app := core.New("test", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when disabled", rec.Code)
	}
}

func TestInfoEndpointEnabledByDefault(t *testing.T) {
	server := web.New(":0")
	info := NewInfo()
	app := core.New("test", server, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/info", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (enabled by default)", rec.Code)
	}
}

func TestInfoEndpointWithHealthModule(t *testing.T) {
	server := web.New(":0")
	health := NewHealth()
	info := NewInfo(WithVersion("2.0.0"))
	app := core.New("dual", server, health, info)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	healthReq := httptest.NewRequest("GET", "/actuator/health", nil)
	healthRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(healthRec, healthReq)

	if healthRec.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200", healthRec.Code)
	}

	infoReq := httptest.NewRequest("GET", "/actuator/info", nil)
	infoRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(infoRec, infoReq)

	if infoRec.Code != http.StatusOK {
		t.Errorf("info status = %d, want 200", infoRec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(infoRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result["version"] != "2.0.0" {
		t.Errorf("version = %v, want 2.0.0", result["version"])
	}
}
