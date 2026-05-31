package invocation

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusExecuting Status = "EXECUTING"
	StatusDone      Status = "DONE"
	StatusFailed    Status = "FAILED"
)

type Invocation struct {
	InvocationId   uuid.UUID  `db:"invocation_id"`
	FunctionId     uuid.UUID  `db:"function_id"`
	Status         Status     `db:"status"`
	InvocationTime *time.Time `db:"invocation_time"`
	EndTime        *time.Time `db:"end_time"`
	LogPath        *string    `db:"log_path"`
}
