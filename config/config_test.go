package config

import (
	"context"
	"testing"
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
