package actuator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func TestCounterIncAndAdd(t *testing.T) {
	c := NewCounter("http.requests")
	c.Inc()
	c.Inc()
	c.Add(3)

	if c.Value() != 5 {
		t.Fatalf("counter = %d, want 5", c.Value())
	}
	if c.Name() != "http.requests" {
		t.Errorf("Name = %q, want %q", c.Name(), "http.requests")
	}
}

func TestGaugeSetAndDelta(t *testing.T) {
	g := NewGauge("connections.open")
	g.Set(10)
	if g.Value() != 10 {
		t.Fatalf("gauge = %d, want 10", g.Value())
	}

	g.Inc()
	g.Add(2)
	g.Dec()

	if g.Value() != 12 {
		t.Fatalf("gauge = %d, want 12", g.Value())
	}
	if g.Name() != "connections.open" {
		t.Errorf("Name = %q, want %q", g.Name(), "connections.open")
	}
}

func TestCounterConcurrency(t *testing.T) {
	c := NewCounter("concurrent")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()

	if c.Value() != 100 {
		t.Fatalf("counter = %d, want 100", c.Value())
	}
}

func TestGaugeConcurrency(t *testing.T) {
	g := NewGauge("concurrent")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			g.Inc()
		}()
		go func() {
			defer wg.Done()
			g.Dec()
		}()
	}
	wg.Wait()

	if g.Value() != 0 {
		t.Fatalf("gauge = %d, want 0", g.Value())
	}
}

func TestMetricsModuleSnapshot(t *testing.T) {
	m := NewMetrics()
	reqs := NewCounter("requests")
	errs := NewCounter("errors")
	open := NewGauge("open")

	m.RegisterCounter(reqs)
	m.RegisterCounter(errs)
	m.RegisterGauge(open)

	reqs.Inc()
	reqs.Inc()
	errs.Inc()
	open.Set(7)

	snap := m.Snapshot()
	if snap["requests"] != 2 {
		t.Errorf("requests = %d, want 2", snap["requests"])
	}
	if snap["errors"] != 1 {
		t.Errorf("errors = %d, want 1", snap["errors"])
	}
	if snap["open"] != 7 {
		t.Errorf("open = %d, want 7", snap["open"])
	}
}

func TestMetricsModuleIgnoresNil(t *testing.T) {
	m := NewMetrics()
	m.RegisterCounter(nil)
	m.RegisterGauge(nil)

	snap := m.Snapshot()
	if len(snap) != 0 {
		t.Errorf("snapshot = %v, want empty", snap)
	}
}

func TestMetricsEndpointDisabled(t *testing.T) {
	server := web.New(":0")
	metrics := NewMetrics(WithMetricsEnabled(false))
	app := core.New("test", server, metrics)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when disabled", rec.Code)
	}
}

func TestMetricsEndpointEnabledByDefault(t *testing.T) {
	server := web.New(":0")
	metrics := NewMetrics()
	app := core.New("test", server, metrics)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/actuator/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	server := web.New(":0")
	metrics := NewMetrics()
	reqs := NewCounter("requests")
	open := NewGauge("open")
	metrics.RegisterCounter(reqs)
	metrics.RegisterGauge(open)

	app := core.New("test", server, metrics)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	reqs.Inc()
	reqs.Inc()
	open.Set(42)

	req := httptest.NewRequest("GET", "/actuator/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if body["requests"] != 2 {
		t.Errorf("requests = %d, want 2", body["requests"])
	}
	if body["open"] != 42 {
		t.Errorf("open = %d, want 42", body["open"])
	}
}

func TestMetricsEndpointRejectsNonGet(t *testing.T) {
	server := web.New(":0")
	metrics := NewMetrics()
	app := core.New("test", server, metrics)

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("POST", "/actuator/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestMetricsEndpointIntegration(t *testing.T) {
	server := web.New(":0")
	metrics := NewMetrics()
	reqs := NewCounter("requests")
	metrics.RegisterCounter(reqs)

	server.HandleFunc("/bump", func(w http.ResponseWriter, r *http.Request) {
		reqs.Inc()
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, metrics)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	for i := 0; i < 3; i++ {
		bumpRec := httptest.NewRecorder()
		server.Handler().ServeHTTP(bumpRec, httptest.NewRequest("GET", "/bump", nil))
		if bumpRec.Code != http.StatusOK {
			t.Fatalf("bump status = %d, want 200", bumpRec.Code)
		}
	}

	metricsRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRec, httptest.NewRequest("GET", "/actuator/metrics", nil))

	var body map[string]int64
	if err := json.Unmarshal(metricsRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["requests"] != 3 {
		t.Errorf("requests = %d, want 3", body["requests"])
	}
}
