package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSourceTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.toml")
	content := []byte("[server]\nport = 9090\nhost = \"localhost\"\n\n[app]\nname = \"demo\"\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	source := NewFileSource(path)
	values, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("load toml: %v", err)
	}

	if values["server.port"] != "9090" {
		t.Fatalf("server.port = %q, want 9090", values["server.port"])
	}
	if values["server.host"] != "localhost" {
		t.Fatalf("server.host = %q, want localhost", values["server.host"])
	}
	if values["app.name"] != "demo" {
		t.Fatalf("app.name = %q, want demo", values["app.name"])
	}
}

func TestFileSourceJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.json")
	content := []byte(`{"server":{"port":"8080"},"app":{"name":"json-app"}}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	source := NewFileSource(path)
	values, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("load json: %v", err)
	}

	if values["server.port"] != "8080" {
		t.Fatalf("server.port = %q, want 8080", values["server.port"])
	}
	if values["app.name"] != "json-app" {
		t.Fatalf("app.name = %q, want json-app", values["app.name"])
	}
}

func TestFileSourceUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.xml")
	if err := os.WriteFile(path, []byte("<config></config>"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	source := NewFileSource(path)
	_, err := source.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestFileSourceMissingFile(t *testing.T) {
	source := NewFileSource(filepath.Join(t.TempDir(), "missing.toml"))
	_, err := source.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileSourceName(t *testing.T) {
	source := NewFileSource("/etc/app/application.toml")
	if source.Name() != "file:/etc/app/application.toml" {
		t.Fatalf("name = %q, want file:/etc/app/application.toml", source.Name())
	}
}
