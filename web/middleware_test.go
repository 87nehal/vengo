package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/87nehal/vengo/core"
)

func TestMiddlewareChain(t *testing.T) {
	server := New(":0")

	var order []string

	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware1-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware1-after")
		})
	})

	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware2-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware2-after")
		})
	})

	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("middleware chain length = %d, want %d", len(order), len(expected))
	}

	for i, v := range order {
		if v != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestMiddlewareNilIgnored(t *testing.T) {
	server := New(":0")
	server.Use(nil)

	if len(server.middlewares) != 0 {
		t.Errorf("nil middleware was added, got %d middlewares", len(server.middlewares))
	}
}

func TestMiddlewareWithRealServer(t *testing.T) {
	server := New(":0")

	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "test")
			next.ServeHTTP(w, r)
		})
	})

	server.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	app := core.New("test", server)
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer app.Stop(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Header().Get("X-Custom") != "test" {
		t.Errorf("header X-Custom = %q, want %q", rec.Header().Get("X-Custom"), "test")
	}

	if rec.Body.String() != "hello" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "hello")
	}
}
