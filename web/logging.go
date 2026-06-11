package web

import (
	"log/slog"
	"net/http"
	"time"
)

type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *ResponseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func RequestLogger(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &ResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rec, r)

			duration := time.Since(start)

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.statusCode),
				slog.Duration("duration", duration),
				slog.Int("bytes", rec.bytes),
				slog.String("remote", r.RemoteAddr),
			)
		})
	}
}
