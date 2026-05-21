package dast

import "context"

type DetonationBox interface {
	Detonate(ctx context.Context, localPath string) (string, error)
}
