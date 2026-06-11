package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/87nehal/vengo/config"
)

type routeEntry struct {
	Method  string `json:"method"`
	Pattern string `json:"pattern"`
}

func runRoutes(stdout io.Writer, stderr io.Writer) int {
	data, err := os.ReadFile("vengo-routes.json")
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(stderr, "no vengo-routes.json found in current directory")
			_, _ = fmt.Fprintln(stderr, "generate it from your app with:")
			_, _ = fmt.Fprintln(stderr, "  data, _ := webServer.RoutesJSON()")
			_, _ = fmt.Fprintln(stderr, "  os.WriteFile(\"vengo-routes.json\", data, 0644)")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "read vengo-routes.json: %v\n", err)
		return 1
	}

	var routes []routeEntry
	if err := json.Unmarshal(data, &routes); err != nil {
		_, _ = fmt.Fprintf(stderr, "parse vengo-routes.json: %v\n", err)
		return 1
	}

	if len(routes) == 0 {
		_, _ = fmt.Fprintln(stdout, "no routes registered")
		return 0
	}

	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Pattern != routes[j].Pattern {
			return routes[i].Pattern < routes[j].Pattern
		}
		return routes[i].Method < routes[j].Method
	})

	_, _ = fmt.Fprintln(stdout, "Registered Routes:")
	_, _ = fmt.Fprintln(stdout, "--------------------------------------------------")
	for _, r := range routes {
		method := r.Method
		if method == "" {
			method = "*"
		}
		_, _ = fmt.Fprintf(stdout, "  %-7s %s\n", method, r.Pattern)
	}
	return 0
}

type doctorCheck struct {
	Name    string
	OK      bool
	Message string
}

func runDoctor(stdout io.Writer, stderr io.Writer) int {
	checks := []doctorCheck{
		checkGoVersion(),
		checkModulePath(),
		checkVengoVersion(),
		checkEnvSanity(),
	}

	failed := 0
	_, _ = fmt.Fprintln(stdout, "Vengo Doctor")
	_, _ = fmt.Fprintln(stdout, strings.Repeat("=", 50))
	for _, c := range checks {
		marker := "OK"
		if !c.OK {
			marker = "FAIL"
			failed++
		}
		_, _ = fmt.Fprintf(stdout, "  [%s] %-20s %s\n", marker, c.Name, c.Message)
	}
	_, _ = fmt.Fprintln(stdout, strings.Repeat("=", 50))
	if failed > 0 {
		_, _ = fmt.Fprintf(stdout, "%d check(s) failed\n", failed)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "all checks passed")
	return 0
}

func checkGoVersion() doctorCheck {
	out, err := captureGoVersion()
	if err != nil {
		return doctorCheck{Name: "go version", OK: false, Message: "go not found: " + err.Error()}
	}
	return doctorCheck{Name: "go version", OK: true, Message: out}
}

func checkModulePath() doctorCheck {
	wd, err := os.Getwd()
	if err != nil {
		return doctorCheck{Name: "module path", OK: false, Message: err.Error()}
	}
	data, err := os.ReadFile(filepath.Join(wd, "go.mod"))
	if err != nil {
		return doctorCheck{Name: "module path", OK: false, Message: "no go.mod in current directory"}
	}
	module := parseGoModModule(data)
	if module == "" {
		return doctorCheck{Name: "module path", OK: false, Message: "go.mod has no module directive"}
	}
	return doctorCheck{Name: "module path", OK: true, Message: module}
}

func checkVengoVersion() doctorCheck {
	return doctorCheck{Name: "vengo version", OK: true, Message: version}
}

func checkEnvSanity() doctorCheck {
	if os.Getenv("APP_PROFILE") == "" && os.Getenv("VENGO_PROFILE") == "" {
		return doctorCheck{Name: "env", OK: true, Message: "no profile set (using defaults)"}
	}
	profile := config.ActiveProfile()
	return doctorCheck{Name: "env", OK: true, Message: "active profile = " + profile}
}

func captureGoVersion() (string, error) {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseGoModModule(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
