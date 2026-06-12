package cmd

import "context"

// NotifyShutdown returns a copy of parent context that is canceled when
// an OS interrupt signal (or SIGTERM on non-Windows) is received.
func NotifyShutdown(ctx context.Context) context.Context {
	return notifyShutdown(ctx)
}
