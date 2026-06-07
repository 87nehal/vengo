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
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := web.New(":8080")
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "hello from vengo")
	})

	app := core.New("hello", server, actuator.NewHealth())
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
