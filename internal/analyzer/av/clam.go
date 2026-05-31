package av

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/bchisham/go-clamd"
)

type ClamAV struct {
	client *clamd.Clamd
}

func NewClamAV(clamPath string) *ClamAV {
	return &ClamAV{
		client: clamd.NewClamd(clamPath),
	}
}
func (a *ClamAV) ScanLocalPath(path string) (bool, error) {
	log.Printf("[CLAMAV] Initiating multi-threaded path scan on: %s", path)

	// MultiScanFile maps to clamd's MULTISCAN command, dropping execution into SMP threads
	results, err := a.client.MultiScanFile(path)
	if err != nil {
		return false, fmt.Errorf("[CLAMAV] multi-threaded scan failed to start: %w", err)
	}

	isClean := true

	// Read results stream from the channel
	for result := range results {
		switch result.Status {
		case clamd.RES_FOUND:
			{
				log.Printf("[CLAMAV] Thread flagged virus! File: %s | Threat: %s", result.Path, result.Description)

				isClean = false
				break
			}
		case clamd.RES_ERROR:
			{
				return false, fmt.Errorf("[CLAMAV] engine thread failure on file %s: %s", result.Path, result.Description)
			}
		}

	}

	if !isClean {
		return false, nil // Malicious signature caught
	}

	log.Println("[CLAMAV] Multi-threaded sweep complete: Workspace is clean")
	return true, nil
}
func (a *ClamAV) ScanStream(ctx context.Context, reader io.Reader) (bool, error) {
	log.Println("[CLAMAV] Beginning scan for viruses")
	results, err := a.client.ScanStreamContext(ctx, reader)
	if err != nil {
		return false, fmt.Errorf("[CLAMAV] failed to initiate scan: %w", err)
	}
	var isClean bool
	var scanErr error
	var statusFound bool
	for result := range results {
		if statusFound {
			continue
		}

		switch result.Status {
		case clamd.RES_OK:
			log.Println("[CLAMAV] File looks clean")
			isClean = true
			statusFound = true

		case clamd.RES_ERROR:
			scanErr = fmt.Errorf("[CLAMAV] error: %s", result.Description)
			statusFound = true

		case clamd.RES_FOUND:
			log.Printf("[CLAMAV] Virus detected! Description: %s", result.Description)
			isClean = false
			statusFound = true
		}
	}
	if !statusFound {
		return false, fmt.Errorf("[CLAMAV] scan finished without returning a valid status")
	}
	if scanErr != nil {
		return false, scanErr
	}

	return isClean, nil
}
