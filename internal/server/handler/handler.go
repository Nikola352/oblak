package handler

import (
	"oblak/internal/server/events"
	"oblak/internal/server/function"

	"github.com/minio/minio-go/v7"
)

type Handler struct {
	functionStore *function.Store
	filestore     *minio.Client
	quarantineBus *events.Bus[events.QuarantineEvent]
	extractionBus *events.Bus[events.ExtractionEvent]
}

func New(functionStore *function.Store, filestore *minio.Client, quarantineBus *events.Bus[events.QuarantineEvent],
	extractionBus *events.Bus[events.ExtractionEvent]) *Handler {
	return &Handler{functionStore: functionStore, filestore: filestore, quarantineBus: quarantineBus, extractionBus: extractionBus}
}
