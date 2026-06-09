package config

import (
	"context"
	"testing"

	"github.com/87nehal/vengo/core"
)

func TestLoadUsesLaterSourcesAsOverrides(t *testing.T) {
	config, err := Load(context.Background(),
		NewMapSource("defaults", map[string]string{"server.port": "8080", "app.name": "demo"}),
		NewMapSource("env", map[string]string{"server.port": "9090"}),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	port, _ := config.Get("server.port")
	if port != "9090" {
		t.Fatalf("server.port = %q, want 9090", port)
	}
}

func TestReportRedactsSensitiveValues(t *testing.T) {
	config, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"database.password": "secret-value",
		"server.port":       "8080",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	report := config.Report()
	for _, entry := range report {
		if entry.Key == "database.password" && (!entry.Redacted || entry.Value != "<redacted>") {
			t.Fatalf("sensitive entry was not redacted: %+v", entry)
		}
	}
}

func TestKeysReturnsSortedKeys(t *testing.T) {
	cfg, err := Load(context.Background(), NewMapSource("test", map[string]string{
		"b.key": "2",
		"a.key": "1",
		"c.key": "3",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	keys := cfg.Keys()
	if len(keys) != 3 {
		t.Fatalf("len(keys) = %d, want 3", len(keys))
	}
	if keys[0] != "a.key" || keys[1] != "b.key" || keys[2] != "c.key" {
		t.Fatalf("keys = %v, want [a.key b.key c.key]", keys)
	}
}

func TestSourceOfReturnsSourceName(t *testing.T) {
	cfg, err := Load(context.Background(),
		NewMapSource("defaults", map[string]string{"server.port": "8080"}),
		NewMapSource("overrides", map[string]string{"server.port": "9090"}),
	)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	source, ok := cfg.SourceOf("server.port")
	if !ok {
		t.Fatal("SourceOf returned false for existing key")
	}
	if source != "overrides" {
		t.Fatalf("source = %q, want overrides", source)
	}
}

func TestSourceOfReturnsFalseForMissingKey(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{}))
	_, ok := cfg.SourceOf("missing.key")
	if ok {
		t.Fatal("SourceOf returned true for missing key")
	}
}

func TestFromAppReturnsConfig(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{"key": "value"}))
	app := core.New("test")
	app.SetConfig(cfg)

	retrieved, err := FromApp(app)
	if err != nil {
		t.Fatalf("FromApp: %v", err)
	}
	val, _ := retrieved.Get("key")
	if val != "value" {
		t.Fatalf("key = %q, want value", val)
	}
}

func TestFromAppFailsWhenNoConfig(t *testing.T) {
	app := core.New("test")
	_, err := FromApp(app)
	if err == nil {
		t.Fatal("expected error when no config registered")
	}
}

func TestBindFromApp(t *testing.T) {
	cfg, _ := Load(context.Background(), NewMapSource("test", map[string]string{"server.port": "3000"}))
	app := core.New("test")
	app.SetConfig(cfg)

	type ServerConfig struct {
		Port int `config:"server.port" default:"8080"`
	}

	var s ServerConfig
	if err := BindFromApp(app, &s); err != nil {
		t.Fatalf("BindFromApp: %v", err)
	}
	if s.Port != 3000 {
		t.Fatalf("port = %d, want 3000", s.Port)
	}
}
