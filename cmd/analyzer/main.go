//go:build linux && amd64

package main

import (
	"context"
	"log"
	"oblak/internal/analyzer/message"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	defer cancel()
	if err := message.StartListener(ctx); err != nil {
		log.Fatalf("Failed to start service: %v", err)

	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down gracefully...", sig)

	cancel()

	wg.Wait()

	log.Println("Service stopped. Exiting.")

}
