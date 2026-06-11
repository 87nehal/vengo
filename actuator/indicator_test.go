package actuator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

type fakeIndicator struct {
	name   string
	health Health
}

func (f fakeIndicator) HealthIndicatorName() string   { return f.name }
func (f fakeIndicator) Health(context.Context) Health { return f.health }

func TestHealthWithIndicators(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(
		WithIndicators(
			fakeIndicator{name: "db", health: Health{Status: StatusUp, Details: map[string]any{"pool": 5}}},
			fakeIndicator{name: "cache", health: Health{Status: StatusDown, Details: map[string]any{"error": "timeout"}}},
		),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	req := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body["status"] != StatusDown {
		t.Fatalf("top-level status = %v, want DOWN", body["status"])
	}

	checks, ok := body["checks"].(map[string]any)
	if !ok {
		t.Fatalf("checks field missing or not a map")
	}

	dbEntry, ok := checks["db"].(map[string]any)
	if !ok {
		t.Fatalf("db check missing or not a map")
	}
	if dbEntry["status"] != StatusUp {
		t.Errorf("db status = %v, want UP", dbEntry["status"])
	}

	cacheEntry, ok := checks["cache"].(map[string]any)
	if !ok {
		t.Fatalf("cache check missing or not a map")
	}
	if cacheEntry["status"] != StatusDown {
		t.Errorf("cache status = %v, want DOWN", cacheEntry["status"])
	}
	if cacheEntry["error"] != "timeout" {
		t.Errorf("cache error detail = %v, want 'timeout'", cacheEntry["error"])
	}
}

func TestHealthNoIndicators(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth())

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	req := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body["status"] != StatusUp {
		t.Errorf("status = %v, want UP", body["status"])
	}
	if _, ok := body["checks"]; ok {
		t.Errorf("empty checks should be omitted from response")
	}
}

func TestHealthMixedIndicatorsAndChecks(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewHealth(
		WithChecks(Check{
			Name:  "legacy",
			Check: func(context.Context) error { return nil },
		}),
		WithIndicators(
			fakeIndicator{name: "modern", health: Health{Status: StatusDown}},
		),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })

	req := httptest.NewRequest(http.MethodGet, "/actuator/health", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"DOWN"`) {
		t.Errorf("body = %s, want DOWN", body)
	}
}

func TestAggregateWithUnknownStatus(t *testing.T) {
	indicators := []HealthIndicator{
		fakeIndicator{name: "a", health: Health{Status: StatusUp}},
		fakeIndicator{name: "b", health: Health{Status: StatusUnknown}},
	}
	result := aggregate(context.Background(), indicators)
	if result.Status != StatusUnknown {
		t.Errorf("aggregate status = %s, want UNKNOWN", result.Status)
	}
}

func TestAggregateWithDownOverridesUnknown(t *testing.T) {
	indicators := []HealthIndicator{
		fakeIndicator{name: "a", health: Health{Status: StatusUnknown}},
		fakeIndicator{name: "b", health: Health{Status: StatusDown}},
	}
	result := aggregate(context.Background(), indicators)
	if result.Status != StatusDown {
		t.Errorf("aggregate status = %s, want DOWN", result.Status)
	}
}
