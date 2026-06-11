package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/87nehal/vengo/web"
)

func BenchmarkHTTPThroughput(b *testing.B) {
	server := web.New(":0")
	server.HandleFunc("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	handler := server.Handler()
	req := httptest.NewRequest("GET", "/api/hello", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
