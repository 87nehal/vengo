package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupPrefix(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "users" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "users")
	}
}

func TestGroupWithoutPrefix404(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGroupMiddleware(t *testing.T) {
	server := New(":0")

	groupCalled := false
	apiGroup := server.Group("/api", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groupCalled = true
			w.Header().Set("X-Group", "test")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if !groupCalled {
		t.Error("group middleware was not called")
	}

	if rec.Header().Get("X-Group") != "test" {
		t.Errorf("header X-Group = %q, want %q", rec.Header().Get("X-Group"), "test")
	}
}

func TestGroupInheritsServerMiddleware(t *testing.T) {
	server := New(":0")

	serverCalled := false
	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverCalled = true
			w.Header().Set("X-Server", "global")
			next.ServeHTTP(w, r)
		})
	})

	groupCalled := false
	apiGroup := server.Group("/api", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			groupCalled = true
			w.Header().Set("X-Group", "api")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if !serverCalled {
		t.Error("server middleware was not called")
	}

	if !groupCalled {
		t.Error("group middleware was not called")
	}

	if rec.Header().Get("X-Server") != "global" {
		t.Errorf("header X-Server = %q, want %q", rec.Header().Get("X-Server"), "global")
	}

	if rec.Header().Get("X-Group") != "api" {
		t.Errorf("header X-Group = %q, want %q", rec.Header().Get("X-Group"), "api")
	}
}

func TestGroupUseMiddleware(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	middlewareCalled := false
	apiGroup.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Added", "via-use")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("middleware added via Use() was not called")
	}

	if rec.Header().Get("X-Added") != "via-use" {
		t.Errorf("header X-Added = %q, want %q", rec.Header().Get("X-Added"), "via-use")
	}
}

func TestMultipleGroups(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	apiGroup.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("api-users"))
	})

	adminGroup := server.Group("/admin")
	adminGroup.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("admin-dashboard"))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/api/users", "api-users"},
		{"/admin/dashboard", "admin-dashboard"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want %d", tt.path, rec.Code, http.StatusOK)
		}

		if rec.Body.String() != tt.want {
			t.Errorf("GET %s: body = %q, want %q", tt.path, rec.Body.String(), tt.want)
		}
	}
}

func TestGroupMethodPattern(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	apiGroup.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "users" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "users")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/users", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGroupHandle(t *testing.T) {
	server := New(":0")

	apiGroup := server.Group("/api")
	apiGroup.Handle("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("users"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "users" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "users")
	}
}

func TestGroupMiddlewareOrder(t *testing.T) {
	server := New(":0")

	var order []string

	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "server")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup := server.Group("/api", func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "group1")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "group2")
			next.ServeHTTP(w, r)
		})
	})

	apiGroup.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	expected := []string{"server", "group1", "group2", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d", len(order), len(expected))
	}

	for i, v := range order {
		if v != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, v, expected[i])
		}
	}
}
