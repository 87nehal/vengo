package testutil_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/testutil"
	"github.com/87nehal/vengo/web"
)

type DummyService struct {
	Value string
}

type Greeter interface {
	Greet() string
}

type RealGreeter struct{}

func (g *RealGreeter) Greet() string {
	return "Hello from RealGreeter"
}

type MockGreeter struct{}

func (g *MockGreeter) Greet() string {
	return "Hello from MockGreeter"
}

type Consumer struct {
	Greeter Greeter `inject:""`
}

func TestAppLifecycleAndOverrides(t *testing.T) {
	srv := &DummyService{Value: "original"}
	app := testutil.NewApp(t)

	err := app.Register("dummy", srv)
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}

	mockSrv := &DummyService{Value: "mocked"}
	app.OverrideService("dummy", mockSrv)

	resolved, err := core.Get[*DummyService](app.App, "dummy")
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if resolved.Value != "mocked" {
		t.Errorf("expected value 'mocked', got %q", resolved.Value)
	}
}

func TestProviderOverrides(t *testing.T) {
	app := testutil.NewApp(t)

	err := core.Provide(app.App, func() Greeter {
		return &RealGreeter{}
	})
	if err != nil {
		t.Fatalf("failed to provide RealGreeter: %v", err)
	}

	err = core.Provide(app.App, func() *Consumer {
		return &Consumer{}
	})
	if err != nil {
		t.Fatalf("failed to provide Consumer: %v", err)
	}

	err = app.OverrideProvider(func() Greeter {
		return &MockGreeter{}
	})
	if err != nil {
		t.Fatalf("failed to override provider: %v", err)
	}

	consumer, err := core.Resolve[*Consumer](app.App)
	if err != nil {
		t.Fatalf("failed to resolve Consumer: %v", err)
	}

	greeting := consumer.Greeter.Greet()
	if greeting != "Hello from MockGreeter" {
		t.Errorf("expected greeting 'Hello from MockGreeter', got %q", greeting)
	}
}

func TestHTTPServerAndHelpers(t *testing.T) {
	server := web.New(":9090")
	server.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	})
	server.HandleFunc("POST /json", func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]string
		if err := web.BindJSON(r, &reqBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		web.WriteJSON(w, http.StatusOK, map[string]string{"received": reqBody["input"]})
	})

	app := testutil.NewApp(t, server)
	app.Start()

	url := app.URL("/hello")
	if strings.Contains(url, ":9090") {
		t.Errorf("expected URL to use ephemeral port, got %q", url)
	}

	resp := app.Get("/hello").ExpectStatus(http.StatusOK).ExpectContains("hello world")
	body := resp.BodyString()
	if body != "hello world" {
		t.Errorf("expected body 'hello world', got %q", body)
	}

	postBody := strings.NewReader(`{"input": "vengo"}`)
	postResp := app.Post("/json", "application/json", postBody).ExpectStatus(http.StatusOK)
	var res map[string]string
	postResp.ExpectJSON(&res)
	if res["received"] != "vengo" {
		t.Errorf("expected received 'vengo', got %q", res["received"])
	}
}

func TestSetupConfig(t *testing.T) {
	tomlContent := `
[app]
name = "test-config-app"
[server]
port = 8888
`
	testutil.SetupConfig(t, tomlContent)

	cfg, err := config.LoadDefaults(context.Background(), "")
	if err != nil {
		t.Fatalf("failed to load defaults: %v", err)
	}

	appName, ok := cfg.Get("app.name")
	if !ok || appName != "test-config-app" {
		t.Errorf("expected app.name to be 'test-config-app', got %q (found=%t)", appName, ok)
	}

	serverPort, ok := cfg.Get("server.port")
	if !ok || serverPort != "8888" {
		t.Errorf("expected server.port to be '8888', got %q (found=%t)", serverPort, ok)
	}
}
