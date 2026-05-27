package queue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"oblak/internal/function"
	"oblak/internal/vm/service"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
)

const maxPayloadBytes = 64 * 1024

type BuildHandler struct {
	service *service.EnvironmentPrepareService
}

func NewBuildHandler(service *service.EnvironmentPrepareService) *BuildHandler {
	return &BuildHandler{service}
}

func (h *BuildHandler) Handle(msg *message.Message) error {
	var m service.BuildMessage
	if err := decodePayload(msg.Payload, &m); err != nil {
		log.Printf("build handler: malformed message: %v", err)
		msg.Ack()
		return nil
	}
	if err := validateObjectName(m.CodeObjectName); err != nil {
		log.Printf("build handler: invalid code_object_name: %v", err)
		msg.Ack()
		return nil
	}
	if err := validateObjectName(m.DependenciesObjectName); err != nil {
		log.Printf("build handler: invalid dependencies_object_name: %v", err)
		msg.Ack()
		return nil
	}
	if err := h.service.Prepare(msg.Context(), m); err != nil {
		return err
	}
	return nil
}

type ExecuteHandler struct {
	service *service.ExecutionService
}

func NewExecuteHandler(service *service.ExecutionService) *ExecuteHandler {
	return &ExecuteHandler{service}
}

func (h *ExecuteHandler) Handle(msg *message.Message) error {
	var m service.ExecuteMessage
	if err := decodePayload(msg.Payload, &m); err != nil {
		log.Printf("execute handler: malformed message: %v", err)
		msg.Ack()
		return nil
	}
	if err := validateObjectName(m.CodeObjectName); err != nil {
		log.Printf("execute handler: invalid code_object_name: %v", err)
		msg.Ack()
		return nil
	}
	if err := validateObjectName(m.DependenciesObjectName); err != nil {
		log.Printf("execute handler: invalid dependencies_object_name: %v", err)
		msg.Ack()
		return nil
	}
	if err := h.service.Execute(msg.Context(), m); err != nil {
		return err
	}
	return nil
}

// statusFailPublisher wraps a Publisher so that publishing to the DLQ also
// marks the corresponding function as failed in the database.
type statusFailPublisher struct {
	base  message.Publisher
	store *function.Store
}

func (p *statusFailPublisher) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		var m service.BuildMessage
		if err := decodePayload(msg.Payload, &m); err == nil {
			if dbErr := p.store.UpdateFunctionStatus(msg.Context(), m.FunctionId, function.StatusFailed); dbErr != nil {
				log.Printf("dlq: failed to update status to failed: %v", dbErr)
			}
		}
	}
	return p.base.Publish(topic, messages...)
}

func (p *statusFailPublisher) Close() error {
	return p.base.Close()
}

func decodePayload(payload []byte, dst any) error {
	if len(payload) > maxPayloadBytes {
		return fmt.Errorf("payload too large: %d bytes", len(payload))
	}
	dec := json.NewDecoder(io.LimitReader(bytes.NewReader(payload), maxPayloadBytes))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func validateObjectName(name string) error {
	if name == "" {
		return fmt.Errorf("empty object name")
	}
	if len(name) > 1024 {
		return fmt.Errorf("object name too long")
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("object name contains null byte")
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == ".." {
			return fmt.Errorf("object name contains path traversal")
		}
	}
	return nil
}
