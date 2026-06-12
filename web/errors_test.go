package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorMiddleware_WithoutMappers(t *testing.T) {
	mux := http.NewServeMux()
	server := New(":0")
	server.mux = mux // use custom test mux

	// Register test handler returning Web Error
	server.HandleError("GET /web-err", func(w http.ResponseWriter, r *http.Request) error {
		return NewError(http.StatusForbidden, "access denied")
	})

	// Register test handler returning Standard Error
	server.HandleError("GET /std-err", func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("something went wrong")
	})

	handler := applyMiddleware(server.mux, []Middleware{ErrorMiddleware()})

	// Test 1: Web Error
	req := httptest.NewRequest("GET", "/web-err", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}

	var respBody map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil {
		t.Fatal(err)
	}
	if respBody["error"] != "access denied" {
		t.Errorf("expected 'access denied', got %q", respBody["error"])
	}

	// Test 2: Standard Error
	req = httptest.NewRequest("GET", "/std-err", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil {
		t.Fatal(err)
	}
	if respBody["error"] != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %q", respBody["error"])
	}
}

func TestErrorMiddleware_WithMapper(t *testing.T) {
	mux := http.NewServeMux()
	server := New(":0")
	server.mux = mux

	customErr := errors.New("custom error")

	server.HandleError("GET /custom", func(w http.ResponseWriter, r *http.Request) error {
		return customErr
	})

	customMapper := func(err error) (int, string, bool) {
		if errors.Is(err, customErr) {
			return http.StatusBadRequest, "mapped custom error", true
		}
		return 0, "", false
	}

	handler := applyMiddleware(server.mux, []Middleware{ErrorMiddleware(customMapper)})

	req := httptest.NewRequest("GET", "/custom", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var respBody map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil {
		t.Fatal(err)
	}
	if respBody["error"] != "mapped custom error" {
		t.Errorf("expected 'mapped custom error', got %q", respBody["error"])
	}
}
