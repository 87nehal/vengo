package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/87nehal/vengo/core"
)

type testProcessor struct{}

func (p *testProcessor) Process(ctx context.Context) error {
	return nil
}

func TestWorkerConfigureResolvesProcessor(t *testing.T) {
	worker := NewWorker(time.Hour)
	app := core.New("cli-worker-test", worker)
	if err := core.Provide(app, func() TaskProcessor { return &testProcessor{} }); err != nil {
		t.Fatal(err)
	}

	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if worker.Processor == nil {
		t.Fatal("worker processor was not resolved")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerConfigureRequiresProcessorProvider(t *testing.T) {
	worker := NewWorker(time.Hour)
	app := core.New("cli-worker-test", worker)

	err := app.Start(context.Background())
	if err == nil {
		t.Fatal("expected missing processor provider error")
	}
	if !strings.Contains(err.Error(), "TaskProcessor") {
		t.Fatalf("error = %q, want missing TaskProcessor", err.Error())
	}
}
