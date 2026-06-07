package autoconfigure

import (
	"fmt"
	"sync"

	"github.com/87nehal/vengo/core"
)

type Condition func(*core.App) bool

type Configurer struct {
	Name      string
	Condition Condition
	Configure func(*core.App) error
}

type Registry struct {
	mu          sync.Mutex
	configurers []Configurer
}

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{}
}

func Register(configurer Configurer) {
	DefaultRegistry.Register(configurer)
}

func Apply(app *core.App) error {
	return DefaultRegistry.Apply(app)
}

func (r *Registry) Register(configurer Configurer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configurers = append(r.configurers, configurer)
}

func (r *Registry) Apply(app *core.App) error {
	r.mu.Lock()
	configurers := append([]Configurer(nil), r.configurers...)
	r.mu.Unlock()

	for _, configurer := range configurers {
		if configurer.Configure == nil {
			return fmt.Errorf("autoconfigurer %q has no configure function", configurer.Name)
		}
		if configurer.Condition != nil && !configurer.Condition(app) {
			continue
		}
		if err := configurer.Configure(app); err != nil {
			return fmt.Errorf("apply autoconfigurer %q: %w", configurer.Name, err)
		}
	}
	return nil
}

func Always(*core.App) bool {
	return true
}

func OnMissingService(name string) Condition {
	return func(app *core.App) bool {
		return !app.Has(name)
	}
}
