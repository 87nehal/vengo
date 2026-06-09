package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

const ConfigServiceName = "config"

type Module interface {
	Name() string
	Configure(app *App) error
}

type Hook struct {
	Name  string
	Start func(context.Context) error
	Stop  func(context.Context) error
}

type App struct {
	name string

	mu           sync.Mutex
	modules      []Module
	hooks        []Hook
	startedHooks []Hook
	services     map[string]any
	configured   bool
	started      bool
}

func New(name string, modules ...Module) *App {
	app := &App{
		name:     name,
		services: make(map[string]any),
	}
	app.modules = append(app.modules, modules...)
	return app
}

func (a *App) Name() string {
	return a.name
}

func (a *App) Use(module Module) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.modules = append(a.modules, module)
}

func (a *App) RegisterHook(hook Hook) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hooks = append(a.hooks, hook)
}

func (a *App) Register(name string, value any) error {
	if name == "" {
		return errors.New("service name cannot be empty")
	}
	if value == nil {
		return fmt.Errorf("service %q cannot be nil", name)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.services[name]; exists {
		return fmt.Errorf("service %q is already registered", name)
	}
	a.services[name] = value
	return nil
}

func (a *App) Has(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, exists := a.services[name]
	return exists
}

func (a *App) Get(name string) (any, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	value, exists := a.services[name]
	return value, exists
}

func (a *App) ServiceNames() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	names := make([]string, 0, len(a.services))
	for name := range a.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (a *App) SetConfig(cfg any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.services[ConfigServiceName] = cfg
}

func (a *App) Config() (any, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	value, exists := a.services[ConfigServiceName]
	return value, exists
}

func Get[T any](app *App, name string) (T, error) {
	var zero T
	value, exists := app.Get(name)
	if !exists {
		return zero, fmt.Errorf("service %q is not registered", name)
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("service %q has type %T", name, value)
	}
	return typed, nil
}

func (a *App) Start(ctx context.Context) error {
	if err := a.configure(); err != nil {
		return err
	}

	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return nil
	}
	hooks := append([]Hook(nil), a.hooks...)
	a.mu.Unlock()

	started := make([]Hook, 0, len(hooks))
	for _, hook := range hooks {
		if hook.Start == nil {
			started = append(started, hook)
			continue
		}
		if err := hook.Start(ctx); err != nil {
			_ = stopHooks(ctx, started)
			return fmt.Errorf("start hook %q: %w", hook.Name, err)
		}
		started = append(started, hook)
	}

	a.mu.Lock()
	a.startedHooks = started
	a.started = true
	a.mu.Unlock()
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.started {
		a.mu.Unlock()
		return nil
	}
	hooks := append([]Hook(nil), a.startedHooks...)
	a.startedHooks = nil
	a.started = false
	a.mu.Unlock()

	return stopHooks(ctx, hooks)
}

func (a *App) configure() error {
	a.mu.Lock()
	if a.configured {
		a.mu.Unlock()
		return nil
	}
	modules := append([]Module(nil), a.modules...)
	a.mu.Unlock()

	for _, module := range modules {
		if module == nil {
			return errors.New("module cannot be nil")
		}
		if err := module.Configure(a); err != nil {
			return fmt.Errorf("configure module %q: %w", module.Name(), err)
		}
	}

	a.mu.Lock()
	a.configured = true
	a.mu.Unlock()
	return nil
}

func stopHooks(ctx context.Context, hooks []Hook) error {
	var errs []error
	for index := len(hooks) - 1; index >= 0; index-- {
		hook := hooks[index]
		if hook.Stop == nil {
			continue
		}
		if err := hook.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop hook %q: %w", hook.Name, err))
		}
	}
	return errors.Join(errs...)
}
