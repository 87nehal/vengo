//go:build !windows

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func notifyShutdown(ctx context.Context) context.Context {
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	return ctx
}
