package extractor

import (
	"context"
	"log"
	"oblak/internal/function"
	"oblak/internal/server/analyzerpublisher"
	"oblak/internal/server/events"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type Service struct {
	db            *pgxpool.Pool
	filestore     *minio.Client
	quarantineBus *events.Bus[events.QuarantineEvent]
	extractionBus *events.Bus[events.ExtractionEvent]
	functionStore *function.Store
	extractor     *Extractor
}

func NewService(db *pgxpool.Pool, filestore *minio.Client, functionStore *function.Store,
	quarantineBus *events.Bus[events.QuarantineEvent], extractionBus *events.Bus[events.ExtractionEvent]) *Service {
	return &Service{
		db:            db,
		filestore:     filestore,
		quarantineBus: quarantineBus,
		extractionBus: extractionBus,
		functionStore: functionStore,
		extractor:     NewExtractor(filestore),
	}
}

func (s *Service) StartQuarantineWorker() {
	ch := s.quarantineBus.Subscribe()
	go func() {
		for event := range ch {
			log.Printf("Quarantine event: function %s uploaded", event.FunctionID)
			// try to extract (extractor saves to the bucket)
			err := s.extractor.ExtractFromQuarantine(event)
			functionStatus := function.StatusExtracted
			if err != nil {
				functionStatus = function.StatusFailed
				log.Printf("Error extracting zip for function with id=%s, %s",
					event.FunctionID, err)
			}

			// updates the status in the db
			err = s.functionStore.UpdateFunctionStatus(context.Background(), event.FunctionID, functionStatus)
			if err != nil {
				log.Printf("Error updating status %s",
					err)
				return
			}

			// publish other event
			s.extractionBus.Publish(events.ExtractionEvent{
				FunctionID: event.FunctionID,
				UserID:     event.UserID,
				Bucket:     event.Bucket,
				Path:       event.Path,
				Timestamp:  time.Now(),
			})

		}
	}()
}

func (s *Service) StartExtractionWorker() {
	ch := s.extractionBus.Subscribe()
	go func() {
		for event := range ch {
			log.Printf("Extraction event: function %s", event.FunctionID)
			funcMessage := analyzerpublisher.CreateMessage(event.FunctionID, event.Path)
			if err := analyzerpublisher.StartPublisher(funcMessage); err != nil {
				log.Fatalf("Failed to start service: %v", err)
			}
		}
	}()
}
