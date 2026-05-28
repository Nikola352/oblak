package function

import (
	"github.com/google/uuid"
)

type Status string

const (
	StatusQuarantined          Status = "QUARANTINED"
	StatusExtracted            Status = "EXTRACTED"
	StatusProcessing           Status = "PROCESSING"
	StatusScanning             Status = "SCANNING"
	StatusDetected             Status = "DETECTED"
	StatusVerified             Status = "VERIFIED"
	StatusPreparingEnvironment Status = "PREPARING_ENVIRONMENT"
	StatusReady                Status = "READY"
	StatusFailed               Status = "FAILED"
	StatusDeleted              Status = "DELETED"
)

type Function struct {
	FunctionId uuid.UUID `db:"function_id"  json:"function_id"`
	UserId     uuid.UUID `db:"user_id"`
	Status     Status    `db:"status" json:"status"`
	Bucket     string    `db:"bucket"    json:"bucket"`
	Path       string    `db:"path"      json:"path"`
}
