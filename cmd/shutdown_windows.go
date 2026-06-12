//go:build windows

package cmd

import (
	"context"
	"os"
	"os/signal"
)

func notifyShutdown(ctx context.Context) context.Context {
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
	return ctx
}
