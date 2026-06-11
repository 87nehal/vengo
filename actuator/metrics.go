package actuator

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const MetricsServiceName = "actuator.metrics"

type Counter struct {
	name string
	val  atomic.Int64
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Inc()         { c.val.Add(1) }
func (c *Counter) Add(n int64)  { c.val.Add(n) }
func (c *Counter) Value() int64 { return c.val.Load() }
func (c *Counter) Name() string { return c.name }

type Gauge struct {
	name string
	val  atomic.Int64
}

func NewGauge(name string) *Gauge {
	return &Gauge{name: name}
}

func (g *Gauge) Set(n int64)  { g.val.Store(n) }
func (g *Gauge) Inc()         { g.val.Add(1) }
func (g *Gauge) Dec()         { g.val.Add(-1) }
func (g *Gauge) Add(n int64)  { g.val.Add(n) }
func (g *Gauge) Value() int64 { return g.val.Load() }
func (g *Gauge) Name() string { return g.name }

type MetricsModule struct {
	path     string
	mu       sync.Mutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
	enabled  bool
}

func NewMetrics(options ...MetricsOption) *MetricsModule {
	module := &MetricsModule{
		path:     "/actuator/metrics",
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
		enabled:  true,
	}
	for _, opt := range options {
		if opt != nil {
			opt(module)
		}
	}
	return module
}

type MetricsOption func(*MetricsModule)

func WithMetricsPath(path string) MetricsOption {
	return func(m *MetricsModule) {
		if path != "" {
			m.path = path
		}
	}
}

func WithMetricsEnabled(enabled bool) MetricsOption {
	return func(m *MetricsModule) {
		m.enabled = enabled
	}
}

func (m *MetricsModule) Name() string {
	return "actuator.metrics"
}

func (m *MetricsModule) RegisterCounter(c *Counter) {
	if c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.name != "" {
		m.counters[c.name] = c
	}
}

func (m *MetricsModule) RegisterGauge(g *Gauge) {
	if g == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if g.name != "" {
		m.gauges[g.name] = g
	}
}

func (m *MetricsModule) Snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.counters)+len(m.gauges))
	for name, c := range m.counters {
		out[name] = c.Value()
	}
	for name, g := range m.gauges {
		out[name] = g.Value()
	}
	return out
}

func (m *MetricsModule) Configure(app *core.App) error {
	if !m.enabled {
		return nil
	}
	server, err := core.Get[*web.Server](app, web.ServiceName)
	if err != nil {
		return err
	}
	if err := app.Register(MetricsServiceName, m); err != nil {
		return err
	}
	server.HandleFunc(m.path, m.handle)
	return nil
}

func (m *MetricsModule) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	web.WriteJSON(w, http.StatusOK, m.Snapshot())
}
