package web

import (
	"net/http"
	"testing"
)

func TestRouteRegistry_EmptyServer(t *testing.T) {
	server := New(":0")
	routes := server.Routes()

	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestRouteRegistry_SingleRoute(t *testing.T) {
	server := New(":0")
	server.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes := server.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	if routes[0].Pattern != "/users" {
		t.Errorf("pattern = %q, want %q", routes[0].Pattern, "/users")
	}
	if routes[0].Method != "" {
		t.Errorf("method = %q, want empty", routes[0].Method)
	}
}

func TestRouteRegistry_MultipleRoutes(t *testing.T) {
	server := New(":0")
	server.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {})

	routes := server.Routes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	expected := []string{"/users", "/posts", "/comments"}
	for i, route := range routes {
		if route.Pattern != expected[i] {
			t.Errorf("route[%d].Pattern = %q, want %q", i, route.Pattern, expected[i])
		}
	}
}

func TestRouteRegistry_WithHTTPMethod(t *testing.T) {
	server := New(":0")
	server.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {})

	routes := server.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	if routes[0].Pattern != "/users" {
		t.Errorf("pattern = %q, want %q", routes[0].Pattern, "/users")
	}
	if routes[0].Method != "GET" {
		t.Errorf("method = %q, want %q", routes[0].Method, "GET")
	}
}

func TestRouteRegistry_MixedMethods(t *testing.T) {
	server := New(":0")
	server.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {})

	routes := server.Routes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	expected := []struct {
		method  string
		pattern string
	}{
		{"GET", "/users"},
		{"POST", "/users"},
		{"", "/posts"},
	}

	for i, route := range routes {
		if route.Method != expected[i].method {
			t.Errorf("route[%d].Method = %q, want %q", i, route.Method, expected[i].method)
		}
		if route.Pattern != expected[i].pattern {
			t.Errorf("route[%d].Pattern = %q, want %q", i, route.Pattern, expected[i].pattern)
		}
	}
}

func TestRouteRegistry_GroupRoutes(t *testing.T) {
	server := New(":0")
	api := server.Group("/api/v1")
	api.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {})
	api.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {})

	routes := server.Routes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	expected := []string{"/api/v1/users", "/api/v1/posts"}
	for i, route := range routes {
		if route.Pattern != expected[i] {
			t.Errorf("route[%d].Pattern = %q, want %q", i, route.Pattern, expected[i])
		}
	}
}

func TestRouteRegistry_ReturnsCopy(t *testing.T) {
	server := New(":0")
	server.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {})

	routes1 := server.Routes()
	routes1[0].Pattern = "/modified"

	routes2 := server.Routes()
	if routes2[0].Pattern != "/users" {
		t.Errorf("Routes() did not return a copy, got %q", routes2[0].Pattern)
	}
}

func TestParsePattern_SimplePath(t *testing.T) {
	method, path := parsePattern("/users")
	if method != "" {
		t.Errorf("method = %q, want empty", method)
	}
	if path != "/users" {
		t.Errorf("path = %q, want %q", path, "/users")
	}
}

func TestParsePattern_WithMethod(t *testing.T) {
	method, path := parsePattern("GET /users")
	if method != "GET" {
		t.Errorf("method = %q, want %q", method, "GET")
	}
	if path != "/users" {
		t.Errorf("path = %q, want %q", path, "/users")
	}
}

func TestParsePattern_PostMethod(t *testing.T) {
	method, path := parsePattern("POST /users")
	if method != "POST" {
		t.Errorf("method = %q, want %q", method, "POST")
	}
	if path != "/users" {
		t.Errorf("path = %q, want %q", path, "/users")
	}
}

func TestParsePattern_WithPathParams(t *testing.T) {
	method, path := parsePattern("GET /users/{id}")
	if method != "GET" {
		t.Errorf("method = %q, want %q", method, "GET")
	}
	if path != "/users/{id}" {
		t.Errorf("path = %q, want %q", path, "/users/{id}")
	}
}

func TestParsePattern_MultipleSpaces(t *testing.T) {
	method, path := parsePattern("DELETE /users/123")
	if method != "DELETE" {
		t.Errorf("method = %q, want %q", method, "DELETE")
	}
	if path != "/users/123" {
		t.Errorf("path = %q, want %q", path, "/users/123")
	}
}
