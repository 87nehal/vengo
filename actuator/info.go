package actuator

import (
	"net/http"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

const InfoServiceName = "actuator.info"

type InfoModule struct {
	path    string
	appName string
	version string
	commit  string
	build   string
	extra   map[string]any
	enabled bool
}

type InfoOption func(*InfoModule)

func NewInfo(options ...InfoOption) *InfoModule {
	module := &InfoModule{
		path:    "/actuator/info",
		extra:   make(map[string]any),
		enabled: true,
	}
	for _, opt := range options {
		if opt != nil {
			opt(module)
		}
	}
	return module
}

func WithInfoPath(path string) InfoOption {
	return func(m *InfoModule) {
		if path != "" {
			m.path = path
		}
	}
}

func WithVersion(version string) InfoOption {
	return func(m *InfoModule) {
		m.version = version
	}
}

func WithCommit(commit string) InfoOption {
	return func(m *InfoModule) {
		m.commit = commit
	}
}

func WithBuild(build string) InfoOption {
	return func(m *InfoModule) {
		m.build = build
	}
}

func WithInfoExtra(key string, value any) InfoOption {
	return func(m *InfoModule) {
		if key != "" {
			m.extra[key] = value
		}
	}
}

func WithInfoEnabled(enabled bool) InfoOption {
	return func(m *InfoModule) {
		m.enabled = enabled
	}
}

func (m *InfoModule) Name() string {
	return "actuator.info"
}

func (m *InfoModule) Configure(app *core.App) error {
	if !m.enabled {
		return nil
	}
	server, err := core.Get[*web.Server](app, web.ServiceName)
	if err != nil {
		return err
	}
	if err := app.Register(InfoServiceName, m); err != nil {
		return err
	}
	m.appName = app.Name()
	server.HandleFunc(m.path, m.handle)
	return nil
}

func (m *InfoModule) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	info := map[string]any{
		"name": m.appName,
	}

	if m.version != "" {
		info["version"] = m.version
	}
	if m.commit != "" {
		info["commit"] = m.commit
	}
	if m.build != "" {
		info["build"] = m.build
	}

	for k, v := range m.extra {
		info[k] = v
	}

	web.WriteJSON(w, http.StatusOK, info)
}
