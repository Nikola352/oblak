package handler

import (
	"context"
	"errors"
	"oblak/internal/api"
	"oblak/internal/function"
)

func (h *Handler) GetFunctionStatus(ctx context.Context, request api.GetFunctionStatusRequestObject) (api.GetFunctionStatusResponseObject, error) {
	functionStatus, err := h.functionStore.GetFunctionStatusById(ctx, request.FunctionId)
	if err != nil {
		if errors.Is(err, function.ErrFunctionNotFound) {
			return api.GetFunctionStatus404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Error: "function not found",
				},
			}, nil
		}

		return api.GetFunctionStatus400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error: "bad request",
			},
		}, nil
	}

	return api.GetFunctionStatus200JSONResponse{Status: functionStatus}, nil
}
