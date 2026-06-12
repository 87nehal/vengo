package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseHelpers(t *testing.T) {
	// 1. Test OK
	rec := httptest.NewRecorder()
	err := OK(rec, map[string]string{"message": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("OK: expected 200, got %d", rec.Code)
	}
	var res map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res["message"] != "hello" {
		t.Errorf("OK: expected 'hello', got %q", res["message"])
	}

	// 2. Test Created
	rec = httptest.NewRecorder()
	err = Created(rec, map[string]string{"id": "123"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Created: expected 201, got %d", rec.Code)
	}

	// 3. Test NoContent
	rec = httptest.NewRecorder()
	err = NoContent(rec)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("NoContent: expected 204, got %d", rec.Code)
	}

	// 4. Test Text
	rec = httptest.NewRecorder()
	err = Text(rec, http.StatusBadRequest, "bad request body")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Text: expected 400, got %d", rec.Code)
	}
	if rec.Body.String() != "bad request body" {
		t.Errorf("Text: expected 'bad request body', got %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("Text: unexpected content type: %q", rec.Header().Get("Content-Type"))
	}
}
