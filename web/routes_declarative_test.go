package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/87nehal/vengo/autowire"
	"github.com/87nehal/vengo/core"
)

type testAPI struct {
	value string
}

func newTestAPI() *testAPI {
	return &testAPI{value: "api-ready"}
}

func (a *testAPI) HandleGet(w http.ResponseWriter, r *http.Request) error {
	return OK(w, map[string]string{"api": a.value})
}

func (a *testAPI) Routes() DeclarativeGroup {
	return Routes("/test-api",
		GET("/status", a.HandleGet),
	)
}

func TestDeclarativeRouting_AutoRegistration(t *testing.T) {
	// 1. Register testAPI constructor in autowire
	autowire.Register(newTestAPI)

	// 2. Configure app with web server
	server := New(":0")
	app := core.New("test-app", server)

	// Configure app (which registers autowired providers)
	if err := app.Configure(); err != nil {
		t.Fatalf("failed to configure app: %v", err)
	}

	// 3. Start the app (which runs Server.Start, resolving and registering routes)
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("failed to start app: %v", err)
	}
	defer func() { _ = app.Stop(ctx) }()

	// Verify routes are registered
	routes := server.Routes()
	found := false
	for _, r := range routes {
		if r.Pattern == "/test-api/status" && r.Method == "GET" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected route GET /test-api/status to be registered, registered routes: %+v", routes)
	}

	// 4. Test request execution
	req := httptest.NewRequest("GET", "/test-api/status", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	if !reflect.DeepEqual(rr.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected json content-type, got %q", rr.Header().Get("Content-Type"))
	}
}
