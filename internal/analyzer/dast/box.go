package dast

import "context"

type DetonationBox interface {
	Detonate(ctx context.Context, localPath string) (*ExecutionResult, error)
	WriteJSONReport(result *ExecutionResult, outputPath string) error
}
