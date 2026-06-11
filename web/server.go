package web

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/87nehal/vengo/core"
)

const ServiceName = "web.server"

type Middleware func(http.Handler) http.Handler

type Group struct {
	server      *Server
	prefix      string
	middlewares []Middleware
}

type Server struct {
	addr string
	mux  *http.ServeMux

	mu          sync.Mutex
	listener    net.Listener
	httpServer  *http.Server
	middlewares []Middleware
	routes      []Route
}

type Route struct {
	Pattern string
	Method  string
}

func New(addr string) *Server {
	if addr == "" {
		addr = ":8080"
	}
	return &Server{
		addr:        addr,
		mux:         http.NewServeMux(),
		middlewares: make([]Middleware, 0),
	}
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
	s.mu.Lock()
	s.routes = append(s.routes, parseRoute(pattern))
	s.mu.Unlock()
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mu.Lock()
	s.routes = append(s.routes, parseRoute(pattern))
	s.mu.Unlock()
	s.mux.HandleFunc(pattern, handler)
}

func parseRoute(pattern string) Route {
	method, path := parsePattern(pattern)
	return Route{
		Pattern: path,
		Method:  method,
	}
}

func parsePattern(pattern string) (method, path string) {
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func (s *Server) Handler() http.Handler {
	return applyMiddleware(s.mux, s.middlewares)
}

func applyMiddleware(handler http.Handler, middlewares []Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, e.Code, map[string]any{
		"error": e.Message,
	})
}

func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func WrapError(code int, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func BadRequest(message string) *Error {
	return NewError(http.StatusBadRequest, message)
}

func Unauthorized(message string) *Error {
	return NewError(http.StatusUnauthorized, message)
}

func Forbidden(message string) *Error {
	return NewError(http.StatusForbidden, message)
}

func NotFound(message string) *Error {
	return NewError(http.StatusNotFound, message)
}

func InternalServerError(message string) *Error {
	return NewError(http.StatusInternalServerError, message)
}

type Validator interface {
	Valid() error
}

func BindJSON(r *http.Request, target any) *Error {
	if r.Body == nil {
		return BadRequest("request body is empty")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return WrapError(http.StatusBadRequest, "invalid JSON", err)
	}

	if validator, ok := target.(Validator); ok {
		if err := validator.Valid(); err != nil {
			return WrapError(http.StatusBadRequest, "validation failed", err)
		}
	}

	return nil
}

func (s *Server) Use(middlewares ...Middleware) {
	for _, mw := range middlewares {
		if mw != nil {
			s.middlewares = append(s.middlewares, mw)
		}
	}
}

func (s *Server) Group(prefix string, middlewares ...Middleware) *Group {
	return &Group{
		server:      s,
		prefix:      prefix,
		middlewares: middlewares,
	}
}

func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) {
	fullPattern := g.prefix + pattern
	wrappedHandler := applyMiddleware(http.HandlerFunc(handler), g.middlewares)
	g.server.Handle(fullPattern, wrappedHandler)
}

func (g *Group) Handle(pattern string, handler http.Handler) {
	fullPattern := g.prefix + pattern
	wrappedHandler := applyMiddleware(handler, g.middlewares)
	g.server.Handle(fullPattern, wrappedHandler)
}

func (g *Group) Use(middlewares ...Middleware) {
	for _, mw := range middlewares {
		if mw != nil {
			g.middlewares = append(g.middlewares, mw)
		}
	}
}

func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

func (s *Server) Routes() []Route {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Route, len(s.routes))
	copy(result, s.routes)
	return result
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
