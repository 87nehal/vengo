# Testing Toolkit

Vengo provides a testing toolkit under the `testutil` package to make it easy to write unit and integration tests for Vengo applications. The toolkit focuses on in-process application testing, automatic port management, dependency overrides (for mocking), and configuration isolation.

## Key Features

- **In-Process Lifecycle**: Run a complete instance of your application inside the test process.
- **Automatic Cleanup**: Automatically stops the application and releases bound ports at the end of the test using Go's `t.Cleanup`.
- **Ephemeral Port Binding**: Automatically intercepts registered `web.Server` modules and binds them to `127.0.0.1:0` to prevent port collisions.
- **Dependency Overrides**: Swap out services or container providers with mocks or test doubles before the app starts.
- **HTTP Test Client**: Simple request helpers (`Get`, `Post`, `Request`) and response parsers (`Status`, `BodyString`, `JSON`).
- **Config Isolation**: Easily write temporary TOML configs for specific test scenarios using `SetupConfig`.

---

## Getting Started

To use the testing toolkit, import `github.com/87nehal/vengo/testutil`.

### 1. Creating a Test App

Use `testutil.NewApp(t, modules...)` to initialize a test harness. It returns a wrapped `*testutil.App` that manages the test lifecycle.

```go
func TestMyHandler(t *testing.T) {
    server := web.New(":8080") // will be overridden to :0
    app := testutil.NewApp(t, server)

    // Setup routes & providers...

    app.Start()

    // App is running on an ephemeral port here
}
```

### 2. Dependency Overrides

You can override registered services or constructor-based providers before starting the app.

#### Overriding Named Services
```go
mockDB := &MockDatabase{}
app.OverrideService("data.db", mockDB)
```

#### Overriding Container Providers
```go
err := app.OverrideProvider(func() MyService {
    return &MockService{}
})
```

### 3. Simulating HTTP Requests

Use the app's HTTP helpers to issue requests. The client automatically routes requests to the ephemeral port.

```go
resp := app.Get("/users/123")
if resp.Status() != http.StatusOK {
    t.Errorf("expected 200, got %d", resp.Status())
}

var user User
resp.JSON(&user)
if user.Name != "John Doe" {
	t.Errorf("unexpected name: %s", user.Name)
}
```

### 4. Configuration Setup

Write temporary configurations for tests without polluting the project root directory.

```go
testutil.SetupConfig(t, `
[app]
name = "test-run"
[database]
url = "sqlite::memory:"
`)
```

---

## Full Example

Below is a complete test illustrating the testing toolkit in action:

```go
package main_test

import (
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/testutil"
	"github.com/87nehal/vengo/web"
)

type Greeter interface {
	Greet(name string) string
}

type RealGreeter struct{}
func (g *RealGreeter) Greet(name string) string { return "Hello, " + name }

type MockGreeter struct{}
func (g *MockGreeter) Greet(name string) string { return "Mocked " + name }

type Handler struct {
	Greeter Greeter `inject:""`
}

func (h *Handler) Register(server *web.Server) {
	server.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		web.WriteJSON(w, 200, map[string]string{"msg": h.Greeter.Greet(name)})
	})
}

func TestHelloAPI(t *testing.T) {
	// Setup isolated configuration
	testutil.SetupConfig(t, `
[app]
name = "test-greeting"
`)

	server := web.New(":0")
	app := testutil.NewApp(t, server)

	// Register normal providers
	core.Provide(app.App, func() Greeter { return &RealGreeter{} })
	core.Provide(app.App, func() *Handler { return &Handler{} })

	// Override with a test double
	app.OverrideProvider(func() Greeter { return &MockGreeter{} })

	// Resolve and wire routes
	h, _ := core.Resolve[*Handler](app.App)
	h.Register(server)

	// Start app on ephemeral port
	app.Start()

	// Perform requests and verify assertions
	resp := app.Get("/hello?name=Bob")
	if resp.Status() != 200 {
		t.Fatalf("expected status 200, got %d", resp.Status())
	}

	var data map[string]string
	resp.JSON(&data)
	if data["msg"] != "Mocked Bob" {
		t.Errorf("expected 'Mocked Bob', got %q", data["msg"])
	}
}
```
