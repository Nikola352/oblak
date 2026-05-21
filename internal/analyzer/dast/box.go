package dast

import "context"

type DetonationBox interface {
	Detonate(ctx context.Context) error
}
