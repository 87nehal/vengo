# Vengo Security Module

The `security` module integrates comprehensive security features into a Vengo web application. It includes standard security middlewares, session management, token validation, and configuration-driven security setup.

## Features

- **CORS Middleware**: Implements standard Cross-Origin Resource Sharing.
- **Secure Headers Middleware**: Injects common security-related response headers (`X-Frame-Options`, `X-Content-Type-Options`, CSP, HSTS, etc.).
- **CSRF Protection**: Provides double-submit cookie validation for state-changing HTTP requests.
- **Session Middleware**: Supports pluggable session stores. Includes out-of-the-box `CookieSessionStore` (cryptographically signed cookies) and `InMemorySessionStore`.
- **JWT Middleware**: Self-contained HS256 JWT verifier with claims parsing.
- **API Key Middleware**: Simple header or query parameter validation.
- **Auth & Role Verification**: Injects User information into request context and exposes `RequireRole` authorization middleware.

---

## Configuration

Add the following settings to your `application.toml` to configure security:

```toml
[security.api-key]
enabled = false
keys = "my-api-key-1,my-api-key-2"
header = "X-API-Key"

[security.jwt]
enabled = false
secret = "your-jwt-signing-secret"
issuer = "your-jwt-issuer"

[security.session]
enabled = false
secret = "your-session-cookie-secret"
cookie = "vengo_session"

[security.headers]
enabled = false
frame-options = "DENY"
content-type-options = "nosniff"
xss-protection = "1; mode=block"
referrer-policy = "strict-origin-when-cross-origin"
content-security-policy = ""
hsts = "max-age=63072000; includeSubDomains"

[security.cors]
enabled = false
allowed-origins = "*"
allowed-methods = "GET,POST,PUT,DELETE,OPTIONS"
allowed-headers = "Content-Type,Authorization"
allow-credentials = false
max-age = 1800

[security.csrf]
enabled = false
cookie-name = "XSRF-TOKEN"
header-name = "X-XSRF-TOKEN"
```

---

## Pluggable Session Store

Vengo defines the `SessionStore` interface:

```go
type SessionStore interface {
    Get(r *http.Request, name string) (*Session, error)
    Save(r *http.Request, w http.ResponseWriter, session *Session) error
}
```

It registers a default `CookieSessionStore` (which signs cookie payloads using HMAC-SHA256). You can override this behavior by registering your own implementation of `SessionStore` in the Vengo application context before configuring the `security` module.

---

## Middleware Pipeline Order

When registered, the security module registers its enabled middlewares in the following order:

1. **CORS** (Outermost - responds early to preflights)
2. **Secure Headers**
3. **CSRF**
4. **Session**
5. **JWT / API Key / Auth** (Innermost - runs authentication checks)

---

## Quick Start Example

Here is how to run a web server with Vengo's security module:

```go
package main

import (
    "github.com/87nehal/vengo/config"
    "github.com/87nehal/vengo/core"
    "github.com/87nehal/vengo/security"
    "github.com/87nehal/vengo/web"
)

func main() {
    // Load config
    cfg, _ := config.LoadDefaults(context.Background(), "")

    // Create server and register routes
    server := web.New(":8080")
    
    // Public endpoint
    server.HandleFunc("GET /public", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, Guest!"))
    })

    // Private endpoint (requires admin role)
    privateGroup := server.Group("/admin", security.AuthMiddleware(), security.RequireRole("admin"))
    privateGroup.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
        user, _ := security.UserFromContext(r.Context())
        w.Write([]byte("Welcome back " + user.ID))
    })

    // Create App and register security module
    app := core.New("my-app", server, security.New())
    app.SetConfig(cfg)

    app.Start(context.Background())
}
```
