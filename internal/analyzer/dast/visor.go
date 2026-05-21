package dast

import "context"

type GVisorBox struct {
}

func NewGVisorBox() *GVisorBox {
	return &GVisorBox{}
}

func (b GVisorBox) Detonate(ctx context.Context) error {
	return nil
}
