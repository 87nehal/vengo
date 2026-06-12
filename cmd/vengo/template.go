package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func vengoLocalPath() string {
	if path := os.Getenv("VENGO_LOCAL_PATH"); path != "" {
		return path
	}
	return ".."
}

type projectModule string

const (
	moduleWeb  projectModule = "web"
	moduleData projectModule = "data"
	moduleAuth projectModule = "auth"
)

var knownModules = map[projectModule]bool{
	moduleWeb:  true,
	moduleData: true,
	moduleAuth: true,
}

func parseNewArgs(args []string) (string, []projectModule, int) {
	if len(args) == 0 {
		return "", nil, 2
	}
	module := filepath.Base(args[0])
	rest := args[1:]
	positional := []string{}
	modulesFlag := false
	modules := []projectModule{}
	for _, a := range rest {
		switch {
		case strings.HasPrefix(a, "--modules="):
			modulesFlag = true
			raw := strings.TrimPrefix(a, "--modules=")
			for _, m := range strings.Split(raw, ",") {
				m = strings.TrimSpace(m)
				if m == "" {
					continue
				}
				if !knownModules[projectModule(m)] {
					return "", nil, 2
				}
				modules = append(modules, projectModule(m))
			}
		case a == "--modules":
			return "", nil, 2
		default:
			positional = append(positional, a)
		}
	}
	if len(positional) > 0 {
		module = positional[0]
	}
	if !modulesFlag {
		modules = []projectModule{moduleWeb}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i] < modules[j] })
	return module, modules, 0
}

func renderProjectTemplate(appName, module string, modules []projectModule) string {
	usesWeb := hasModule(modules, moduleWeb)
	usesData := hasModule(modules, moduleData)
	usesAuth := hasModule(modules, moduleAuth)
	usesHealth := usesWeb

	// Build standard library imports dynamically to avoid unused imports
	stdImports := []string{
		`	"context"`,
		`	"log"`,
		`	"time"`,
	}
	if usesWeb {
		stdImports = append(stdImports, `	"fmt"`, `	"net/http"`)
	}

	// Build Vengo imports dynamically
	vengoImports := []string{
		`	"github.com/87nehal/vengo/cmd"`,
		`	"github.com/87nehal/vengo/config"`,
		`	"github.com/87nehal/vengo/core"`,
	}
	if usesWeb {
		vengoImports = append(vengoImports, `	"github.com/87nehal/vengo/web"`)
	}
	if usesHealth {
		vengoImports = append(vengoImports, `	"github.com/87nehal/vengo/actuator"`)
	}
	if usesData {
		vengoImports = append(vengoImports, `	"github.com/87nehal/vengo/data"`, `	_ "modernc.org/sqlite"`)
	}
	if usesAuth {
		vengoImports = append(vengoImports, `	"github.com/87nehal/vengo/security"`)
	}

	var importsList []string
	importsList = append(importsList, stdImports...)
	importsList = append(importsList, "")
	importsList = append(importsList, vengoImports...)

	var modulesList []string
	if usesWeb {
		modulesList = append(modulesList, "server")
	}
	if usesHealth {
		modulesList = append(modulesList, "actuator.NewHealth()")
	}
	if usesData {
		modulesList = append(modulesList, "data.New()")
	}
	if usesAuth {
		modulesList = append(modulesList, "security.New()")
	}
	modulesExpr := strings.Join(modulesList, ", ")

	configStruct := `type AppConfig struct {
	Port int    ` + "`config:\"server.port\" default:\"8080\"`" + `
	Host string ` + "`config:\"server.host\" default:\"0.0.0.0\"`" + `
	Name string ` + "`config:\"app.name\" default:\"" + appName + "\"`" + `
}
`

	var serverBlock string
	if usesWeb {
		serverBlock = `	cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
	if err != nil {
		log.Fatal(err)
	}
	var appCfg AppConfig
	if err := config.Bind(cfg, &appCfg); err != nil {
		log.Fatal(err)
	}

	addr := fmt.Sprintf(":%d", appCfg.Port)
	server := web.New(addr)
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello from %s\n", appCfg.Name)
	})
`
	} else {
		serverBlock = `	cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
	if err != nil {
		log.Fatal(err)
	}
	var appCfg AppConfig
	if err := config.Bind(cfg, &appCfg); err != nil {
		log.Fatal(err)
	}
`
	}

	appStart := fmt.Sprintf(`	app := core.New(appCfg.Name, %s)
	app.SetConfig(cfg)`, modulesExpr)

	formatArgs := func(s string) string {
		s = strings.ReplaceAll(s, "{appName}", appName)
		s = strings.ReplaceAll(s, "{module}", module)
		return s
	}

	main := formatArgs(`package main

import (
` + strings.Join(importsList, "\n") + `
)

` + configStruct + `
func main() {
	ctx := cmd.NotifyShutdown(context.Background())

` + serverBlock + appStart + `

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
` + renderListenBlock(usesWeb) + `
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
`)

	return main
}

func renderListenBlock(usesWeb bool) string {
	if usesWeb {
		return "	log.Printf(\"listening on %s\\n\", server.Addr())\n"
	}
	return "	log.Printf(\"started %s\\n\", appCfg.Name)\n"
}

func hasModule(modules []projectModule, target projectModule) bool {
	for _, m := range modules {
		if m == target {
			return true
		}
	}
	return false
}
