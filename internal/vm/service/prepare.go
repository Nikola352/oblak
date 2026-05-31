package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"oblak/internal/function"
	"oblak/internal/vm/vm"

	"github.com/google/uuid"
)

type EnvironmentPrepareService struct {
	functionStore *function.Store
	runner        *vm.EnvironmentPrepareRunner
}

func NewEnvironmentPrepareService(functionStore *function.Store, runner *vm.EnvironmentPrepareRunner) *EnvironmentPrepareService {
	return &EnvironmentPrepareService{functionStore, runner}
}

func (s *EnvironmentPrepareService) Prepare(ctx context.Context, msg BuildMessage) error {
	claimed, err := s.functionStore.UpdateFunctionStatusIf(ctx, msg.FunctionId, function.StatusVerified, function.StatusPreparingEnvironment)
	if err != nil {
		return err
	}
	if !claimed { // idempotency check
		return nil // ack
	}

	driveObjectName := fmt.Sprintf("%s-deps.ext4", uuid.New().String())

	objectName := generatePrepareObjectName(msg.FunctionId)
	if err = s.runner.PrepareEnvironment(ctx, msg.CodeObjectName, driveObjectName, objectName); err != nil {
		var userErr *vm.UserError
		if errors.As(err, &userErr) {
			log.Printf("environment prepare: %v", userErr)
			if dbErr := s.functionStore.UpdateFunctionStatus(ctx, msg.FunctionId, function.StatusFailed); dbErr != nil {
				log.Printf("failed to set failed status after failed prepare: %v", dbErr)
			}
			return nil // ack
		}
		log.Printf("environment prepare: %v", err)
		if dbErr := s.functionStore.UpdateFunctionStatus(ctx, msg.FunctionId, function.StatusVerified); dbErr != nil {
			log.Printf("failed to reset status after prepare error: %v", dbErr)
		}
		return err
	}
	if err = s.functionStore.UpdateFunctionStatusAndDrivePath(ctx, msg.FunctionId, function.StatusReady, driveObjectName); err != nil {
		log.Printf("failed to write status to db: %v", err)
		return err
	}
	return nil
}

func generatePrepareObjectName(id uuid.UUID) string {

	return fmt.Sprintf("%s_prepare.log", id.String())
}
