package service

import (
	"context"
	"oblak/internal/vm/vm"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type EnvironmentPrepareService struct {
	db        *pgxpool.Pool
	filestore *minio.Client
	runner    *vm.EnvironmentPrepareRunner
}

func NewEnvironmentPrepareService(db *pgxpool.Pool, filestore *minio.Client, runner *vm.EnvironmentPrepareRunner) *EnvironmentPrepareService {
	return &EnvironmentPrepareService{db, filestore, runner}
}

func (s *EnvironmentPrepareService) Prepare(ctx context.Context, msg BuildMessage) error {
	// TODO: Implement
	return nil
}
