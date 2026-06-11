package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type runOptions struct {
	Target    string
	Build     string
	NoRun     bool
	BuildOnly bool
	Debounce  time.Duration
}

func parseRunArgs(args []string) (runOptions, error) {
	opts := runOptions{Target: ".", Build: "go build -o vengo-app .", Debounce: 300 * time.Millisecond}
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--build="):
			opts.Build = strings.TrimPrefix(arg, "--build=")
		case arg == "--no-run":
			opts.NoRun = true
		case arg == "--build-only":
			opts.BuildOnly = true
		case strings.HasPrefix(arg, "--debounce="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--debounce="))
			if err != nil {
				return opts, fmt.Errorf("invalid --debounce: %w", err)
			}
			opts.Debounce = d
		default:
			opts.Target = arg
		}
	}
	return opts, nil
}

func runRun(stdout io.Writer, stderr io.Writer, args []string) int {
	opts, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return runDevMode(opts, stdout, stderr)
}

func runDevMode(opts runOptions, stdout io.Writer, stderr io.Writer) int {
	absTarget, err := filepath.Abs(opts.Target)
	if err != nil {
		fmt.Fprintf(stderr, "resolve target path: %v\n", err)
		return 1
	}

	if err := os.Chdir(absTarget); err != nil {
		fmt.Fprintf(stderr, "chdir: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		cancel()
	}()

	fmt.Fprintln(stdout, "vengo run: watching", opts.Target, "press Ctrl+C to stop")

	type buildResult struct {
		out io.Writer
	}
	_ = buildResult{}

	events := watchDir(ctx, absTarget, opts.Debounce)

	var mu sync.Mutex
	var currentProc *exec.Cmd

	trigger := func() {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintln(stdout, "[vengo run] rebuilding...")
		if currentProc != nil {
			_ = currentProc.Process.Kill()
			_, _ = currentProc.Process.Wait()
			currentProc = nil
		}
		buildCmd := buildCommand(ctx, opts.Build)
		buildCmd.Stdout = stdout
		buildCmd.Stderr = stderr
		err := buildCmd.Run()
		if err != nil {
			fmt.Fprintf(stdout, "[vengo run] build failed: %v\n", err)
			return
		}
		if opts.NoRun || opts.BuildOnly {
			return
		}
		proc := exec.CommandContext(ctx, "./vengo-app")
		proc.Stdout = stdout
		proc.Stderr = stderr
		if err := proc.Start(); err != nil {
			fmt.Fprintf(stdout, "[vengo run] start failed: %v\n", err)
			return
		}
		currentProc = proc
	}

	trigger()

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			if currentProc != nil {
				_ = currentProc.Process.Kill()
				_, _ = currentProc.Process.Wait()
			}
			mu.Unlock()
			return 0
		case ev, ok := <-events:
			if !ok {
				return 0
			}
			fmt.Fprintf(stdout, "[vengo run] change detected: %s\n", ev)
			trigger()
		}
	}
}

func buildCommand(ctx context.Context, cmdLine string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", cmdLine)
	}
	return exec.CommandContext(ctx, "sh", "-c", cmdLine)
}

func runCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func watchDir(ctx context.Context, dir string, debounce time.Duration) <-chan string {
	out := make(chan string, 16)
	go func() {
		defer close(out)
		snap := snapshotDir(dir)
		t := time.NewTimer(debounce)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				next := snapshotDir(dir)
				if changed, name := diffSnapshot(snap, next); changed {
					snap = next
					select {
					case out <- name:
					default:
					}
				}
				t.Reset(debounce)
			}
		}
	}()
	return out
}

type fileSnapshot map[string]fileInfo

type fileInfo struct {
	size    int64
	modTime int64
}

func snapshotDir(dir string) fileSnapshot {
	snap := make(fileSnapshot)
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		ext := filepath.Ext(rel)
		if ext != ".go" && ext != ".toml" && ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return nil
		}
		snap[rel] = fileInfo{size: info.Size(), modTime: info.ModTime().UnixNano()}
		return nil
	})
	return snap
}

func diffSnapshot(prev, next fileSnapshot) (bool, string) {
	for k, v := range next {
		if old, ok := prev[k]; !ok || old != v {
			return true, k
		}
	}
	for k := range prev {
		if _, ok := next[k]; !ok {
			return true, k
		}
	}
	return false, ""
}
