package service

import (
	"context"
	"oblak/internal/vm/vm"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type ExecutionService struct {
	db        *pgxpool.Pool
	filestore *minio.Client
	runner    *vm.ExecutionRunner
}

func NewExecutionService(db *pgxpool.Pool, filestore *minio.Client, runner *vm.ExecutionRunner) *ExecutionService {
	return &ExecutionService{db, filestore, runner}
}

func (s *ExecutionService) Execute(ctx context.Context, msg ExecuteMessage) error {
	// TODO: implement
	return nil
}
