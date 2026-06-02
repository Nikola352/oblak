package handler

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/api"
	"oblak/internal/invocation"
	"oblak/internal/server/invocations"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) GetInvocationDetails(ctx context.Context, request api.GetInvocationDetailsRequestObject) (api.GetInvocationDetailsResponseObject, error) {
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

	foundInvocation, err := h.invocationStore.GetInvocationDetailsForUser(ctx, userUUID, request.InvocationId)
	if err != nil {
		return nil, err
	}
	logs, err := h.extractLogs(ctx, foundInvocation)
	if err != nil {
		return nil, err
	}
	log.Println(foundInvocation.LogPath)
	response := createInvocationDetailsResponse(foundInvocation, logs)
	return api.GetInvocationDetails200JSONResponse(response), nil
}

func createInvocationDetailsResponse(foundInvocation *invocation.Invocation, logs []invocations.LogLine) api.SingleInvocationDetailsResponse {
	apiLogs := make([]api.LogLine, len(logs))
	for i, line := range logs {
		apiLogs[i] = api.LogLine{
			Timestamp: line.Timestamp,
			Stream:    api.LogLineStream(line.Stream), // Safely cast string to OpenAPI Enum type
			Message:   line.Message,
		}
	}

	// 2. Safely handle the nullable EndTime pointer
	var endTimeStr *time.Time
	if foundInvocation.EndTime != nil {
		//formatted := foundInvocation.EndTime.Format(time.RFC3339)
		endTimeStr = foundInvocation.EndTime
	}
	return api.SingleInvocationDetailsResponse{
		InvocationId:   foundInvocation.InvocationId,
		Status:         string(foundInvocation.Status),
		InvocationTime: *foundInvocation.InvocationTime,
		EndTime:        endTimeStr,
		Logs:           apiLogs,
	}
}

func (h *Handler) extractLogs(ctx context.Context, invocation *invocation.Invocation) ([]invocations.LogLine, error) {
	if invocation.LogPath == nil {
		return make([]invocations.LogLine, 0), nil
	}

	return h.invocationLogStore.GetLogsByInvocationId(ctx, *invocation.LogPath)
}
