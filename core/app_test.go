package core

import (
	"context"
	"reflect"
	"testing"

	"github.com/87nehal/vengo/autowire"
)

type testModule struct {
	events *[]string
}

func (m testModule) Name() string {
	return "test"
}

func (m testModule) Configure(app *App) error {
	*m.events = append(*m.events, "configure")
	if err := app.Register("message", "hello"); err != nil {
		return err
	}
	app.RegisterHook(Hook{
		Name: "test-hook",
		Start: func(context.Context) error {
			*m.events = append(*m.events, "start")
			return nil
		},
		Stop: func(context.Context) error {
			*m.events = append(*m.events, "stop")
			return nil
		},
	})
	return nil
}

func TestAppLifecycle(t *testing.T) {
	events := []string{}
	app := New("test-app", testModule{events: &events})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("stop app: %v", err)
	}

	want := []string{"configure", "start", "stop"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestTypedGet(t *testing.T) {
	app := New("test")
	if err := app.Register("answer", 42); err != nil {
		t.Fatalf("register service: %v", err)
	}

	answer, err := Get[int](app, "answer")
	if err != nil {
		t.Fatalf("get typed service: %v", err)
	}
	if answer != 42 {
		t.Fatalf("answer = %d, want 42", answer)
	}
}

func TestDuplicateServiceFails(t *testing.T) {
	app := New("test")
	if err := app.Register("answer", 42); err != nil {
		t.Fatalf("register service: %v", err)
	}
	if err := app.Register("answer", 43); err == nil {
		t.Fatal("expected duplicate service registration to fail")
	}
}

type depA struct {
	Value string
}

func newDepA() *depA {
	return &depA{Value: "depA-value"}
}

type depB struct {
	DepA *depA `inject:""`
}

func newDepB() *depB {
	return &depB{}
}

func TestAutowireDiscovery(t *testing.T) {
	autowire.Register(newDepA)
	autowire.Register(newDepB)

	app := New("test-autowire")
	if err := app.Configure(); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	b, err := Resolve[*depB](app)
	if err != nil {
		t.Fatalf("resolve depB failed: %v", err)
	}

	if b.DepA == nil {
		t.Fatal("depB.DepA is nil (injection failed)")
	}
	if b.DepA.Value != "depA-value" {
		t.Errorf("expected 'depA-value', got %q", b.DepA.Value)
	}
}

