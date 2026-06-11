package autoconfigure

import (
	"errors"
	"strings"
	"testing"

	"github.com/87nehal/vengo/core"
)

func TestRegistryAppliesMatchingConfigurer(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Configurer{
		Name:      "test",
		Condition: OnMissingService("message"),
		Configure: func(app *core.App) error {
			return app.Register("message", "hello")
		},
	})

	app := core.New("test")
	if err := registry.Apply(app); err != nil {
		t.Fatalf("apply registry: %v", err)
	}

	message, err := core.Get[string](app, "message")
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if message != "hello" {
		t.Fatalf("message = %q, want hello", message)
	}
}

func TestRegistrySkipsNonMatchingConfigurer(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Configurer{
		Name:      "test",
		Condition: func(*core.App) bool { return false },
		Configure: func(app *core.App) error {
			return app.Register("message", "hello")
		},
	})

	app := core.New("test")
	if err := registry.Apply(app); err != nil {
		t.Fatalf("apply registry: %v", err)
	}
	if app.Has("message") {
		t.Fatal("service was registered even though the condition was false")
	}
}

func TestRegistryRejectsMissingConfigureFunction(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Configurer{Name: "broken"})

	err := registry.Apply(core.New("test"))
	if err == nil {
		t.Fatal("expected missing configure function error")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error = %v, want configurer name", err)
	}
}

func TestRegistryWrapsConfigureError(t *testing.T) {
	registry := NewRegistry()
	expected := errors.New("boom")
	registry.Register(Configurer{
		Name:      "broken",
		Condition: Always,
		Configure: func(*core.App) error { return expected },
	})

	err := registry.Apply(core.New("test"))
	if !errors.Is(err, expected) {
		t.Fatalf("error = %v, want wrapped %v", err, expected)
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Fatalf("error = %v, want configurer name", err)
	}
}

func TestAlways(t *testing.T) {
	if !Always(core.New("test")) {
		t.Fatal("Always returned false")
	}
}
