package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/87nehal/vengo/core"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func formatValidationErrors(err error) string {
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, fmt.Sprintf("Field '%s' failed on the '%s' tag", e.Field(), e.Tag()))
		}
		return strings.Join(msgs, ", ")
	}
	return err.Error()
}

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
	ipv4Only    bool
	readyHook   ReadyHook
	errCh       chan error
	app         *core.App
}

type Route struct {
	Pattern string
	Method  string
}

type ServerOption func(*Server)

func WithIPv4Only() ServerOption {
	return func(s *Server) {
		s.ipv4Only = true
	}
}

type ReadyHook func(addr string)

func (s *Server) OnReady(fn ReadyHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readyHook = fn
}

func (s *Server) ErrChan() <-chan error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errCh == nil {
		s.errCh = make(chan error, 1)
	}
	return s.errCh
}

func New(addr string, opts ...ServerOption) *Server {
	if addr == "" {
		addr = ":8080"
	}
	s := &Server{
		addr:        addr,
		mux:         http.NewServeMux(),
		middlewares: make([]Middleware, 0),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

func (s *Server) Name() string {
	return "web"
}

func (s *Server) Configure(app *core.App) error {
	s.mu.Lock()
	s.app = app
	s.mu.Unlock()
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

func (s *Server) HandleError(pattern string, handler ErrorHandlerFunc) {
	s.Handle(pattern, ErrorHandler(handler))
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

	if err := decoder.Decode(target); err != nil {
		return WrapError(http.StatusBadRequest, "invalid JSON", err)
	}

	if validator, ok := target.(Validator); ok {
		if err := validator.Valid(); err != nil {
			return WrapError(http.StatusBadRequest, "validation failed", err)
		}
	}

	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		if err := validate.Struct(target); err != nil {
			return WrapError(http.StatusBadRequest, formatValidationErrors(err), err)
		}
	}

	return nil
}

func BindJSONStrict(r *http.Request, target any) *Error {
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

	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		if err := validate.Struct(target); err != nil {
			return WrapError(http.StatusBadRequest, formatValidationErrors(err), err)
		}
	}

	return nil
}

func BindAndValidate[T any](r *http.Request) (T, *Error) {
	var target T
	if err := BindJSON(r, &target); err != nil {
		return target, err
	}
	return target, nil
}

func (s *Server) Use(middlewares ...Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	g.server.Handle(g.fullPattern(pattern), applyMiddleware(http.HandlerFunc(handler), g.middlewares))
}

func (g *Group) Handle(pattern string, handler http.Handler) {
	g.server.Handle(g.fullPattern(pattern), applyMiddleware(handler, g.middlewares))
}

func (g *Group) HandleError(pattern string, handler ErrorHandlerFunc) {
	g.Handle(pattern, ErrorHandler(handler))
}

// fullPattern prepends the group prefix to the path component of the
// pattern, preserving an optional Go 1.22 method prefix such as "GET /users".
func (g *Group) fullPattern(pattern string) string {
	method, path := parsePattern(pattern)
	if method == "" {
		return g.prefix + path
	}
	return method + " " + g.prefix + path
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

func (s *Server) SetAddr(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addr = addr
}

func (s *Server) Routes() []Route {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Route, len(s.routes))
	copy(result, s.routes)
	return result
}

func (s *Server) RoutesJSON() ([]byte, error) {
	routes := s.Routes()
	type jsonRoute struct {
		Method  string `json:"method"`
		Pattern string `json:"pattern"`
	}
	out := make([]jsonRoute, len(routes))
	for i, r := range routes {
		out[i] = jsonRoute{Method: r.Method, Pattern: r.Pattern}
	}
	return json.MarshalIndent(out, "", "  ")
}

func (s *Server) FormatRoutes(w io.Writer) {
	routes := s.Routes()
	if len(routes) == 0 {
		fmt.Fprintln(w, "no routes registered")
		return
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Pattern == routes[j].Pattern {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Pattern < routes[j].Pattern
	})
	fmt.Fprintln(w, "Registered Routes:")
	fmt.Fprintln(w, strings.Repeat("-", 50))
	for _, r := range routes {
		if r.Method == "" {
			fmt.Fprintf(w, "  %-7s %s\n", "*", r.Pattern)
		} else {
			fmt.Fprintf(w, "  %-7s %s\n", r.Method, r.Pattern)
		}
	}
}

type readyListener struct {
	net.Listener
	once    sync.Once
	onReady func()
}

func (l *readyListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		l.once.Do(l.onReady)
	}
	return conn, err
}

func (s *Server) Start(context.Context) error {
	s.mu.Lock()
	hasServer := s.httpServer != nil
	s.mu.Unlock()
	if hasServer {
		return nil
	}

	if s.app != nil {
		registrarType := reflect.TypeOf((*RouteRegistrar)(nil)).Elem()
		registrars := s.app.Container().ProvidersImplementing(registrarType)
		for _, t := range registrars {
			instance, err := s.app.ResolveType(t)
			if err != nil {
				return fmt.Errorf("resolve route registrar %s: %w", t, err)
			}
			if registrar, ok := instance.(RouteRegistrar); ok {
				group := registrar.Routes()
				g := s.Group(group.Prefix)
				for _, r := range group.Routes {
					pattern := r.Method + " " + r.Pattern
					if r.Method == "" {
						pattern = r.Pattern
					}

					switch h := r.Handler.(type) {
					case ErrorHandlerFunc:
						g.HandleError(pattern, h)
					case func(http.ResponseWriter, *http.Request) error:
						g.HandleError(pattern, h)
					case http.Handler:
						g.Handle(pattern, h)
					case func(http.ResponseWriter, *http.Request):
						g.HandleFunc(pattern, h)
					case http.HandlerFunc:
						g.Handle(pattern, h)
					default:
						return fmt.Errorf("unsupported handler type: %T for route %s", r.Handler, pattern)
					}
				}
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.httpServer != nil {
		return nil
	}

	network := "tcp"
	if s.ipv4Only {
		network = "tcp4"
	}

	listener, err := net.Listen(network, s.addr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           applyMiddleware(s.mux, s.middlewares),
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.listener = listener
	s.httpServer = server

	var rListener net.Listener = listener
	if s.readyHook != nil {
		rListener = &readyListener{
			Listener: listener,
			onReady: func() {
				s.mu.Lock()
				hook := s.readyHook
				s.mu.Unlock()
				if hook != nil {
					hook(listener.Addr().String())
				}
			},
		}
	}

	go func() {
		log.Printf("web server listening on %s (network: %s)", listener.Addr().String(), network)
		if err := server.Serve(rListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("web server error: %v", err)
			s.mu.Lock()
			if s.errCh != nil {
				select {
				case s.errCh <- err:
				default:
				}
			}
			s.mu.Unlock()
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

func WriteJSON(w http.ResponseWriter, status int, value any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(value)
}
