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

func (a *ClamAV) ScanStream(ctx context.Context, reader io.Reader) (bool, error) {
	log.Println("[CLAMAV] Beginning scan for viruses")
	results, err := a.client.ScanStreamContext(ctx, reader)
	if err != nil {
		return false, fmt.Errorf("[CLAMAV] failed to initiate scan: %w", err)
	}
	for result := range results {
		if result.Status == clamd.RES_OK {
			log.Println("[CLAMAV] File looks clean")
			return true, nil // Clean
		} else if result.Status == clamd.RES_FOUND {
			log.Printf("[CLAMAV] Virus detected! Description: %s", result.Description)
			return false, nil // Infected
		} else if result.Status == clamd.RES_ERROR {
			return false, fmt.Errorf("[CLAMAV] error: %s", result.Description)
		}
	}

	return false, fmt.Errorf("[CLAMAV] scan finished without returning a valid status")
}
