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
