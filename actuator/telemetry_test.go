package actuator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

type fakeSpan struct {
	mu         sync.Mutex
	ended      bool
	attributes map[string]any
	errors     []error
	status     SpanStatus
	statusDesc string
}

func newFakeSpan() *fakeSpan {
	return &fakeSpan{attributes: make(map[string]any)}
}

func (s *fakeSpan) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ended = true
}

func (s *fakeSpan) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attributes[key] = value
}

func (s *fakeSpan) RecordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors = append(s.errors, err)
}

func (s *fakeSpan) SetStatus(status SpanStatus, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.statusDesc = description
}

func (s *fakeSpan) SpanID() string  { return "fake-span-id" }
func (s *fakeSpan) TraceID() string { return "fake-trace-id" }

type fakeTracer struct {
	mu    sync.Mutex
	spans []*fakeSpan
}

func newFakeTracer() *fakeTracer {
	return &fakeTracer{}
}

func (t *fakeTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	t.mu.Lock()
	defer t.mu.Unlock()
	span := newFakeSpan()
	t.spans = append(t.spans, span)
	return ContextWithSpan(ctx, span), span
}

func (t *fakeTracer) SpanCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.spans)
}

func (t *fakeTracer) LastSpan() *fakeSpan {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.spans) == 0 {
		return nil
	}
	return t.spans[len(t.spans)-1]
}

func TestNoOpSpanMethods(t *testing.T) {
	span := &noOpSpan{}

	span.End()
	span.SetAttribute("key", "value")
	span.RecordError(nil)
	span.SetStatus(SpanStatusOK, "ok")

	if span.SpanID() != "" {
		t.Errorf("SpanID = %q, want empty", span.SpanID())
	}
	if span.TraceID() != "" {
		t.Errorf("TraceID = %q, want empty", span.TraceID())
	}
}

func TestNoOpTracerStart(t *testing.T) {
	tracer := &noOpTracer{}
	ctx := context.Background()

	returnedCtx, span := tracer.Start(ctx, "test-operation")

	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if returnedCtx == nil {
		t.Fatal("expected non-nil context")
	}
	if _, ok := span.(*noOpSpan); !ok {
		t.Errorf("expected *noOpSpan, got %T", span)
	}
}

func TestNewTelemetryDefaults(t *testing.T) {
	mod := NewTelemetry()

	if mod.Name() != "actuator.telemetry" {
		t.Errorf("Name = %q, want %q", mod.Name(), "actuator.telemetry")
	}
	if mod.Tracer() == nil {
		t.Fatal("expected non-nil tracer")
	}
	if !mod.enabled {
		t.Error("expected enabled by default")
	}
	if !mod.useMiddleware {
		t.Error("expected middleware enabled by default")
	}
}

func TestNewTelemetry_WithTracer(t *testing.T) {
	custom := newFakeTracer()
	mod := NewTelemetry(WithTracer(custom))

	if mod.Tracer() != custom {
		t.Error("expected custom tracer to be used")
	}
}

func TestNewTelemetry_WithNilTracer(t *testing.T) {
	mod := NewTelemetry(WithTracer(nil))

	if mod.Tracer() == nil {
		t.Error("expected default no-op tracer when nil passed")
	}
}

func TestNewTelemetry_Disabled(t *testing.T) {
	mod := NewTelemetry(WithTelemetryEnabled(false))

	if mod.enabled {
		t.Error("expected disabled")
	}
}

func TestNewTelemetry_MiddlewareDisabled(t *testing.T) {
	mod := NewTelemetry(WithTracingMiddleware(false))

	if mod.useMiddleware {
		t.Error("expected middleware disabled")
	}
}

func TestTelemetryModule_Configure_RegistersService(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewTelemetry())

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	if !app.Has(TelemetryServiceName) {
		t.Error("expected telemetry service to be registered")
	}
}

func TestTelemetryModule_Configure_Disabled(t *testing.T) {
	server := web.New(":0")
	app := core.New("test", server, NewTelemetry(WithTelemetryEnabled(false)))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	if app.Has(TelemetryServiceName) {
		t.Error("expected telemetry service NOT registered when disabled")
	}
}

func TestTelemetryModule_InstallsMiddleware(t *testing.T) {
	tracer := newFakeTracer()
	server := web.New(":0")
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, NewTelemetry(
		WithTracer(tracer),
		WithTracingMiddleware(true),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if tracer.SpanCount() != 1 {
		t.Fatalf("span count = %d, want 1", tracer.SpanCount())
	}

	span := tracer.LastSpan()
	if !span.ended {
		t.Error("expected span to be ended")
	}
	if span.attributes["http.method"] != "GET" {
		t.Errorf("http.method = %v, want GET", span.attributes["http.method"])
	}
	if span.attributes["http.url"] != "/test" {
		t.Errorf("http.url = %v, want /test", span.attributes["http.url"])
	}
}

func TestTelemetryModule_MiddlewareDisabled(t *testing.T) {
	tracer := newFakeTracer()
	server := web.New(":0")
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, NewTelemetry(
		WithTracer(tracer),
		WithTracingMiddleware(false),
	))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if tracer.SpanCount() != 0 {
		t.Errorf("span count = %d, want 0 (middleware disabled)", tracer.SpanCount())
	}
}

func TestTracerFromApp(t *testing.T) {
	custom := newFakeTracer()
	app := core.New("test", web.New(":0"), NewTelemetry(WithTracer(custom)))

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	tracer, ok := TracerFromApp(app)
	if !ok {
		t.Fatal("expected tracer to be found")
	}
	if tracer != custom {
		t.Error("expected same tracer instance")
	}
}

func TestTracerFromApp_NotRegistered(t *testing.T) {
	app := core.New("test", web.New(":0"))

	_, ok := TracerFromApp(app)
	if ok {
		t.Error("expected tracer not found")
	}
}

func TestSpanContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	span := newFakeSpan()

	enriched := ContextWithSpan(ctx, span)
	retrieved := SpanFromContext(enriched)

	if retrieved != span {
		t.Error("expected same span from context")
	}
}

func TestSpanFromContext_Missing(t *testing.T) {
	ctx := context.Background()
	span := SpanFromContext(ctx)

	if span != nil {
		t.Errorf("expected nil span, got %v", span)
	}
}

func TestTracingMiddleware_NilTracer(t *testing.T) {
	server := web.New(":0")
	server.Use(TracingMiddleware(nil))
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestTracingMiddleware_SpanInContext(t *testing.T) {
	tracer := newFakeTracer()
	server := web.New(":0")

	var capturedSpan Span
	server.Use(TracingMiddleware(tracer))
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		capturedSpan = SpanFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if capturedSpan == nil {
		t.Fatal("expected span in request context")
	}
	if capturedSpan.SpanID() != "fake-span-id" {
		t.Errorf("SpanID = %q, want fake-span-id", capturedSpan.SpanID())
	}
}

func TestTracingMiddleware_MultipleRequests(t *testing.T) {
	tracer := newFakeTracer()
	server := web.New(":0")
	server.Use(TracingMiddleware(tracer))
	server.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	app := core.New("test", server)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	rec1 := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec1, httptest.NewRequest("GET", "/a", nil))

	rec2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec2, httptest.NewRequest("POST", "/b", nil))

	if tracer.SpanCount() != 2 {
		t.Fatalf("span count = %d, want 2", tracer.SpanCount())
	}

	span1 := tracer.spans[0]
	if span1.attributes["http.method"] != "GET" {
		t.Errorf("span1 method = %v, want GET", span1.attributes["http.method"])
	}

	span2 := tracer.spans[1]
	if span2.attributes["http.method"] != "POST" {
		t.Errorf("span2 method = %v, want POST", span2.attributes["http.method"])
	}
}

func TestTelemetryWithNoOpTracer(t *testing.T) {
	server := web.New(":0")
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		span := SpanFromContext(r.Context())
		if span == nil {
			t.Error("expected span in context even with no-op tracer")
		}
		w.WriteHeader(http.StatusOK)
	})

	app := core.New("test", server, NewTelemetry())
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
