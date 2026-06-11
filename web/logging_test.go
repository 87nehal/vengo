package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestLogger_Basic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server := New(":0")
	server.Use(RequestLogger(logger))
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "response" {
		t.Errorf("expected 'response', got '%s'", w.Body.String())
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Fatal("expected log output, got none")
	}

	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(logOutput), &logData); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}

	if logData["method"] != "GET" {
		t.Errorf("expected method 'GET', got '%v'", logData["method"])
	}
	if logData["path"] != "/test" {
		t.Errorf("expected path '/test', got '%v'", logData["path"])
	}
	if logData["status"].(float64) != 200 {
		t.Errorf("expected status 200, got %v", logData["status"])
	}
	if logData["bytes"].(float64) != 8 {
		t.Errorf("expected 8 bytes, got %v", logData["bytes"])
	}
	if logData["remote"] != "192.168.1.1:12345" {
		t.Errorf("expected remote '192.168.1.1:12345', got '%v'", logData["remote"])
	}
	if _, ok := logData["duration"]; !ok {
		t.Error("expected duration field in log")
	}
}

func TestRequestLogger_ErrorStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server := New(":0")
	server.Use(RequestLogger(logger))
	server.HandleFunc("/notfound", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	var logData map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logData); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}

	if logData["status"].(float64) != 404 {
		t.Errorf("expected status 404, got %v", logData["status"])
	}
}

func TestRequestLogger_NilLogger(t *testing.T) {
	server := New(":0")
	server.Use(RequestLogger(nil))
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestResponseRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &ResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

	rec.Write([]byte("hello"))
	rec.Write([]byte(" world"))

	if rec.bytes != 11 {
		t.Errorf("expected 11 bytes, got %d", rec.bytes)
	}

	rec.WriteHeader(http.StatusCreated)
	if rec.statusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.statusCode)
	}
}

func TestRequestLogger_Duration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	server := New(":0")
	server.Use(RequestLogger(logger))
	server.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	server.Handler().ServeHTTP(w, req)

	var logData map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logData); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}

	durationStr := logData["duration"].(float64)
	if durationStr < 0.01 {
		t.Errorf("expected duration >= 10ms, got '%v'", durationStr)
	}
}
