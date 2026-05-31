package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"oblak/internal/api"
	"oblak/internal/server/vmexecutepublisher"
	"time"

	"github.com/google/uuid"
)

const maxPayloadSize = 1024 * 1024 // 1 MB

func (h *Handler) ExecuteLambda(ctx context.Context, request api.ExecuteLambdaRequestObject) (api.ExecuteLambdaResponseObject, error) {
	payload, err := validatePayload(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to validate payload: %w", err)
	}
	log.Printf("Payload is valid: %s", payload)
	functionId, err := uuid.Parse(request.FunctionId)
	if err != nil {
		return nil, fmt.Errorf("function_id is not valid UUID")
	}
	// check if the user is owner of the method he requests to execute
	function, err := h.functionStore.GetFunctionById(ctx, functionId)
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
	err = vmexecutepublisher.Publish(invocation.InvocationId, function.ArchivePath, *function.DrivePath, payload)
	if err != nil {
		return nil, err
	}

	return api.ExecuteLambda200JSONResponse{Status: "ok"}, nil
}

func validatePayload(body *api.ExecuteLambdaJSONRequestBody) (string, error) {
	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	if len(payloadBytes) > maxPayloadSize {
		return "", fmt.Errorf("payload to large over %d", maxPayloadSize)
	}

	payload := string(payloadBytes)

	return payload, nil
}
