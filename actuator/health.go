package actuator

import (
	"context"
	"fmt"
	"net/http"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const HealthServiceName = "actuator.health"

type Check struct {
	Name  string
	Check func(context.Context) error
}

type HealthModule struct {
	path   string
	checks []Check
}

type Option func(*HealthModule)

func NewHealth(checks ...Check) *HealthModule {
	return NewHealthWithOptions(WithChecks(checks...))
}

func NewHealthWithOptions(options ...Option) *HealthModule {
	module := &HealthModule{path: "/actuator/health"}
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
		module.checks = append(module.checks, checks...)
	}
}

func (m *HealthModule) Name() string {
	return "actuator.health"
}

func (m *HealthModule) Configure(app *core.App) error {
	server, err := core.Get[*web.Server](app, web.ServiceName)
	if err != nil {
		return fmt.Errorf("health module requires web module: %w", err)
	}
	if err := app.Register(HealthServiceName, m); err != nil {
		return err
	}
	server.HandleFunc(m.path, m.handle)
	return nil
}

func (m *HealthModule) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	status := "UP"
	code := http.StatusOK
	checks := map[string]string{}
	for _, check := range m.checks {
		if check.Check == nil {
			continue
		}
		name := check.Name
		if name == "" {
			name = "unnamed"
		}
		if err := check.Check(r.Context()); err != nil {
			status = "DOWN"
			code = http.StatusServiceUnavailable
			checks[name] = err.Error()
			continue
		}
		checks[name] = "UP"
	}

	web.WriteJSON(w, code, map[string]any{
		"status": status,
		"checks": checks,
	})
}
