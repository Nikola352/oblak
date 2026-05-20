package analyzer

import (
	"context"
	"fmt"
	"io"

	"github.com/bchisham/go-clamd"
)

type Antivirus struct {
	client *clamd.Clamd
}

func NewAntivirus(clamPath string) *Antivirus {
	return &Antivirus{
		client: clamd.NewClamd(clamPath),
	}
}
func (a *Antivirus) Ping() error {
	return a.client.Ping()
}

func (a *Antivirus) ScanStream(ctx *context.Context, reader *io.Reader) (bool, error) {
	_, err := a.client.ScanStreamContext(*ctx, *reader)
	if err != nil {
		return false, fmt.Errorf("failed to initiate scan: %w", err)
	}
	return false, fmt.Errorf("scan finished without returning a valid status")
}
