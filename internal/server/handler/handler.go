package handler

import (
	"oblak/internal/function"
	"oblak/internal/invocation"
	"oblak/internal/server/events"
	"oblak/internal/server/ratelimiter"

	"github.com/minio/minio-go/v7"
)

type Handler struct {
	functionStore   *function.Store
	invocationStore *invocation.Store
	filestore       *minio.Client
	quarantineBus   *events.Bus[events.QuarantineEvent]
	extractionBus   *events.Bus[events.ExtractionEvent]
	tokenBucket     *ratelimiter.TokenBucket
}

func New(functionStore *function.Store, invocationStore *invocation.Store, filestore *minio.Client, quarantineBus *events.Bus[events.QuarantineEvent],
	extractionBus *events.Bus[events.ExtractionEvent], tokenBucket *ratelimiter.TokenBucket) *Handler {
	return &Handler{functionStore: functionStore, invocationStore: invocationStore,
		filestore: filestore, quarantineBus: quarantineBus, extractionBus: extractionBus, tokenBucket: tokenBucket}
}
