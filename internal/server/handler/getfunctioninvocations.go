package handler

import (
	"context"
	"fmt"
	"oblak/internal/api"
	"oblak/internal/invocation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetFunctionInvocations(ctx context.Context, request api.GetFunctionInvocationsRequestObject) (api.GetFunctionInvocationsResponseObject, error) {
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

	invocations, err := h.invocationStore.GetInvocationsForUserAndFunction(ctx, userUUID, request.FunctionId)

	if err != nil {
		return nil, err
	}
	response := createInvocationResponse(invocations)
	return api.GetFunctionInvocations200JSONResponse(response), nil
}

func createInvocationResponse(userInvocations []invocation.Invocation) []api.InvocationResponse {
	response := make([]api.InvocationResponse, len(userInvocations))
	for i, f := range userInvocations {

		response[i] = api.InvocationResponse{
			InvocationId:   f.InvocationId,
			Status:         string(f.Status),
			InvocationTime: *f.InvocationTime,
			EndTime:        f.EndTime,
		}
	}
	return response
}
