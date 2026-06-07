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
	app := core.New("test", server, NewHealth(Check{
		Name: "self",
		Check: func(context.Context) error {
			return nil
		},
	}))

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
	app := core.New("test", server, NewHealth(Check{
		Name: "database",
		Check: func(context.Context) error {
			return errors.New("connection refused")
		},
	}))

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
	app := core.New("test", server, NewHealthWithOptions(WithPath("/healthz")))

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
