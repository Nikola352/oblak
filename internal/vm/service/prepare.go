package service

import (
	"context"
	"errors"
	"log"
	"oblak/internal/function"
	"oblak/internal/vm/vm"
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

	if err = s.runner.PrepareEnvironment(ctx, msg.CodeObjectName, msg.DependenciesObjectName); err != nil {
		var userErr *vm.UserError
		if errors.As(err, &userErr) {
			log.Printf("environment prepare: %v", userErr)
			if dbErr := s.functionStore.UpdateFunctionStatus(ctx, msg.FunctionId, function.StatusFailed); dbErr != nil {
				log.Printf("failed to set failed status after failed prepare: %v", dbErr)
			}
			return nil // ack
		}
		if dbErr := s.functionStore.UpdateFunctionStatus(ctx, msg.FunctionId, function.StatusVerified); dbErr != nil {
			log.Printf("failed to reset status after prepare error: %v", dbErr)
		}
		return err
	}

	if err = s.functionStore.UpdateFunctionStatus(ctx, msg.FunctionId, function.StatusReady); err != nil {
		return err
	}

	return nil
}
