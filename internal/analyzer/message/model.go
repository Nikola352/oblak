package message

import (
	"github.com/google/uuid"
)

type FunctionMessage struct {
	FunctionId uuid.UUID `db:"function_id"  json:"function_id"`
	Bucket     string    `db:"bucket"    json:"bucket"`
	Path       string    `db:"path"      json:"path"`
}
