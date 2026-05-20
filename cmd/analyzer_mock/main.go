package main

import (
	"log"
	"oblak/internal/analyzer_mock"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	if err := analyzer_mock.StartPublisher(); err != nil {
		log.Fatalf("Failed to start service: %v", err)

	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v, shutting down gracefully...", sig)

	log.Println("Consumer stopped. Exiting.")

}
