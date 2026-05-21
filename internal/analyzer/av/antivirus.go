package av

import (
	"context"
	"io"
)

type Antivirus interface {
	Ping() error
	ScanStream(ctx context.Context, reader io.Reader) (bool, error)
}
