package analyzer

import (
	"context"
	"errors"
	"log"
	"oblak/internal/analyzer/av"
	"oblak/internal/analyzer/dast"
	"oblak/internal/analyzer/llm"
	"oblak/internal/analyzer/sast"
	"os"
)

type AnalysisOrchestrator struct {
	antivirus     av.Antivirus
	fileLoader    *FileLoader
	sastAnalyzer  sast.StaticAnalyzer
	llmJudge      llm.JudgeLLM
	detonationBox dast.DetonationBox
}

func NewOrchestrator(av av.Antivirus, llm llm.JudgeLLM, analyzer sast.StaticAnalyzer, box dast.DetonationBox) *AnalysisOrchestrator {
	endpoint := "localhost:9000"
	accessKey := "minioadmin"
	secretKey := "minioadmin"
	fileloader := NewFileLoader(endpoint, accessKey, secretKey, "quarantine")

	return &AnalysisOrchestrator{
		antivirus:     av,
		detonationBox: box,
		//antivirus:    av.NewClamAV("tcp://localhost:3310"),
		fileLoader:   fileloader,
		sastAnalyzer: analyzer,
		llmJudge:     llm,
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

	semReport, err := ao.sastAnalyzer.Run(localPath)
	if err != nil {
		log.Fatalf("[SEMGREP] Error running semgrep: %v", err)
	}
	if len(semReport.Results) == 0 {
		//TODO PASS TO DETONATION BOX
		return nil
	}
	for _, finding := range semReport.Results {
		msg := finding.Extra.Message
		code := finding.Extra.Lines

		verdict, err := ao.llmJudge.Ask(msg, code)
		if err != nil {
			log.Printf("Judge failed for finding %s: %v", finding.CheckID, err)
			continue // Or return err if you want to fail the whole message
		}

		log.Printf("[QWEN VERDICT] %s: %s", finding.CheckID, verdict)
	}
	return nil
}
