package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSourcesFindsBaseFile(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	content := []byte("[server]\nport = 8080\n")
	if err := os.WriteFile(filepath.Join(dir, "application.toml"), content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sources := DefaultSources("")
	found := false
	for _, s := range sources {
		if s.Name() == "file:application.toml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected file:application.toml in sources, got %v", sourceNames(sources))
	}
}

func TestDefaultSourcesFindsProfileFile(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	base := []byte("[server]\nport = 8080\n")
	profile := []byte("[server]\nport = 9090\n")
	if err := os.WriteFile(filepath.Join(dir, "application.toml"), base, 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "application-prod.toml"), profile, 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	sources := DefaultSources("prod")
	foundProfile := false
	for _, s := range sources {
		if s.Name() == "file:application-prod.toml" {
			foundProfile = true
			break
		}
	}
	if !foundProfile {
		t.Fatalf("expected file:application-prod.toml in sources, got %v", sourceNames(sources))
	}
}

func TestProfileOverridesBase(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	base := []byte("[server]\nport = 8080\n\n[app]\nname = \"demo\"\n")
	profile := []byte("[server]\nport = 9090\n")
	if err := os.WriteFile(filepath.Join(dir, "application.toml"), base, 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "application-prod.toml"), profile, 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	cfg, err := LoadDefaults(context.Background(), "prod")
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	port, _ := cfg.Get("server.port")
	if port != "9090" {
		t.Fatalf("server.port = %q, want 9090", port)
	}

	name, _ := cfg.Get("app.name")
	if name != "demo" {
		t.Fatalf("app.name = %q, want demo", name)
	}
}

func TestActiveProfileFromEnv(t *testing.T) {
	t.Setenv("APP_PROFILE", "staging")
	t.Setenv("VENGO_PROFILE", "")

	profile := ActiveProfile()
	if profile != "staging" {
		t.Fatalf("profile = %q, want staging", profile)
	}
}

func TestActiveProfileFallsBackToVengoProfile(t *testing.T) {
	t.Setenv("APP_PROFILE", "")
	t.Setenv("VENGO_PROFILE", "dev")

	profile := ActiveProfile()
	if profile != "dev" {
		t.Fatalf("profile = %q, want dev", profile)
	}
}

func TestActiveProfileEmptyWhenUnset(t *testing.T) {
	t.Setenv("APP_PROFILE", "")
	t.Setenv("VENGO_PROFILE", "")

	profile := ActiveProfile()
	if profile != "" {
		t.Fatalf("profile = %q, want empty", profile)
	}
}

func TestDefaultSourcesIncludesEnvSource(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	sources := DefaultSources("")
	found := false
	for _, s := range sources {
		if s.Name() == "env:APP_" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected env:APP_ in sources, got %v", sourceNames(sources))
	}
}

func TestFileExistsReturnsFalseForDirectory(t *testing.T) {
	dir := t.TempDir()
	if fileExists(dir) {
		t.Fatal("fileExists returned true for directory")
	}
}

func sourceNames(sources []Source) []string {
	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = s.Name()
	}
	return names
}
