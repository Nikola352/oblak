package analyzer

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// SemgrepOutput matches the simplified JSON structure from Semgrep CLI
type SemgrepOutput struct {
	Results []SemgrepResult `json:"results"`
}

type SemgrepResult struct {
	CheckID string    `json:"check_id"`
	Path    string    `json:"path"`
	Start   Position  `json:"start"`
	End     Position  `json:"end"`
	Extra   ExtraData `json:"extra"`
}

type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type ExtraData struct {
	Message  string `json:"message"`
	Lines    string `json:"lines"` // This is the actual code snippet Semgrep found
	Severity string `json:"severity"`
}

// RunSemgrep executes the scan and returns findings
func RunSemgrep(targetPath string) (*SemgrepOutput, error) {
	// --json: Returns machine-readable data
	// --config auto: Uses the best community rules for the language detected
	// --quiet: Suppresses progress bars and logs
	semgrepPath := "/home/nikolavelemir/faks/rbs/oblak/.venv/bin/semgrep"
	//
	cmd := exec.Command(semgrepPath, "scan", "--json", "--config", "auto", "--quiet", targetPath)
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
