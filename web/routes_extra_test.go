package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRoutesJSON_Empty(t *testing.T) {
	server := New(":0")
	data, err := server.RoutesJSON()
	if err != nil {
		t.Fatalf("RoutesJSON: %v", err)
	}
	var got []map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty array, got %v", got)
	}
}

func TestRoutesJSON_MultipleRoutes(t *testing.T) {
	server := New(":0")
	server.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {})

	data, err := server.RoutesJSON()
	if err != nil {
		t.Fatalf("RoutesJSON: %v", err)
	}
	var got []map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(got))
	}
}

func TestFormatRoutes_Empty(t *testing.T) {
	var buf bytes.Buffer
	server := New(":0")
	server.FormatRoutes(&buf)
	if !strings.Contains(buf.String(), "no routes registered") {
		t.Fatalf("output = %q, want no routes message", buf.String())
	}
}

func TestFormatRoutes_SortedOutput(t *testing.T) {
	var buf bytes.Buffer
	server := New(":0")
	server.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {})
	server.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {})
	server.FormatRoutes(&buf)

	output := buf.String()
	if !strings.Contains(output, "Registered Routes") {
		t.Fatalf("missing header: %s", output)
	}
	healthIdx := strings.Index(output, "/health")
	usersIdx := strings.Index(output, "/users")
	if healthIdx == -1 || usersIdx == -1 || healthIdx > usersIdx {
		t.Fatalf("expected sorted output, got: %s", output)
	}
	if !strings.Contains(output, "GET") || !strings.Contains(output, "POST") {
		t.Fatalf("expected methods in output, got: %s", output)
	}
}
