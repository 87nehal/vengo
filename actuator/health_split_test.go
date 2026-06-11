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

type livenessIndicator struct {
	name string
	up   bool
}

func (l livenessIndicator) HealthIndicatorName() string { return l.name }
func (l livenessIndicator) ProbeType() ProbeType        { return ProbeLiveness }
func (l livenessIndicator) Health(ctx context.Context) Health {
	if l.up {
		return Health{Status: StatusUp}
	}
	return Health{Status: StatusDown, Details: map[string]any{"error": "not alive"}}
}

type readinessIndicator struct {
	name string
	up   bool
}

func (r readinessIndicator) HealthIndicatorName() string { return r.name }
func (r readinessIndicator) ProbeType() ProbeType        { return ProbeReadiness }
func (r readinessIndicator) Health(ctx context.Context) Health {
	if r.up {
		return Health{Status: StatusUp}
	}
	return Health{Status: StatusDown, Details: map[string]any{"error": "not ready"}}
}

type bothIndicator struct {
	name string
	up   bool
}

func (b bothIndicator) HealthIndicatorName() string { return b.name }
func (b bothIndicator) ProbeType() ProbeType        { return ProbeBoth }
func (b bothIndicator) Health(ctx context.Context) Health {
	if b.up {
		return Health{Status: StatusUp}
	}
	return Health{Status: StatusDown, Details: map[string]any{"error": "unhealthy"}}
}

func splitApp(t *testing.T, indicators ...HealthIndicator) (*web.Server, *core.App) {
	t.Helper()
	server := web.New(":0")
	health := NewHealth(WithIndicators(indicators...))
	app := core.New("test", server, health)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	return server, app
}

func parseBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return result
}

func TestLivenessEndpointOnlyShowsLivenessIndicators(t *testing.T) {
	server, _ := splitApp(t,
		livenessIndicator{name: "app", up: true},
		readinessIndicator{name: "db", up: true},
		bothIndicator{name: "cache", up: true},
	)

	req := httptest.NewRequest("GET", "/actuator/health/liveness", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := parseBody(t, rec)
	checks := body["checks"].(map[string]any)
	if _, ok := checks["app"]; !ok {
		t.Error("liveness indicator 'app' missing")
	}
	if _, ok := checks["cache"]; !ok {
		t.Error("both indicator 'cache' missing in liveness")
	}
	if _, ok := checks["db"]; ok {
		t.Error("readiness indicator 'db' should not appear in liveness")
	}
}

func TestReadinessEndpointOnlyShowsReadinessIndicators(t *testing.T) {
	server, _ := splitApp(t,
		livenessIndicator{name: "app", up: true},
		readinessIndicator{name: "db", up: true},
		bothIndicator{name: "cache", up: true},
	)

	req := httptest.NewRequest("GET", "/actuator/health/readiness", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := parseBody(t, rec)
	checks := body["checks"].(map[string]any)
	if _, ok := checks["db"]; !ok {
		t.Error("readiness indicator 'db' missing")
	}
	if _, ok := checks["cache"]; !ok {
		t.Error("both indicator 'cache' missing in readiness")
	}
	if _, ok := checks["app"]; ok {
		t.Error("liveness indicator 'app' should not appear in readiness")
	}
}

func TestLivenessDown(t *testing.T) {
	server, _ := splitApp(t, livenessIndicator{name: "app", up: false})
	req := httptest.NewRequest("GET", "/actuator/health/liveness", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := parseBody(t, rec)
	if body["status"].(string) != "DOWN" {
		t.Errorf("status = %v, want DOWN", body["status"])
	}
}

func TestReadinessDown(t *testing.T) {
	server, _ := splitApp(t, readinessIndicator{name: "db", up: false})
	req := httptest.NewRequest("GET", "/actuator/health/readiness", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestReadinessDownDoesNotAffectLiveness(t *testing.T) {
	server, _ := splitApp(t,
		livenessIndicator{name: "app", up: true},
		readinessIndicator{name: "db", up: false},
	)

	livenessRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(livenessRec,
		httptest.NewRequest("GET", "/actuator/health/liveness", nil))
	if livenessRec.Code != http.StatusOK {
		t.Errorf("liveness should be UP, got %d", livenessRec.Code)
	}

	readinessRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(readinessRec,
		httptest.NewRequest("GET", "/actuator/health/readiness", nil))
	if readinessRec.Code != http.StatusServiceUnavailable {
		t.Errorf("readiness should be DOWN, got %d", readinessRec.Code)
	}
}

func TestBaseHealthStillWorks(t *testing.T) {
	server, _ := splitApp(t,
		livenessIndicator{name: "app", up: true},
		readinessIndicator{name: "db", up: false},
	)

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/actuator/health", nil))

	body := parseBody(t, rec)
	if body["status"] != "DOWN" {
		t.Errorf("base /health status = %v, want DOWN", body["status"])
	}
	checks := body["checks"].(map[string]any)
	if len(checks) != 2 {
		t.Errorf("base /health checks count = %d, want 2", len(checks))
	}
}

func TestSplitEndpointsRejectNonGet(t *testing.T) {
	server, _ := splitApp(t, livenessIndicator{name: "app", up: true})

	for _, path := range []string{"/actuator/health/liveness", "/actuator/health/readiness"} {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, httptest.NewRequest("POST", path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s POST: status = %d, want 405", path, rec.Code)
		}
	}
}
