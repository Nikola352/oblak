package vmpublish

import (
	"encoding/json"
	"fmt"
	"oblak/internal/vm/config"
	"oblak/internal/vm/queue"
	"oblak/internal/vm/service"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func Publish(functionId uuid.UUID, objectName string) error {
	cfg := config.Load()
	logger := watermill.NewStdLogger(false, false)
	publisher, err := amqp.NewPublisher(queue.PublisherConfig(*cfg), logger)
	if err != nil {
		return fmt.Errorf("failed to init publisher to build queue: %v", err)
	}
	defer func() { _ = publisher.Close() }()

	if err = publish(publisher, "build", service.BuildMessage{
		FunctionId:     functionId,
		CodeObjectName: objectName,
	}); err != nil {
		return fmt.Errorf("failed to publish to build queue: %v", err)
	}

	return nil
}

func publish(pub *amqp.Publisher, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return pub.Publish(topic, message.NewMessage(watermill.NewUUID(), body))
}
