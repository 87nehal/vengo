package web

import (
	"fmt"
	"net/http"
	"strconv"
)

// PathInt returns the route parameter as an integer, returning a BadRequest error if invalid.
func PathInt(r *http.Request, name string) (int, *Error) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, BadRequest(fmt.Sprintf("missing path parameter %s", name))
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, BadRequest(fmt.Sprintf("%s must be an integer", name))
	}
	return n, nil
}

// PathIntRange returns the route parameter as an integer within a range [min, max] inclusive.
func PathIntRange(r *http.Request, name string, min, max int) (int, *Error) {
	n, err := PathInt(r, name)
	if err != nil {
		return 0, err
	}
	if n < min || n > max {
		return 0, BadRequest(fmt.Sprintf("%s must be between %d and %d", name, min, max))
	}
	return n, nil
}

// PathString returns the route parameter as a string, returning a BadRequest error if empty.
func PathString(r *http.Request, name string) (string, *Error) {
	raw := r.PathValue(name)
	if raw == "" {
		return "", BadRequest(fmt.Sprintf("missing path parameter %s", name))
	}
	return raw, nil
}
