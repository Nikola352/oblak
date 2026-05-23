package analyzer_mock

import (
	"encoding/json"
	"fmt"
	message2 "oblak/internal/analyzer/message"
	"oblak/internal/server/function"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func StartPublisher() error {
	amqpURI := "amqp://admin:admin@localhost:5672/"
	exchangeName := "analyzer"
	cfg := publisherConfig(amqpURI, exchangeName)

	publisher, err := amqp.NewPublisher(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}
	funcMessage := createMessage()
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

func createMessage() message2.FunctionMessage {
	id, err := uuid.Parse("dde950d8-0437-4a8a-98cf-70a28096c43b")
	if err != nil {
		panic(err)
	}
	return message2.FunctionMessage{
		Path:       "clean_dependent.tar.gz",
		Bucket:     "quarantine",
		Status:     function.StatusQuarantined,
		FunctionId: id,
	}

}
