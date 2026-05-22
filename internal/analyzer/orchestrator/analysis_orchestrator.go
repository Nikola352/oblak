package orchestrator

import (
	"context"
	"errors"
	"log"
	"oblak/internal/analyzer/audit"
	"oblak/internal/analyzer/av"
	"oblak/internal/analyzer/dast"
	"oblak/internal/analyzer/llm"
	"oblak/internal/analyzer/sanitizer"
	"oblak/internal/analyzer/sast"
	"oblak/internal/server/filestore"
	"os"
)

type AnalysisOrchestrator struct {
	sanitizer         sanitizer.CodeSanitizer
	antivirus         av.Antivirus
	fileLoader        *FileLoader
	sastAnalyzer      sast.StaticAnalyzer
	llmJudge          llm.JudgeLLM
	detonationBox     dast.DetonationBox
	unzipper          *UnzipperService
	dependencyAuditor audit.DependencyAuditor
}

func NewOrchestrator(av av.Antivirus, llm llm.JudgeLLM, analyzer sast.StaticAnalyzer, box dast.DetonationBox, auditor audit.DependencyAuditor) *AnalysisOrchestrator {
	endpoint := "localhost:9000"
	accessKey := "minioadmin"
	secretKey := "minioadmin"
	fileloader := NewFileLoader(endpoint, accessKey, secretKey, string(filestore.QuarantineBucket))

	return &AnalysisOrchestrator{
		sanitizer:         sanitizer.CodeSanitizer{},
		antivirus:         av,
		detonationBox:     box,
		fileLoader:        fileloader,
		sastAnalyzer:      analyzer,
		llmJudge:          llm,
		unzipper:          NewUnzipper(-1),
		dependencyAuditor: auditor,
	}
}

func (ao *AnalysisOrchestrator) AnalyzeFile(ctx context.Context, fileName string) (AnalysisVerdict, error) {
	// 1. Get the filename from the RabbitMQ message payload
	log.Printf("Processing file from queue: %s", fileName)

	// 2. Download from MinIO to /tmp/quarantine/
	localPath, err := ao.fileLoader.Download(ctx, fileName)
	if err != nil {
		log.Printf("Error downloading file: %v", err)
		return FAILURE, err // Returning an error tells Watermill to Nack/Retry
	}

	// Ensure we clean up the local file after the scan finishes
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
		}
	}(localPath)
	extractPath := "/tmp/quarantine/extracted"

	defer func(dir string) {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("[CLEANUP] Warning: Failed to destroy workspace directory %s: %v", dir, err)
		} else {
			log.Printf("[CLEANUP] Successfully purged sandbox workspace: %s", dir)
		}
	}(extractPath)

	err = ao.unzipper.Extract(localPath, extractPath)
	log.Println(err)
	if err != nil {
		return FAILURE, err
	}
	log.Println("UNzziped")

	err = ao.dependencyAuditor.Audit(ctx, extractPath)
	log.Println(err)
	if err != nil {
		return FAILURE, err
	}

	// 4. Scan it
	isClean, err := ao.antivirus.ScanLocalPath(extractPath)
	if err != nil {
		log.Printf("[ANTIVIRUS] Antivirus scan failed to execute: %v", err)
		return FAILURE, err
	}

	// 5. Act on the results
	if !isClean {
		return FAILURE, errors.New("[ANTIVIRUS] file is UNSAFE. REJECTING FILE")
	}

	semReport, err := ao.sastAnalyzer.Run(extractPath)
	if err != nil {
		log.Fatalf("[SAST] Error running semgrep: %v", err)
	}
	if len(semReport.Results) != 0 {
		for _, finding := range semReport.Results {
			verdict, err := ao.askLLMForSast(finding)
			log.Println(err)
			if err != nil {
				continue
			}
			if verdict == llm.MALICIOUS {
				return MALICIOUS, errors.New("LLM Agent flagged the code as malicious, stopping investigation")
			}
		}
	}

	detonationResult, err := ao.detonationBox.Detonate(ctx, extractPath)
	if err != nil {
		return FAILURE, errors.New("[DETONATION] Error running detonation")
	}
	err = ao.detonationBox.WriteJSONReport(detonationResult, "/home/nikolavelemir/res")
	if err != nil {
		return FAILURE, err
	}

	log.Println("[ORCH] Asking LLM for log verdict")
	verdict, err := ao.llmJudge.AskForLogs(ctx, "/home/nikolavelemir/res")
	if err != nil {
		return FAILURE, err
	}
	log.Println("LLM RESPONDED!")
	log.Printf("%s %s %d", verdict.Verdict, verdict.Summary, verdict.ConfidenceScore)
	if verdict.Verdict == "MALICIOUS" {
		return MALICIOUS, nil
	}
	return SAFE, nil
}

func (ao *AnalysisOrchestrator) askLLMForSast(finding sast.SemgrepResult) (llm.SastVerdict, error) {
	msg := finding.Extra.Message
	cleanCode := ao.sanitizer.SanitizeCode(finding.Extra.Lines)
	verdict, err := ao.llmJudge.AskForSAST(msg, cleanCode)
	if err != nil {
		log.Printf("Judge failed for finding %s: %v", finding.CheckID, err)
		return llm.FAILURE, err
	}

	log.Printf("[QWEN VERDICT] %s: %s", finding.CheckID, verdict)
	return verdict, nil
}
