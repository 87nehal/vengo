package autoconfigure

import (
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
