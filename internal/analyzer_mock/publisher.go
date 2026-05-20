package analyzer_mock

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	_ "github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/ThreeDotsLabs/watermill/message"
)

func StartPublisher() error {
	amqpURI := "amqp://admin:admin@localhost:5672/"
	exchangeName := "analyzer"
	cfg := publisherConfig(amqpURI, exchangeName)

	publisher, err := amqp.NewPublisher(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, Fanout!"))

	// 3. Publish to the exchange
	// Note: In fanout, the "topic" string below is ignored by RabbitMQ,
	// but it is still required by the Watermill API.
	err = publisher.Publish("some_topic", msg)
	if err != nil {
		panic(err)
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
