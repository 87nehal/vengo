package web

import (
	"fmt"
	"net/http"
)

// OK writes a 200 OK JSON response.
func OK(w http.ResponseWriter, body any) error {
	return WriteJSON(w, http.StatusOK, body)
}

// Created writes a 201 Created JSON response.
func Created(w http.ResponseWriter, body any) error {
	return WriteJSON(w, http.StatusCreated, body)
}

// Text writes a plain text response with the given status code.
func Text(w http.ResponseWriter, status int, message string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := fmt.Fprint(w, message)
	return err
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, body any) error {
	return WriteJSON(w, status, body)
}
