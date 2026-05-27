package queue

import (
	"oblak/internal/vm/config"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/rabbitmq/amqp091-go"
)

func PrepareEnvConsumerConfig(cfg config.Config) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: cfg.AmqpUri,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: amqp.GenerateExchangeNameConstant(cfg.AmqpVmExchangeName),
			Type:         "direct",
			Durable:      true,
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameConstant(cfg.BuildQueueName),
			Durable:      true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return cfg.BuildQueueName
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}

func ExecuteConsumerConfig(cfg config.Config) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: cfg.AmqpUri,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: amqp.GenerateExchangeNameConstant(cfg.AmqpVmExchangeName),
			Type:         "direct",
			Durable:      true,
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameConstant(cfg.ExecuteQueueName),
			Durable:      true,
			Arguments: amqp091.Table{
				"x-queue-ttl":  int32(60 * 1000),
				"x-max-length": int32(2000),
				"x-overflow":   "reject-publish",
			},
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return cfg.ExecuteQueueName
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}

func PublisherConfig(cfg config.Config) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: cfg.AmqpUri,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: amqp.GenerateExchangeNameConstant(cfg.AmqpVmExchangeName),
			Type:         "direct",
			Durable:      true,
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return topic
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}
