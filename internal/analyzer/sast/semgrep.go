package sast

import (
	"encoding/json"
	"fmt"
	"log"
	"oblak/internal/analyzer/config"
	"os"
	"os/exec"
	"path/filepath"
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
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed resolving absolute engine route path: %w", err)
	}
	log.Println("[SEMGREP] Beginning static analysis")

	err = os.Setenv("SEMGREP_APP_TOKEN", config.Cfg.SemgrepToken)
	if err != nil {
		return nil, fmt.Errorf("failed to set semgrep key to env variable: %w", err)
	}

	cmd := exec.Command(a.binaryPath, "scan", "--json", "--config", "r/python", "--quiet", absPath)
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
	log.Println("[SEMGREP] Finished analyzing!")
	return &report, nil
}
