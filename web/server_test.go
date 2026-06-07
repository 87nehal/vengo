package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleFunc(t *testing.T) {
	server := New(":0")
	server.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	})

	request := httptest.NewRequest(http.MethodGet, "/hello", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", response.Body.String())
	}
}
