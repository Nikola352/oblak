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
	const quarantinePath = "/tmp/quarantine"
	if err := initializeQuarantineDirectory(quarantinePath); err != nil {
		// If we can't write or clean /tmp, our pipeline is broken, so we panic/fail early.
		log.Fatalf("[FATAL] Critical initialization error: %v", err)
	}

	// 2. Continue booting up services...
	log.Println("[MAIN] Initializing background storage and connection dependencies...")
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

func initializeQuarantineDirectory(path string) error {
	log.Printf("[INIT] Booting up sandbox pipeline. Purging quarantine path: %s", path)

	if err := os.RemoveAll(path); err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	log.Println("[INIT] Quarantine folder initialized and empty.")
	return nil
}
