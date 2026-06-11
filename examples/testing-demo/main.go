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

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

// MessageService defines the business logic contract.
type MessageService interface {
	Greet(name string) string
}

// SimpleMessageService is the real implementation of MessageService.
type SimpleMessageService struct {
	Prefix string
}

func (s *SimpleMessageService) Greet(name string) string {
	return fmt.Sprintf("%s, %s!", s.Prefix, name)
}

// UserHandler handles HTTP traffic and depends on MessageService.
type UserHandler struct {
	MsgSrv MessageService `inject:""`
}

func (h *UserHandler) Register(server *web.Server) {
	server.HandleFunc("GET /greet", h.HandleGreet)
}

func (h *UserHandler) HandleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Guest"
	}
	greeting := h.MsgSrv.Greet(name)
	web.WriteJSON(w, http.StatusOK, map[string]string{"message": greeting})
}

// AppConfig binds application settings.
type AppConfig struct {
	Prefix string `config:"greeting.prefix" default:"Hello"`
	Port   int    `config:"server.port" default:"8080"`
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

	server := web.New(fmt.Sprintf(":%d", appCfg.Port))

	app := core.New("testing-demo", server)
	app.SetConfig(cfg)

	err = core.Provide(app, func() MessageService {
		return &SimpleMessageService{Prefix: appCfg.Prefix}
	})
	if err != nil {
		log.Fatal(err)
	}

	err = core.Provide(app, func() *UserHandler {
		return &UserHandler{}
	})
	if err != nil {
		log.Fatal(err)
	}

	handler, err := core.Resolve[*UserHandler](app)
	if err != nil {
		log.Fatal(err)
	}
	handler.Register(server)

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("App listening on %s", server.Addr())

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
