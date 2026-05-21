package sast

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

// SemgrepOutput matches the simplified JSON structure from Semgrep CLI

type SemgrepAnalyzer struct {
	binaryPath string
}

func NewSemgrepAnalyzer(binaryPath string) *SemgrepAnalyzer {
	return &SemgrepAnalyzer{binaryPath: binaryPath}
}

func (a SemgrepAnalyzer) Run(targetPath string) (*SemgrepOutput, error) {
	//
	log.Println("[SEMGREP] Beginning static analysis")
	cmd := exec.Command(a.binaryPath, "scan", "--json", "--config", "r/python", "--quiet", targetPath)
	output, err := cmd.Output()
	if err != nil {
		// Semgrep returns exit code 1 if findings are found; we check output length instead
		if len(output) == 0 {
			return nil, fmt.Errorf("semgrep execution failed: %w", err)
		}
	}

	var report SemgrepOutput
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("failed to parse semgrep json: %w", err)
	}
	fmt.Println("[SEMGREP] Finished analyzing!")
	return &report, nil
}
