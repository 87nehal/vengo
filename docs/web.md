# Web

The `web` package provides an HTTP server module built on stdlib `net/http` (Go 1.22+), with middleware chains, route groups, structured errors, JSON binding, and a request logging middleware.

## Server

Create and register a server with the app:

```go
server := web.New(":8080")
app := core.New("myapp", server)
```

The server hooks into the app lifecycle automatically via `App.Start` / `App.Stop`. It listens on the configured address and shuts down gracefully.

### Methods

| Method | Description |
|--------|-------------|
| `Handle(pattern, handler)` | Register an `http.Handler` for a pattern |
| `HandleFunc(pattern, handler)` | Register a `func(http.ResponseWriter, *http.Request)` for a pattern |
| `Handler()` | Returns the handler with all middleware applied |
| `Use(middlewares...)` | Add server-level middleware |
| `Group(prefix, middlewares...)` | Create a route group with shared prefix and middleware |
| `Addr()` | Returns the listener address (useful when using `:0`) |
| `Routes()` | Returns a copy of all registered routes |

### Route Patterns

Patterns follow Go 1.22+ `net/http` conventions. Method-prefixed patterns are supported:

```go
server.HandleFunc("GET /users", listUsers)
server.HandleFunc("POST /users", createUser)
server.HandleFunc("GET /users/{id}", getUser)
```

## Middleware

A middleware is a `func(http.Handler) http.Handler`. Server-level middleware wraps the entire mux. Group-level middleware wraps only the group's routes.

```go
server.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Custom", "value")
        next.ServeHTTP(w, r)
    })
})
```

Middleware executes in registration order (first registered runs first). Group middleware runs after server middleware.

## Route Groups

Groups share a prefix and optional middleware:

```go
api := server.Group("/api/v1", authMiddleware)
api.HandleFunc("/users", listUsers)      // handles /api/v1/users
api.HandleFunc("/posts", listPosts)      // handles /api/v1/posts
```

Groups inherit server-level middleware automatically. Group middleware only applies to routes within the group.

```go
admin := server.Group("/admin", requireAdmin)
admin.HandleFunc("/dashboard", dashboard) // /admin/dashboard
```

## Structured Errors

`web.Error` provides HTTP error responses with JSON rendering:

```go
if user == nil {
    web.NotFound("user not found").ServeHTTP(w, r)
    return
}
```

### Error Constructors

| Function | Status Code |
|----------|-------------|
| `BadRequest(msg)` | 400 |
| `Unauthorized(msg)` | 401 |
| `Forbidden(msg)` | 403 |
| `NotFound(msg)` | 404 |
| `InternalServerError(msg)` | 500 |
| `NewError(code, msg)` | custom |
| `WrapError(code, msg, cause)` | custom with wrapped cause |

All errors implement `error`, `Unwrap()`, and `http.Handler`. The JSON response format is `{"error": "message"}`.

## JSON Request Binding

`web.BindJSON` decodes a JSON request body into a struct, rejecting unknown fields:

```go
func createUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := web.BindJSON(r, &req); err != nil {
        err.ServeHTTP(w, r)
        return
    }
    // use req...
}
```

### Validator Interface

If the target implements `web.Validator`, its `Valid() error` method is called automatically after decoding:

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateUserRequest) Valid() error {
    if r.Name == "" {
        return errors.New("name is required")
    }
    return nil
}
```

## JSON Response Helper

```go
web.WriteJSON(w, http.StatusOK, map[string]string{"message": "hello"})
```

Sets `Content-Type: application/json` and encodes the value.

## Request Logging

`web.RequestLogger` provides structured request logging via `log/slog`:

```go
server.Use(web.RequestLogger(logger))
```

Logs each request with: `method`, `path`, `status`, `duration`, `bytes`, `remote`. Falls back to `slog.Default()` if no logger is provided.

## Route Registry

`server.Routes()` returns a copy of all registered routes (pattern + method). Useful for diagnostics and CLI `vengo routes`:

```go
for _, route := range server.Routes() {
    fmt.Printf("%s %s\n", route.Method, route.Pattern)
}
```

### Serializing Routes

`server.RoutesJSON()` returns a JSON array of `{method, pattern}` objects, sorted by registration. `server.FormatRoutes(w)` writes a human-readable table to any `io.Writer`. These are used by the `vengo routes` command, which reads a `vengo-routes.json` produced by an app:

```go
data, _ := webServer.RoutesJSON()
os.WriteFile("vengo-routes.json", data, 0o644)
```

```bash
vengo routes
# Registered Routes:
# --------------------------------------------------
#   *       /health
#   GET     /users
#   POST    /users
```

## Integration With Actuator

The `actuator` package uses `web.Server` for all its endpoints. Register the server before actuator modules:

```go
server := web.New(":8080")
health := actuator.NewHealth()
info := actuator.NewInfo(actuator.WithVersion("1.0.0"))
app := core.New("myapp", server, health, info)
```
