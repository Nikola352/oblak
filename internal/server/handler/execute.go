package handler

import (
	"context"
	"fmt"
	"oblak/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) ExecuteLambda(ctx context.Context, request api.ExecuteLambdaRequestObject) (api.ExecuteLambdaResponseObject, error) {
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

	// upload to DB
	//f, err := h.functionStore.CreateFunction(ctx, userUUID, function.StatusQuarantined, objectName)
	//if err != nil {
	//	return nil, fmt.Errorf("error saving function to db: %w", err)
	//}
	fmt.Println(userUUID, request.FunctionId)

	//upload to rabbitMQ or channel event
	//h.quarantineBus.Publish(events.QuarantineEvent{
	//	FunctionID: f.FunctionId,
	//	UserID:     f.UserId,
	//	Bucket:     filestore.NewBucketRegistry().Name(filestore.QuarantineBucket),
	//	Path:       f.ArchivePath,
	//	Timestamp:  time.Now(),
	//})

	return api.ExecuteLambda200JSONResponse{Status: "ok"}, nil
}
