package service

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/invocation"
	"oblak/internal/vm/vm"
	"time"

	"github.com/google/uuid"
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

	beginTime := time.Now()

	logsObjectName := generateObjectName(msg.InvocationId, beginTime)

	if err = s.runner.Execute(ctx, msg.CodeObjectName, msg.DependenciesObjectName, logsObjectName); err != nil {
		if dbErr := s.invocationStore.UpdateInvocationStatus(ctx, msg.InvocationId, invocation.StatusPending); dbErr != nil {
			log.Printf("failed to reset status after execute error: %v", dbErr)
		}
		return err
	}

	endTime := time.Now()
	if err = s.invocationStore.UpdateInvocationStatusAndLogPathAndEndTime(ctx, msg.InvocationId, invocation.StatusDone, logsObjectName, endTime); err != nil {
		log.Printf("failed to reset status after execute error: %v", err)
	}

	return nil
}
func generateObjectName(id uuid.UUID, currentTime time.Time) string {

	return fmt.Sprintf("%s_%s.log", id.String(), currentTime.Format("2006_01_02_15_04_05"))
}
