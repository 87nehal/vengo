package actuator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func TestHealthEndpoint(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(WithChecks(Check{
		Name: "self",
		Check: func(context.Context) error {
			return nil
		},
	})))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	request := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"status":"UP"`) {
		t.Fatalf("body did not contain UP status: %s", response.Body.String())
	}
}

func TestHealthEndpointReportsFailure(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(WithChecks(Check{
		Name: "database",
		Check: func(context.Context) error {
			return errors.New("connection refused")
		},
	})))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	request := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(response.Body.String(), `"status":"DOWN"`) {
		t.Fatalf("body did not contain DOWN status: %s", response.Body.String())
	}
}

func TestHealthEndpointSupportsCustomPath(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(WithPath("/healthz")))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHealthEndpointRejectsNonGet(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth())

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	request := httptest.NewRequest(http.MethodPost, "/actuator/health", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewHealthAcceptsPathAndChecks(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(
		WithPath("/healthz"),
		WithChecks(Check{
			Name:  "self",
			Check: func(context.Context) error { return nil },
		}),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"status":"UP"`) {
		t.Fatalf("body did not contain UP status: %s", response.Body.String())
	}
}

func TestHealthEndpointDisabled(t *testing.T) {
	server := web.New(":0")
	health := NewHealth(WithEnabled(false))
	app := core.New("test", server, health)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when disabled", rec.Code)
	}
}

func TestHealthEndpointEnabledByDefault(t *testing.T) {
	server := web.New(":0")
	health := NewHealth()
	app := core.New("test", server, health)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
