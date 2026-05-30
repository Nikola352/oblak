package function

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQuarantined          Status = "QUARANTINED"
	StatusExtracted            Status = "EXTRACTED"
	StatusScanning             Status = "SCANNING"
	StatusDetected             Status = "DETECTED"
	StatusVerified             Status = "VERIFIED"
	StatusPreparingEnvironment Status = "PREPARING_ENVIRONMENT"
	StatusReady                Status = "READY"
	StatusFailed               Status = "FAILED"
)

type Function struct {
	FunctionId  uuid.UUID `db:"function_id"  json:"function_id"`
	UserId      uuid.UUID `db:"user_id"      json:"user_id"`
	Status      Status    `db:"status"       json:"status"`
	ArchivePath string    `db:"archive_path" json:"archive_path"`
	DrivePath   *string   `db:"drive_path"   json:"drive_path"`
}

type FunctionWithLatestInvocation struct {
	FunctionId uuid.UUID
	UserId     uuid.UUID
	Status     Status

	// Executions fields (can be nil if the function was never run)
	LatestInvocationId        *uuid.UUID
	LatestInvocationStatus    *string
	LatestInvocationTimestamp *time.Time
}
