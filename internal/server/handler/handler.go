package handler

import (
	"oblak/internal/function"
	"oblak/internal/invocation"
	"oblak/internal/server/events"

	"github.com/minio/minio-go/v7"
)

type Handler struct {
	functionStore   *function.Store
	invocationStore *invocation.Store
	filestore       *minio.Client
	quarantineBus   *events.Bus[events.QuarantineEvent]
	extractionBus   *events.Bus[events.ExtractionEvent]
}

func New(functionStore *function.Store, invocationStore *invocation.Store, filestore *minio.Client, quarantineBus *events.Bus[events.QuarantineEvent],
	extractionBus *events.Bus[events.ExtractionEvent]) *Handler {
	return &Handler{functionStore: functionStore, invocationStore: invocationStore, filestore: filestore, quarantineBus: quarantineBus, extractionBus: extractionBus}
}
