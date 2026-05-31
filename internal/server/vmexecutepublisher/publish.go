package vmexecutepublisher

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

func Publish(invocationId uuid.UUID, objectName, dependencyName string) error {
	cfg := config.Load()
	logger := watermill.NewStdLogger(false, false)
	publisher, err := amqp.NewPublisher(queue.PublisherConfig(*cfg), logger)
	if err != nil {
		return fmt.Errorf("failed to init publisher to build queue: %v", err)
	}
	defer func() { _ = publisher.Close() }()

	if err = publish(publisher, "execute", service.ExecuteMessage{
		InvocationId:           invocationId,
		CodeObjectName:         objectName,
		DependenciesObjectName: dependencyName,
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
