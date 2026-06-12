package web

import (
	"context"
	"errors"
	"net/http"
)

type errorKey struct{}

type errorHolder struct {
	err error
}

// ErrorHandlerFunc is an HTTP handler that can return an error.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ErrorMapper converts a Go error to an HTTP status and message.
type ErrorMapper func(err error) (code int, message string, handled bool)

// ErrorMiddleware catches errors returned by handlers and maps them to JSON.
func ErrorMiddleware(mappers ...ErrorMapper) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			holder := &errorHolder{}
			r = r.WithContext(context.WithValue(r.Context(), errorKey{}, holder))

			next.ServeHTTP(w, r)

			if holder.err != nil {
				handleError(w, r, holder.err, mappers)
			}
		})
	}
}

// ErrorHandler wraps an ErrorHandlerFunc to implement http.Handler.
func ErrorHandler(fn ErrorHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := fn(w, r)
		if err != nil {
			if holder, ok := r.Context().Value(errorKey{}).(*errorHolder); ok {
				holder.err = err
			} else {
				// Fallback if ErrorMiddleware is not registered
				var webErr *Error
				if errors.As(err, &webErr) {
					_ = WriteJSON(w, webErr.Code, map[string]any{"error": webErr.Message})
				} else {
					_ = WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				}
			}
		}
	})
}

func handleError(w http.ResponseWriter, r *http.Request, err error, mappers []ErrorMapper) {
	for _, mapper := range mappers {
		if code, msg, handled := mapper(err); handled {
			_ = WriteJSON(w, code, map[string]any{"error": msg})
			return
		}
	}

	var webErr *Error
	if errors.As(err, &webErr) {
		_ = WriteJSON(w, webErr.Code, map[string]any{"error": webErr.Message})
		return
	}

	_ = WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
}
