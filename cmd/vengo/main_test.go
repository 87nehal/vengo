package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "vengo") {
		t.Fatalf("stdout = %q, want version output", stdout.String())
	}
}

func TestNewCommandCreatesProjectFiles(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "orders-api")

	code := run([]string{"new", target, "github.com/example/orders-api"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	goMod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if !strings.Contains(string(goMod), "module github.com/example/orders-api") {
		t.Fatalf("unexpected go.mod content: %s", goMod)
	}

	mainSource, err := os.ReadFile(filepath.Join(target, "main.go"))
	if err != nil {
		t.Fatalf("read generated main.go: %v", err)
	}
	if !strings.Contains(string(mainSource), "github.com/87nehal/vengo/core") {
		t.Fatalf("generated main.go does not import the framework: %s", mainSource)
	}
	if !strings.Contains(string(mainSource), "hello from orders-api") {
		t.Fatalf("generated main.go does not use the project name: %s", mainSource)
	}
}

func TestNewCommandDefaultsModuleToProjectName(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "billing-api")

	code := run([]string{"new", target}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	goMod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if !strings.Contains(string(goMod), "module billing-api") {
		t.Fatalf("unexpected go.mod content: %s", goMod)
	}
}

func TestConfigCommandShowsResolvedConfig(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	content := []byte("[server]\nport = 8080\n\n[app]\nname = \"test-app\"\n")
	if err := os.WriteFile(filepath.Join(dir, "application.toml"), content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"config"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "server.port") {
		t.Fatalf("output missing server.port: %s", output)
	}
	if !strings.Contains(output, "app.name") {
		t.Fatalf("output missing app.name: %s", output)
	}
}

func TestConfigCommandRedactsSecrets(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	content := []byte("[database]\npassword = \"super-secret\"\n")
	if err := os.WriteFile(filepath.Join(dir, "application.toml"), content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"config"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, "super-secret") {
		t.Fatalf("output contains unredacted secret: %s", output)
	}
	if !strings.Contains(output, "<redacted>") {
		t.Fatalf("output missing redacted marker: %s", output)
	}
}

func TestDepsCommandPrintsGraph(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	graph := `[{"name":"newRepo","type":"*Repo","dependencies":[]},{"name":"newService","type":"*Service","dependencies":["newRepo"]}]`
	if err := os.WriteFile(filepath.Join(dir, "vengo-deps.json"), []byte(graph), 0o644); err != nil {
		t.Fatalf("write deps file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"deps"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Dependency Graph") {
		t.Fatalf("output missing header: %s", output)
	}
	if !strings.Contains(output, "newRepo") {
		t.Fatalf("output missing newRepo: %s", output)
	}
	if !strings.Contains(output, "newService") {
		t.Fatalf("output missing newService: %s", output)
	}
	if !strings.Contains(output, "<- newRepo") {
		t.Fatalf("output missing dependency arrow: %s", output)
	}
}

func TestDepsCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"deps"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no vengo-deps.json") {
		t.Fatalf("stderr = %q, want missing file message", stderr.String())
	}
}

func TestDepsCommandEmptyGraph(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	if err := os.WriteFile(filepath.Join(dir, "vengo-deps.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("write deps file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"deps"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no providers registered") {
		t.Fatalf("stdout = %q, want empty graph message", stdout.String())
	}
}
