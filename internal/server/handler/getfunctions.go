package handler

import (
	"context"
	"fmt"
	"oblak/internal/api"
	"oblak/internal/function"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetUserFunctions(ctx context.Context, request api.GetUserFunctionsRequestObject) (api.GetUserFunctionsResponseObject, error) {
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
	userFunctions, err := h.functionStore.GetFunctionsByUserId(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	response := createGetUserFunctionsResponse(userFunctions)

	return api.GetUserFunctions200JSONResponse(response), nil
}
func createGetUserFunctionsResponse(userFunctions []function.Function) []api.FunctionEntity {
	response := make([]api.FunctionEntity, len(userFunctions))
	for i, f := range userFunctions {
		response[i] = api.FunctionEntity{
			Id:     f.FunctionId.String(),
			Status: string(f.Status),
		}
	}
	return response
}
