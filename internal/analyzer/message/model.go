package message

import (
	"oblak/internal/server/function"

	"github.com/google/uuid"
)

type FunctionMessage struct {
	FunctionId uuid.UUID       `db:"function_id"  json:"function_id"`
	Status     function.Status `db:"status" json:"status"`
	Bucket     string          `db:"bucket"    json:"bucket"`
	Path       string          `db:"path"      json:"path"`
}
