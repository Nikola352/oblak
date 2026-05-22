package audit

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type PipAuditor struct {
	auditBinaryPath string
}

func NewPipAuditor(auditBinaryPath string) *PipAuditor {
	return &PipAuditor{auditBinaryPath: auditBinaryPath}
}

func (pa *PipAuditor) Audit(ctx context.Context, dirPath string) error {
	log.Println("[PIP Auditor] Beginning dependency audit")
	reqPath := filepath.Join(dirPath, "requirements.txt")
	if _, err := os.Stat(reqPath); os.IsNotExist(err) {
		return nil
	}
	cmd := exec.CommandContext(ctx, pa.auditBinaryPath, "-r", reqPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("[PIP Auditor] Found flagged vulnerabilities or failed")
		return fmt.Errorf("dependency audit flagged vulnerabilities or failed:\n%s", string(output))
	}
	log.Println("[PIP Auditor] Audit finished, no problems found")

	return nil
}
