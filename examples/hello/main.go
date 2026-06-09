package main

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
	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

type AppConfig struct {
	Port int    `config:"server.port" default:"8080"`
	Host string `config:"server.host" default:"localhost"`
	Name string `config:"app.name" default:"hello"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
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

	app := core.New(appCfg.Name, server, actuator.NewHealth())
	app.SetConfig(cfg)

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s", server.Addr())

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
