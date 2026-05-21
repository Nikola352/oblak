package analyzer

import (
	"context"
	"errors"
	"log"
	"os"
)

type AnalysisOrchestrator struct {
	antivirus  *Antivirus
	fileLoader *FileLoader
}

func NewOrchestrator() *AnalysisOrchestrator {
	endpoint := "localhost:9000"
	accessKey := "minioadmin"
	secretKey := "minioadmin"
	fileloader := NewFileLoader(endpoint, accessKey, secretKey, "quarantine")

	return &AnalysisOrchestrator{
		antivirus:  NewAntivirus("tcp://localhost:3310"),
		fileLoader: fileloader,
	}
}
func (ao *AnalysisOrchestrator) AnalyzeFile(ctx context.Context, fileName string) error {
	// 1. Get the filename from the RabbitMQ message payload
	log.Printf("Processing file from queue: %s", fileName)

	// 2. Download from MinIO to /tmp/quarantine/
	localPath, err := ao.fileLoader.Download(ctx, fileName)
	if err != nil {
		log.Printf("Error downloading file: %v", err)
		return err // Returning an error tells Watermill to Nack/Retry
	}

	// Ensure we clean up the local file after the scan finishes
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {

		}
	}(localPath)

	// 3. Open the downloaded file to pass it to ClamAV
	file, err := os.Open(localPath)
	if err != nil {
		log.Printf("Error opening local file: %v", err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// 4. Scan it
	isClean, err := ao.antivirus.ScanStream(ctx, file)
	if err != nil {
		log.Printf("[CLAMAV] Antivirus scan failed to execute: %v", err)
		return err
	}

	// 5. Act on the results
	if !isClean {
		return errors.New("[CLAMAV] file is UNSAFE. REJECTING FILE")
	}

	semReport, err := RunSemgrep(localPath)
	if err != nil {
		log.Fatalf("[SEMGREP] Error running semgrep: %v", err)
	}
	for _, finding := range semReport.Results {
		log.Println("dsadadas", finding)
	}
	return nil
}
