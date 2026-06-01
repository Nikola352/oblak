package analyzerpublisher

import (
	"encoding/json"
	"fmt"
	"oblak/internal/analyzer/config"
	message2 "oblak/internal/analyzer/message"
	"oblak/internal/server/filestore"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func StartPublisher(funcMessage message2.FunctionMessage) error {
	amqpURI := config.Cfg.AMQPURI
	exchangeName := "analyzer"
	cfg := publisherConfig(amqpURI, exchangeName)

	publisher, err := amqp.NewPublisher(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	payloadBytes, err := json.Marshal(funcMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal function message struct: %w", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	err = publisher.Publish("some_topic", msg)
	if err != nil {
		return fmt.Errorf("failed to broadcast payload to rabbitmq exchange: %w", err)
	}

	return nil
}

func publisherConfig(amqpURI, exchangeName string) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: amqpURI,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: amqp.GenerateExchangeNameConstant(exchangeName),
			Type:         "fanout",
			Durable:      true,
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return ""
			},
		},

		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}

func CreateMessage(id uuid.UUID, path string) message2.FunctionMessage {
	return message2.FunctionMessage{
		Path:       path,
		Bucket:     filestore.NewBucketRegistry().Name(filestore.QuarantineBucket),
		FunctionId: id,
	}
}
