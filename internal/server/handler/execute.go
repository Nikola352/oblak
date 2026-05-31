package handler

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/api"
	"oblak/internal/server/vmexecutepublisher"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ExecuteLambda(ctx context.Context, request api.ExecuteLambdaRequestObject) (api.ExecuteLambdaResponseObject, error) {
	// rate limiting token bucket
	if !h.tokenBucket.CheckTokenCondition() {
		return nil, fmt.Errorf("execution rate limit exceeded")
	}
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return nil, fmt.Errorf("failed to get gin context")
	}
	userID, exists := ginCtx.Get("user_id")
	if !exists {
		return nil, fmt.Errorf("user_id not found in context")
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("user_id is not a string")
	}
	functionId, err := uuid.Parse(request.FunctionId)
	if err != nil {
		return nil, fmt.Errorf("function_id is not valid UUID")
	}

	// check if the user is owner of the method he requests to execute
	function, err := h.functionStore.CheckFunctionIdAndUserId(ctx, functionId, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check function %v", err)
	}

	// upload invocation to DB
	invocation, err := h.invocationStore.CreateInvocation(ctx, function.FunctionId, time.Now())
	if err != nil {
		return nil, fmt.Errorf("error saving invocation to db: %w", err)
	}
	log.Printf("Invocation created: %v", invocation.InvocationId)

	// upload event for firecracker
	if function.DrivePath == nil {
		return nil, fmt.Errorf("function drive path is nil")
	}
	err = vmexecutepublisher.Publish(invocation.InvocationId, function.ArchivePath, *function.DrivePath)
	if err != nil {
		return nil, err
	}

	return api.ExecuteLambda200JSONResponse{Status: "ok"}, nil
}
