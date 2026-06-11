package actuator

import (
	"context"
	"fmt"
	"net/http"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const HealthServiceName = "actuator.health"

const (
	StatusUp      = "UP"
	StatusDown    = "DOWN"
	StatusUnknown = "UNKNOWN"
)

type ProbeType int

const (
	ProbeLiveness ProbeType = iota
	ProbeReadiness
	ProbeBoth
)

type Health struct {
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

type HealthIndicator interface {
	HealthIndicatorName() string
	Health(ctx context.Context) Health
}

type ProbedIndicator interface {
	HealthIndicator
	ProbeType() ProbeType
}

type Check struct {
	Name  string
	Check func(context.Context) error
}

type checkIndicator struct {
	check Check
}

func (c checkIndicator) HealthIndicatorName() string {
	if c.check.Name == "" {
		return "unnamed"
	}
	return c.check.Name
}

func (c checkIndicator) Health(ctx context.Context) Health {
	if c.check.Check == nil {
		return Health{Status: StatusUp}
	}
	if err := c.check.Check(ctx); err != nil {
		return Health{Status: StatusDown, Details: map[string]any{"error": err.Error()}}
	}
	return Health{Status: StatusUp}
}

type HealthModule struct {
	path       string
	indicators []HealthIndicator
	enabled    bool
}

type Option func(*HealthModule)

func NewHealth(options ...Option) *HealthModule {
	module := &HealthModule{
		path:    "/actuator/health",
		enabled: true,
	}
	for _, option := range options {
		if option != nil {
			option(module)
		}
	}
	return module
}

func WithPath(path string) Option {
	return func(module *HealthModule) {
		if path == "" {
			return
		}
		module.path = path
	}
}

func WithChecks(checks ...Check) Option {
	return func(module *HealthModule) {
		for _, c := range checks {
			module.indicators = append(module.indicators, checkIndicator{check: c})
		}
	}
}

func WithIndicators(indicators ...HealthIndicator) Option {
	return func(module *HealthModule) {
		for _, ind := range indicators {
			if ind != nil {
				module.indicators = append(module.indicators, ind)
			}
		}
	}
}

func WithEnabled(enabled bool) Option {
	return func(module *HealthModule) {
		module.enabled = enabled
	}
}

func (m *HealthModule) Name() string {
	return "actuator.health"
}

func (m *HealthModule) Configure(app *core.App) error {
	if !m.enabled {
		return nil
	}
	server, err := core.Get[*web.Server](app, web.ServiceName)
	if err != nil {
		return fmt.Errorf("health module requires web module: %w", err)
	}
	if err := app.Register(HealthServiceName, m); err != nil {
		return err
	}
	server.HandleFunc(m.path, m.handle)
	server.HandleFunc(m.path+"/liveness", m.handleLiveness)
	server.HandleFunc(m.path+"/readiness", m.handleReadiness)
	return nil
}

func (m *HealthModule) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	aggregated := aggregate(r.Context(), m.indicators)
	code := http.StatusOK
	if aggregated.Status == StatusDown {
		code = http.StatusServiceUnavailable
	}

	response := map[string]any{"status": aggregated.Status}
	if len(aggregated.Details) > 0 {
		response["checks"] = aggregated.Details
	}
	web.WriteJSON(w, code, response)
}

func (m *HealthModule) handleLiveness(w http.ResponseWriter, r *http.Request) {
	m.handleProbe(w, r, ProbeLiveness)
}

func (m *HealthModule) handleReadiness(w http.ResponseWriter, r *http.Request) {
	m.handleProbe(w, r, ProbeReadiness)
}

func (m *HealthModule) handleProbe(w http.ResponseWriter, r *http.Request, probeType ProbeType) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	filtered := make([]HealthIndicator, 0, len(m.indicators))
	for _, ind := range m.indicators {
		if probed, ok := ind.(ProbedIndicator); ok {
			pt := probed.ProbeType()
			if pt == probeType || pt == ProbeBoth {
				filtered = append(filtered, ind)
			}
		} else {
			filtered = append(filtered, ind)
		}
	}

	aggregated := aggregate(r.Context(), filtered)
	code := http.StatusOK
	if aggregated.Status == StatusDown {
		code = http.StatusServiceUnavailable
	}

	response := map[string]any{"status": aggregated.Status}
	if len(aggregated.Details) > 0 {
		response["checks"] = aggregated.Details
	}
	web.WriteJSON(w, code, response)
}

func aggregate(ctx context.Context, indicators []HealthIndicator) Health {
	overall := StatusUp
	checks := map[string]any{}

	for _, ind := range indicators {
		health := ind.Health(ctx)
		name := ind.HealthIndicatorName()
		if name == "" {
			name = "unnamed"
		}
		if health.Status == StatusDown {
			overall = StatusDown
		} else if overall != StatusDown && (health.Status == StatusUnknown || health.Status == "") {
			overall = StatusUnknown
		}
		entry := map[string]any{"status": health.Status}
		for k, v := range health.Details {
			entry[k] = v
		}
		checks[name] = entry
	}

	return Health{Status: overall, Details: checks}
}
