package events

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeQuarantine EventType = "Quarantine"
	EventTypeExtracted  EventType = "Extracted"
)

type Event interface {
	GetType() string
	GetTimestamp() time.Time
}

type QuarantineEvent struct {
	FunctionID uuid.UUID `json:"function_id"`
	UserID     uuid.UUID `json:"user_id"`
	Bucket     string    `json:"bucket"`
	Path       string    `json:"path"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e QuarantineEvent) GetType() EventType      { return EventTypeQuarantine }
func (e QuarantineEvent) GetTimestamp() time.Time { return e.Timestamp }

type ExtractionEvent struct {
	FunctionID uuid.UUID `json:"function_id"`
	UserID     uuid.UUID `json:"user_id"`
	Bucket     string    `json:"bucket"`
	Path       string    `json:"path"`
	Timestamp  time.Time `json:"timestamp"`
}

func (e ExtractionEvent) GetType() EventType      { return EventTypeExtracted }
func (e ExtractionEvent) GetTimestamp() time.Time { return e.Timestamp }
