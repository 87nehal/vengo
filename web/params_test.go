package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathParams(t *testing.T) {
	mux := http.NewServeMux()
	server := New(":0")
	server.mux = mux

	server.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := PathInt(r, "id")
		if err != nil {
			err.ServeHTTP(w, r)
			return
		}
		_ = WriteJSON(w, http.StatusOK, map[string]int{"id": id})
	})

	server.HandleFunc("GET /users/{id}/range", func(w http.ResponseWriter, r *http.Request) {
		id, err := PathIntRange(r, "id", 1, 10)
		if err != nil {
			err.ServeHTTP(w, r)
			return
		}
		_ = WriteJSON(w, http.StatusOK, map[string]int{"id": id})
	})

	server.HandleFunc("GET /names/{name}", func(w http.ResponseWriter, r *http.Request) {
		name, err := PathString(r, "name")
		if err != nil {
			err.ServeHTTP(w, r)
			return
		}
		_ = WriteJSON(w, http.StatusOK, map[string]string{"name": name})
	})

	// Test PathInt success
	req := httptest.NewRequest("GET", "/users/42", nil)
	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Test PathInt failure (not int)
	req = httptest.NewRequest("GET", "/users/abc", nil)
	rr = httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	// Test PathIntRange success
	req = httptest.NewRequest("GET", "/users/5/range", nil)
	rr = httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Test PathIntRange failure (out of range)
	req = httptest.NewRequest("GET", "/users/15/range", nil)
	rr = httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	// Test PathString success
	req = httptest.NewRequest("GET", "/names/bob", nil)
	rr = httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}
