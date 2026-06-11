package main

import (
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/testutil"
	"github.com/87nehal/vengo/web"
)

// MockMessageService is a mock implementation of MessageService for testing.
type MockMessageService struct{}

func (m *MockMessageService) Greet(name string) string {
	return "Mocked Hello " + name
}

func TestGreetingsAPI(t *testing.T) {
	testutil.SetupConfig(t, `
[greeting]
prefix = "Welcome"
`)

	server := web.New(":0")
	app := testutil.NewApp(t, server)

	err := core.Provide(app.App, func() MessageService {
		return &SimpleMessageService{Prefix: "Welcome"}
	})
	if err != nil {
		t.Fatalf("failed to provide service: %v", err)
	}

	err = core.Provide(app.App, func() *UserHandler {
		return &UserHandler{}
	})
	if err != nil {
		t.Fatalf("failed to provide handler: %v", err)
	}

	err = app.OverrideProvider(func() MessageService {
		return &MockMessageService{}
	})
	if err != nil {
		t.Fatalf("failed to override provider: %v", err)
	}

	handler, err := core.Resolve[*UserHandler](app.App)
	if err != nil {
		t.Fatalf("failed to resolve user handler: %v", err)
	}
	handler.Register(server)

	app.Start()

	resp := app.Get("/greet?name=Alice")
	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}

	var data map[string]string
	resp.JSON(&data)
	if data["message"] != "Mocked Hello Alice" {
		t.Errorf("expected mocked greeting, got %q", data["message"])
	}
}
