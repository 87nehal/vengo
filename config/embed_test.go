package config

import (
	"context"
	"testing"
	"testing/fstest"
)

func TestEmbedSourceTOML(t *testing.T) {
	fsys := fstest.MapFS{
		"application.toml": &fstest.MapFile{
			Data: []byte("[server]\nport = 3000\nhost = \"embedded\"\n"),
		},
	}

	source := NewEmbedSource(fsys, "application.toml")
	values, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("load embedded toml: %v", err)
	}

	if values["server.port"] != "3000" {
		t.Fatalf("server.port = %q, want 3000", values["server.port"])
	}
	if values["server.host"] != "embedded" {
		t.Fatalf("server.host = %q, want embedded", values["server.host"])
	}
}

func TestEmbedSourceJSON(t *testing.T) {
	fsys := fstest.MapFS{
		"config.json": &fstest.MapFile{
			Data: []byte(`{"app":{"name":"embedded-app"}}`),
		},
	}

	source := NewEmbedSource(fsys, "config.json")
	values, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("load embedded json: %v", err)
	}

	if values["app.name"] != "embedded-app" {
		t.Fatalf("app.name = %q, want embedded-app", values["app.name"])
	}
}

func TestEmbedSourceMissingFile(t *testing.T) {
	fsys := fstest.MapFS{}
	source := NewEmbedSource(fsys, "missing.toml")
	_, err := source.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for missing embedded file")
	}
}

func TestEmbedSourceUnsupportedExtension(t *testing.T) {
	fsys := fstest.MapFS{
		"config.xml": &fstest.MapFile{
			Data: []byte("<config></config>"),
		},
	}

	source := NewEmbedSource(fsys, "config.xml")
	_, err := source.Load(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestEmbedSourceName(t *testing.T) {
	fsys := fstest.MapFS{}
	source := NewEmbedSource(fsys, "application.toml")
	if source.Name() != "embed:application.toml" {
		t.Fatalf("name = %q, want embed:application.toml", source.Name())
	}
}
