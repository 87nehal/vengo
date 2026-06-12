package testutil

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/87nehal/vengo/core"
	"github.com/87nehal/vengo/web"
)

// NewTestServer starts the app if not already started, registers its shutdown in t.Cleanup,
// and returns a running httptest.Server wrapping the registered web.Server's handler.
func NewTestServer(t testing.TB, app *core.App) *httptest.Server {
	if err := app.Configure(); err != nil {
		t.Fatalf("failed to configure app: %v", err)
	}

	srvVal, exists := app.Get("web.server")
	if !exists {
		t.Fatalf("web.server is not registered in the app")
	}
	srv, ok := srvVal.(*web.Server)
	if !ok {
		t.Fatalf("web.server service is not of type *web.Server")
	}

	srv.SetAddr("127.0.0.1:0")

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("start test server app: %v", err)
	}

	t.Cleanup(func() {
		_ = app.Stop(context.Background())
	})

	testServer := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		testServer.Close()
	})

	return testServer
}
