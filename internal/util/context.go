package util

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func SigtermCancelledContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(sigChan)
	}()

	return ctx, cancel
}
