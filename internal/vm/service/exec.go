package service

import (
	"context"
	"log"
	"oblak/internal/invocation"
	"oblak/internal/vm/vm"
	"time"
)

type ExecutionService struct {
	invocationStore *invocation.Store
	runner          *vm.ExecutionRunner
}

func NewExecutionService(invocationStore *invocation.Store, runner *vm.ExecutionRunner) *ExecutionService {
	return &ExecutionService{invocationStore, runner}
}

func (s *ExecutionService) Execute(ctx context.Context, msg ExecuteMessage) error {
	claimed, err := s.invocationStore.UpdateInvocationStatusIf(
		ctx,
		msg.InvocationId,
		invocation.StatusPending,
		invocation.StatusExecuting,
	)
	if err != nil {
		return err
	}
	if !claimed { // idempotency check
		return nil // ack
	}

	if err = s.runner.Execute(ctx, msg.CodeObjectName, msg.DependenciesObjectName); err != nil {
		if dbErr := s.invocationStore.UpdateInvocationStatus(ctx, msg.InvocationId, invocation.StatusPending); dbErr != nil {
			log.Printf("failed to reset status after execute error: %v", dbErr)
		}
		return err
	}

	if err = s.invocationStore.UpdateInvocationStatusAndEndTime(ctx, msg.InvocationId, invocation.StatusDone, time.Now()); err != nil {
		log.Printf("failed to reset status after execute error: %v", err)
	}

	return nil
}
