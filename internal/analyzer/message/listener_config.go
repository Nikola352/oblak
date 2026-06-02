package message

import "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"

func consumerConfig(amqpURI, exchangeName, queueName string) amqp.Config {
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
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameConstant(queueName),
			Durable:      true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return ""
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
