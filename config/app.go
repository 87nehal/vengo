package config

import (
	"fmt"

	"github.com/87nehal/vengo/core"
)

func FromApp(app *core.App) (*Config, error) {
	raw, exists := app.Config()
	if !exists {
		return nil, fmt.Errorf("no config registered on app")
	}
	cfg, ok := raw.(*Config)
	if !ok {
		return nil, fmt.Errorf("registered config has type %T, want *config.Config", raw)
	}
	return cfg, nil
}

func BindFromApp(app *core.App, target any) error {
	cfg, err := FromApp(app)
	if err != nil {
		return err
	}
	return Bind(cfg, target)
}
