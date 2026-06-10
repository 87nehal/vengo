package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/87nehal/vengo/config"
)

const version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version":
		_, _ = fmt.Fprintf(stdout, "vengo %s\n", version)
		return 0
	case "new":
		if len(args) < 2 {
			_, _ = fmt.Fprintln(stderr, "usage: vengo new <dir> [module]")
			return 2
		}
		module := filepath.Base(args[1])
		if len(args) > 2 {
			module = args[2]
		}
		if err := createProject(args[1], module); err != nil {
			_, _ = fmt.Fprintf(stderr, "create project: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "created %s\n", args[1])
		return 0
	case "config":
		profile := ""
		if len(args) > 1 {
			profile = args[1]
		}
		return runConfig(stdout, stderr, profile)
	case "deps":
		return runDeps(stdout, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vengo commands:")
	_, _ = fmt.Fprintln(w, "  version")
	_, _ = fmt.Fprintln(w, "  new <dir> [module]")
	_, _ = fmt.Fprintln(w, "  config [profile]")
	_, _ = fmt.Fprintln(w, "  deps")
}

func createProject(dir string, module string) error {
	if module == "" || module == "." || module == string(filepath.Separator) {
		return fmt.Errorf("module path cannot be empty")
	}

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory %q already exists", dir)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	goMod := fmt.Sprintf("module %s\n\ngo 1.22\n", module)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return err
	}

	appName := filepath.Base(dir)
	mainSource := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/87nehal/vengo/actuator"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := web.New(":8080")
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "hello from %s")
	})

	app := core.New("%s", server, actuator.NewHealth())
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %%s", server.Addr())

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
`, appName, appName)

	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSource), 0o644)
}

func runConfig(stdout io.Writer, stderr io.Writer, profile string) int {
	if profile == "" {
		profile = config.ActiveProfile()
	}

	cfg, err := config.LoadDefaults(context.Background(), profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	report := cfg.Report()
	if len(report) == 0 {
		_, _ = fmt.Fprintln(stdout, "no configuration values found")
		return 0
	}

	if profile != "" {
		_, _ = fmt.Fprintf(stdout, "active profile: %s\n\n", profile)
	}

	for _, entry := range report {
		_, _ = fmt.Fprintf(stdout, "%-40s = %-20s [%s]\n", entry.Key, entry.Value, entry.Source)
	}
	return 0
}

type depsNode struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Dependencies []string `json:"dependencies,omitempty"`
}

func runDeps(stdout io.Writer, stderr io.Writer) int {
	data, err := os.ReadFile("vengo-deps.json")
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(stderr, "no vengo-deps.json found in current directory")
			_, _ = fmt.Fprintln(stderr, "generate it from your app with:")
			_, _ = fmt.Fprintln(stderr, "  data, _ := app.Container().GraphJSON()")
			_, _ = fmt.Fprintln(stderr, "  os.WriteFile(\"vengo-deps.json\", data, 0644)")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "read vengo-deps.json: %v\n", err)
		return 1
	}

	var nodes []depsNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		_, _ = fmt.Fprintf(stderr, "parse vengo-deps.json: %v\n", err)
		return 1
	}

	if len(nodes) == 0 {
		_, _ = fmt.Fprintln(stdout, "no providers registered")
		return 0
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	_, _ = fmt.Fprintln(stdout, "Dependency Graph:")
	_, _ = fmt.Fprintln(stdout, "--------------------------------------------------")
	for _, node := range nodes {
		if len(node.Dependencies) == 0 {
			_, _ = fmt.Fprintf(stdout, "  %s (%s)\n", node.Name, node.Type)
		} else {
			_, _ = fmt.Fprintf(stdout, "  %s (%s)\n", node.Name, node.Type)
			for _, dep := range node.Dependencies {
				_, _ = fmt.Fprintf(stdout, "    <- %s\n", dep)
			}
		}
	}
	return 0
}
