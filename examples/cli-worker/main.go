package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/87nehal/vengo/config"
	"github.com/87nehal/vengo/core"
)

// TaskProcessor performs background work.
type TaskProcessor interface {
	Process(ctx context.Context) error
}

// LoggingProcessor is the default processor that logs work ticks.
type LoggingProcessor struct{}

func (p *LoggingProcessor) Process(ctx context.Context) error {
	log.Println("Processing tick task...")
	return nil
}

// Worker is a background task processor that implements core.Module.
type Worker struct {
	Processor TaskProcessor
	Interval  time.Duration
	stopChan  chan struct{}
}

// NewWorker initializes a new Worker module.
func NewWorker(interval time.Duration) *Worker {
	return &Worker{
		Interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Name returns the module name.
func (w *Worker) Name() string {
	return "worker"
}

// Configure registers the worker service and lifecycle hooks on the app.
func (w *Worker) Configure(app *core.App) error {
	processor, err := core.Resolve[TaskProcessor](app)
	if err != nil {
		return err
	}
	w.Processor = processor

	if err := app.Register("worker", w); err != nil {
		return err
	}
	app.RegisterHook(core.Hook{
		Name:  "worker",
		Start: w.Start,
		Stop:  w.Stop,
	})
	return nil
}

// Start launches the worker execution in a separate goroutine.
func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Worker starting with interval: %v", w.Interval)
	go w.run()
	return nil
}

// Stop closes execution loops.
func (w *Worker) Stop(ctx context.Context) error {
	log.Println("Worker shutting down...")
	close(w.stopChan)
	return nil
}

func (w *Worker) run() {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), w.Interval)
			if err := w.Processor.Process(ctx); err != nil {
				log.Printf("Error processing task: %v", err)
			}
			cancel()
		case <-w.stopChan:
			log.Println("Worker stopped run loop.")
			return
		}
	}
}

// AppConfig binds application settings.
type AppConfig struct {
	Interval time.Duration `config:"worker.interval" default:"1s"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadDefaults(ctx, config.ActiveProfile())
	if err != nil {
		log.Fatal(err)
	}

	var appCfg AppConfig
	if err := config.Bind(cfg, &appCfg); err != nil {
		log.Fatal(err)
	}

	worker := NewWorker(appCfg.Interval)
	app := core.New("cli-worker", worker)
	app.SetConfig(cfg)

	err = core.Provide(app, func() TaskProcessor {
		return &LoggingProcessor{}
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("CLI Worker is running. Press Ctrl+C to stop.")
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(shutdownCtx); err != nil {
		log.Fatal(err)
	}
	log.Println("CLI Worker shutdown completed.")
}
