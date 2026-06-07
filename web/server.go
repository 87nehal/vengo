package web

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/87nehal/vengo/core"
)

const ServiceName = "web.server"

type Server struct {
	addr string
	mux  *http.ServeMux

	mu         sync.Mutex
	listener   net.Listener
	httpServer *http.Server
}

func New(addr string) *Server {
	if addr == "" {
		addr = ":8080"
	}
	return &Server{addr: addr, mux: http.NewServeMux()}
}

func (s *Server) Name() string {
	return "web"
}

func (s *Server) Configure(app *core.App) error {
	if err := app.Register(ServiceName, s); err != nil {
		return err
	}
	app.RegisterHook(core.Hook{
		Name:  ServiceName,
		Start: s.Start,
		Stop:  s.Stop,
	})
	return nil
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

func (s *Server) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpServer != nil {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.listener = listener
	s.httpServer = server

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Future diagnostics package will record background server failures.
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	server := s.httpServer
	s.httpServer = nil
	s.listener = nil
	s.mu.Unlock()

	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
