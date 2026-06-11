package testutil

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

// App is a testing harness wrapper around core.App.
type App struct {
	*core.App
	t *testing.T
}

// NewApp creates a new test instance of the Vengo application.
// It automatically intercepts any registered *web.Server and configures it to listen on an ephemeral port.
// It also registers a cleanup function via t.Cleanup to automatically stop the app when the test finishes.
func NewApp(t *testing.T, modules ...core.Module) *App {
	// Intercept web.Server if present in modules
	for _, m := range modules {
		if srv, ok := m.(*web.Server); ok {
			srv.SetAddr("127.0.0.1:0")
		}
	}

	coreApp := core.New("test-app", modules...)
	app := &App{
		App: coreApp,
		t:   t,
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Stop(ctx); err != nil {
			t.Logf("Warning: failed to stop test application: %v", err)
		}
	})

	return app
}

// Start configures the application and runs its lifecycle start hooks.
// It also ensures any transitively registered web server uses an ephemeral port.
func (a *App) Start() {
	if err := a.Configure(); err != nil {
		a.t.Fatalf("failed to configure test application: %v", err)
	}

	if srv, err := core.Get[*web.Server](a.App, "web.server"); err == nil {
		addr := srv.Addr()
		if !strings.HasSuffix(addr, ":0") {
			srv.SetAddr("127.0.0.1:0")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.App.Start(ctx); err != nil {
		a.t.Fatalf("failed to start test application: %v", err)
	}
}

// URL constructs a full HTTP URL using the ephemeral port of the running web.Server.
func (a *App) URL(path string) string {
	srv, err := core.Get[*web.Server](a.App, "web.server")
	if err != nil {
		a.t.Fatalf("failed to retrieve web.Server from app: %v", err)
	}

	addr := srv.Addr()
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return "http://" + addr + path
}

// Response is a wrapper around http.Response providing utility methods for testing assertions.
type Response struct {
	*http.Response
	t *testing.T
}

// Status returns the HTTP status code of the response.
func (r *Response) Status() int {
	return r.StatusCode
}

// BodyString reads and returns the response body as a string.
func (r *Response) BodyString() string {
	r.t.Helper()
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		r.t.Fatalf("failed to read response body: %v", err)
	}
	return string(body)
}

// JSON decodes the JSON response body into the target structure.
func (r *Response) JSON(target any) {
	r.t.Helper()
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		r.t.Fatalf("failed to decode response JSON: %v", err)
	}
}

// Request executes an HTTP request against the running application.
func (a *App) Request(method, path string, body io.Reader) *Response {
	a.t.Helper()
	req, err := http.NewRequest(method, a.URL(path), body)
	if err != nil {
		a.t.Fatalf("failed to create request %s %s: %v", method, path, err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		a.t.Fatalf("failed to execute request %s %s: %v", method, path, err)
	}

	return &Response{Response: resp, t: a.t}
}

// Get executes a GET request against the running application.
func (a *App) Get(path string) *Response {
	a.t.Helper()
	return a.Request("GET", path, nil)
}

// Post executes a POST request with the specified content type and body.
func (a *App) Post(path string, contentType string, body io.Reader) *Response {
	a.t.Helper()
	req, err := http.NewRequest("POST", a.URL(path), body)
	if err != nil {
		a.t.Fatalf("failed to create POST request: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		a.t.Fatalf("failed to execute POST request: %v", err)
	}

	return &Response{Response: resp, t: a.t}
}

// SetupConfig writes a temporary configuration file with the given content,
// sets the VENGO_CONFIG_DIR environment variable to point to it,
// and registers a cleanup handler to restore the environment variable.
func SetupConfig(t *testing.T, tomlContent string) string {
	tempDir := t.TempDir()

	configPath := filepath.Join(tempDir, "application.toml")
	err := os.WriteFile(configPath, []byte(tomlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temporary config file: %v", err)
	}

	t.Setenv("VENGO_CONFIG_DIR", tempDir)
	return configPath
}
