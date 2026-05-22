package av

import (
	"context"
	"io"
)

type Antivirus interface {
	ScanStream(ctx context.Context, reader io.Reader) (bool, error)
}
