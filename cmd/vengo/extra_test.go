package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRoutesCommandPrintsRoutes(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	content := `[{"method":"GET","pattern":"/users"},{"method":"POST","pattern":"/users"},{"method":"","pattern":"/health"}]`
	if err := os.WriteFile(filepath.Join(dir, "vengo-routes.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"routes"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Registered Routes") {
		t.Fatalf("missing header: %s", output)
	}
	if !strings.Contains(output, "/users") || !strings.Contains(output, "/health") {
		t.Fatalf("missing routes: %s", output)
	}
	if !strings.Contains(output, "GET") || !strings.Contains(output, "POST") {
		t.Fatalf("missing methods: %s", output)
	}
	healthIdx := strings.Index(output, "/health")
	usersIdx := strings.Index(output, "/users")
	if healthIdx > usersIdx {
		t.Fatalf("expected /health before /users: %s", output)
	}
}

func TestRoutesCommandMissingFile(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"routes"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no vengo-routes.json") {
		t.Fatalf("stderr = %q, want missing file message", stderr.String())
	}
}

func TestRoutesCommandInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	if err := os.WriteFile(filepath.Join(dir, "vengo-routes.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"routes"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "parse vengo-routes.json") {
		t.Fatalf("stderr = %q, want parse error", stderr.String())
	}
}

func TestRoutesCommandEmptyList(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	if err := os.WriteFile(filepath.Join(dir, "vengo-routes.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"routes"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no routes registered") {
		t.Fatalf("stdout = %q, want empty message", stdout.String())
	}
}

func TestDoctorCommandRunsAllChecks(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test-doctor\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Vengo Doctor") {
		t.Fatalf("missing header: %s", output)
	}
	if !strings.Contains(output, "go version") {
		t.Fatalf("missing go version check: %s", output)
	}
	if !strings.Contains(output, "vengo version") {
		t.Fatalf("missing vengo version check: %s", output)
	}
	if !strings.Contains(output, "test-doctor") {
		t.Fatalf("missing module path: %s", output)
	}
	if !strings.Contains(output, "all checks passed") {
		t.Fatalf("missing summary line: %s", output)
	}
}

func TestDoctorCommandFailsWhenNoGoMod(t *testing.T) {
	dir := t.TempDir()
	original, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(original)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"doctor"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Fatalf("expected FAIL marker: %s", stdout.String())
	}
}

func TestNewCommandWithModulesFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "with-data")

	code := run([]string{"new", target, "github.com/example/with-data", "--modules=data"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	main, err := os.ReadFile(filepath.Join(target, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if strings.Contains(string(main), "web.New") {
		t.Fatalf("expected web module excluded, got: %s", main)
	}
}

func TestNewCommandWithUnknownModule(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "bad-modules")

	code := run([]string{"new", target, "github.com/example/bad", "--modules=foo"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestParseRunArgs(t *testing.T) {
	opts, err := parseRunArgs([]string{".", "--build=go build .", "--debounce=500ms", "--no-run"})
	if err != nil {
		t.Fatalf("parseRunArgs: %v", err)
	}
	if opts.Target != "." {
		t.Errorf("target = %q, want .", opts.Target)
	}
	if opts.Build != "go build ." {
		t.Errorf("build = %q", opts.Build)
	}
	if !opts.NoRun {
		t.Errorf("NoRun = false, want true")
	}
	if opts.Debounce != 500*time.Millisecond {
		t.Errorf("debounce = %v", opts.Debounce)
	}
}

func TestParseRunArgsInvalidDebounce(t *testing.T) {
	_, err := parseRunArgs([]string{".", "--debounce=not-a-duration"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSnapshotAndDiff(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	a := snapshotDir(dir)
	if _, ok := a["main.go"]; !ok {
		t.Fatalf("snapshot missing main.go: %v", a)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("ab"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	b := snapshotDir(dir)
	changed, name := diffSnapshot(a, b)
	if !changed {
		t.Fatalf("expected change detected")
	}
	if name != "main.go" {
		t.Errorf("name = %q, want main.go", name)
	}
}

func TestNewCommandCreatesApplicationToml(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "toml-app")

	code := run([]string{"new", target}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	toml, err := os.ReadFile(filepath.Join(target, "application.toml"))
	if err != nil {
		t.Fatalf("read toml: %v", err)
	}
	if !strings.Contains(string(toml), "toml-app") {
		t.Errorf("expected toml to mention app name: %s", toml)
	}
}

func TestBuildCommandUsesShellForCurrentOS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := buildCommand(ctx, "echo hello")
	if cmd.Process != nil {
		t.Fatalf("process should be nil before Start")
	}
	if runtime.GOOS == "windows" && cmd.Path == "" {
		t.Fatalf("expected cmd path on windows")
	}
}

func TestWatchDirDetectsChange(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	events := watchDir(ctx, dir, 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	select {
	case name := <-events:
		if name != "main.go" {
			t.Errorf("event name = %q, want main.go", name)
		}
	case <-time.After(800 * time.Millisecond):
		t.Fatalf("no change event received within timeout")
	}
}

func TestGeneratedMainCompiles(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}
	t.Setenv("VENGO_LOCAL_PATH", repoRoot)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = repoRoot
	if out, err := tidyCmd.CombinedOutput(); err != nil {
		t.Skipf("cannot run go mod tidy on repo root: %v\n%s", err, out)
	}

	tests := []struct {
		name    string
		modules string
	}{
		{"default", ""},
		{"data-only", "data"},
		{"full-stack", "web,data,auth"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			target := filepath.Join(t.TempDir(), "compile-test-"+tc.name)

			args := []string{"new", target, "github.com/example/compile-test-" + tc.name}
			if tc.modules != "" {
				args = append(args, "--modules="+tc.modules)
			}

			if code := run(args, &stdout, &stderr); code != 0 {
				t.Fatalf("new failed: %s", stderr.String())
			}

			tidyGenCmd := exec.Command("go", "mod", "tidy")
			tidyGenCmd.Dir = target
			if out, err := tidyGenCmd.CombinedOutput(); err != nil {
				t.Fatalf("go mod tidy on generated project: %v\n%s", err, out)
			}

			cmd := exec.Command("go", "build", "./...")
			cmd.Dir = target
			var buildOut bytes.Buffer
			buildOut.WriteString(stdout.String())
			buildErr := bytes.Buffer{}
			cmd.Stdout = &buildOut
			cmd.Stderr = &buildErr
			if err := cmd.Run(); err != nil {
				t.Fatalf("generated project failed to build: %v\nstdout=%s\nstderr=%s", err, buildOut.String(), buildErr.String())
			}
		})
	}
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if data, _ := os.ReadFile(filepath.Join(dir, "go.mod")); data != nil && bytes.Contains(data, []byte("github.com/87nehal/vengo")) {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found")
}
