package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u *testUser) Valid() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.Age < 0 || u.Age > 150 {
		return errors.New("age must be between 0 and 150")
	}
	return nil
}

func TestBindJSON_Success(t *testing.T) {
	body := `{"name":"John","email":"john@example.com","age":30}`
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var user testUser
	if err := BindJSON(r, &user); err != nil {
		t.Fatalf("BindJSON() error = %v", err)
	}

	if user.Name != "John" {
		t.Errorf("Name = %q, want %q", user.Name, "John")
	}
	if user.Email != "john@example.com" {
		t.Errorf("Email = %q, want %q", user.Email, "john@example.com")
	}
	if user.Age != 30 {
		t.Errorf("Age = %d, want %d", user.Age, 30)
	}
}

func TestBindJSON_InvalidJSON(t *testing.T) {
	body := `{invalid json}`
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var user testUser
	err := BindJSON(r, &user)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
}

func TestBindJSON_EmptyBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/users", nil)
	r.Body = nil

	var user testUser
	err := BindJSON(r, &user)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
}

func TestBindJSON_ValidationFailure(t *testing.T) {
	body := `{"name":"","email":"john@example.com","age":30}`
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var user testUser
	err := BindJSON(r, &user)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error = %q, want to contain 'name is required'", err.Error())
	}
}

func TestBindJSON_UnknownFields(t *testing.T) {
	body := `{"name":"John","email":"john@example.com","age":30,"unknown":"field"}`
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var user testUser
	err := BindJSON(r, &user)
	if err != nil {
		t.Fatalf("expected BindJSON to succeed with unknown fields, got: %v", err)
	}
	if user.Name != "John" {
		t.Errorf("Name = %q, want John", user.Name)
	}
}

func TestBindJSONStrict_UnknownFields(t *testing.T) {
	body := `{"name":"John","email":"john@example.com","age":30,"unknown":"field"}`
	r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var user testUser
	err := BindJSONStrict(r, &user)
	if err == nil {
		t.Fatal("expected error for BindJSONStrict with unknown fields")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, http.StatusBadRequest)
	}
}

func TestBindJSON_WithoutValidator(t *testing.T) {
	type simple struct {
		Name string `json:"name"`
	}

	body := `{"name":"test"}`
	r := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))

	var s simple
	if err := BindJSON(r, &s); err != nil {
		t.Fatalf("BindJSON() error = %v", err)
	}

	if s.Name != "test" {
		t.Errorf("Name = %q, want %q", s.Name, "test")
	}
}

func TestBindJSON_IntegrationWithHandler(t *testing.T) {
	server := New(":0")

	server.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		var user testUser
		if err := BindJSON(r, &user); err != nil {
			err.ServeHTTP(w, r)
			return
		}
		WriteJSON(w, http.StatusCreated, user)
	})

	body := `{"name":"Jane","email":"jane@example.com","age":25}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	if !strings.Contains(rec.Body.String(), `"name":"Jane"`) {
		t.Errorf("body = %q, want to contain user data", rec.Body.String())
	}
}
