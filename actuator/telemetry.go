package actuator

import (
	"context"
	"net/http"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const TelemetryServiceName = "actuator.telemetry"

type SpanStatus int

const (
	SpanStatusOK SpanStatus = iota
	SpanStatusError
)

type Span interface {
	End()
	SetAttribute(key string, value any)
	RecordError(err error)
	SetStatus(status SpanStatus, description string)
	SpanID() string
	TraceID() string
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type noOpSpan struct{}

func (s *noOpSpan) End()                         {}
func (s *noOpSpan) SetAttribute(string, any)     {}
func (s *noOpSpan) RecordError(error)            {}
func (s *noOpSpan) SetStatus(SpanStatus, string) {}
func (s *noOpSpan) SpanID() string               { return "" }
func (s *noOpSpan) TraceID() string              { return "" }

type noOpTracer struct{}

func (t *noOpTracer) Start(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &noOpSpan{}
}

var defaultNoOpTracer Tracer = &noOpTracer{}

type TelemetryModule struct {
	tracer        Tracer
	enabled       bool
	useMiddleware bool
}

type TelemetryOption func(*TelemetryModule)

func NewTelemetry(opts ...TelemetryOption) *TelemetryModule {
	m := &TelemetryModule{
		tracer:        defaultNoOpTracer,
		enabled:       true,
		useMiddleware: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	return m
}

func WithTracer(tracer Tracer) TelemetryOption {
	return func(m *TelemetryModule) {
		if tracer != nil {
			m.tracer = tracer
		}
	}
}

func WithTelemetryEnabled(enabled bool) TelemetryOption {
	return func(m *TelemetryModule) {
		m.enabled = enabled
	}
}

func WithTracingMiddleware(enabled bool) TelemetryOption {
	return func(m *TelemetryModule) {
		m.useMiddleware = enabled
	}
}

func (m *TelemetryModule) Name() string {
	return "actuator.telemetry"
}

func (m *TelemetryModule) Tracer() Tracer {
	return m.tracer
}

func (m *TelemetryModule) Configure(app *core.App) error {
	if !m.enabled {
		return nil
	}

	if err := app.Register(TelemetryServiceName, m); err != nil {
		return err
	}

	if m.useMiddleware {
		server, err := core.Get[*web.Server](app, web.ServiceName)
		if err != nil {
			return nil
		}
		server.Use(TracingMiddleware(m.tracer))
	}

	return nil
}

func TracerFromApp(app *core.App) (Tracer, bool) {
	mod, err := core.Get[*TelemetryModule](app, TelemetryServiceName)
	if err != nil {
		return nil, false
	}
	return mod.Tracer(), true
}

type spanContextKey struct{}

func SpanFromContext(ctx context.Context) Span {
	s, _ := ctx.Value(spanContextKey{}).(Span)
	return s
}

func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, spanContextKey{}, span)
}

func TracingMiddleware(tracer Tracer) web.Middleware {
	if tracer == nil {
		tracer = defaultNoOpTracer
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()

			span.SetAttribute("http.method", r.Method)
			span.SetAttribute("http.url", r.URL.Path)
			span.SetAttribute("http.remote", r.RemoteAddr)

			next.ServeHTTP(w, r.WithContext(ContextWithSpan(ctx, span)))
		})
	}
}
