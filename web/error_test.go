package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorServeHTTP(t *testing.T) {
	err := NewError(http.StatusBadRequest, "invalid input")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	err.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"error":"invalid input"`) {
		t.Errorf("body = %q, want to contain error message", body)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
}

func TestErrorError(t *testing.T) {
	err := NewError(http.StatusNotFound, "not found")
	if err.Error() != "not found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "not found")
	}

	cause := errors.New("connection refused")
	wrappedErr := WrapError(http.StatusInternalServerError, "service unavailable", cause)
	if !strings.Contains(wrappedErr.Error(), "service unavailable") {
		t.Errorf("Error() = %q, want to contain message", wrappedErr.Error())
	}
	if !strings.Contains(wrappedErr.Error(), "connection refused") {
		t.Errorf("Error() = %q, want to contain cause", wrappedErr.Error())
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := WrapError(http.StatusInternalServerError, "wrapped", cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	simpleErr := NewError(http.StatusBadRequest, "simple")
	if errors.Unwrap(simpleErr) != nil {
		t.Errorf("Unwrap() = %v, want nil for simple error", errors.Unwrap(simpleErr))
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		wantCode int
	}{
		{"bad request", BadRequest("invalid"), http.StatusBadRequest},
		{"unauthorized", Unauthorized("auth required"), http.StatusUnauthorized},
		{"forbidden", Forbidden("access denied"), http.StatusForbidden},
		{"not found", NotFound("missing"), http.StatusNotFound},
		{"internal", InternalServerError("server error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", tt.err.Code, tt.wantCode)
			}

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			tt.err.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("ServeHTTP status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

func TestErrorAsHandler(t *testing.T) {
	server := New(":0")
	server.Handle("/error", NotFound("resource not found"))

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"error":"resource not found"`) {
		t.Errorf("body = %q, want to contain error message", body)
	}
}

func TestErrorJSON(t *testing.T) {
	err := NewError(http.StatusBadRequest, "validation failed")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	err.ServeHTTP(rec, req)

	body := rec.Body.String()
	expected := `{"error":"validation failed"}`
	if !strings.Contains(body, expected) {
		t.Errorf("body = %q, want to contain %q", body, expected)
	}
}
