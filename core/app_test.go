package core

import (
	"context"
	"reflect"
	"testing"
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
