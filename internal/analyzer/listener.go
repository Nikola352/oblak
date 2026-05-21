package analyzer

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/analyzer/av"
	"oblak/internal/analyzer/dast"
	"oblak/internal/analyzer/llm"
	"oblak/internal/analyzer/sast"

	"github.com/ThreeDotsLabs/watermill"
	_ "github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	_ "github.com/ThreeDotsLabs/watermill/message"
)

func StartListener(ctx context.Context) error {
	queueName := "analyzer_data"
	amqpURI := "amqp://admin:admin@localhost:5672/"
	exchangeName := "analyzer"
	topicName := "analyzer"

	cfg := consumerConfig(amqpURI, exchangeName, queueName)

	var myJudge llm.JudgeLLM = llm.NewQwenJudge("http://localhost:11434")
	var semgrepAnalyzer sast.StaticAnalyzer = sast.NewSemgrepAnalyzer("/home/nikola-velemir/faks/rbs/oblak/.venv/bin/semgrep")
	var clamAV av.Antivirus = av.NewClamAV("tcp://localhost:3310")
	var gvisorBox dast.DetonationBox = dast.NewGVisorBox()

	orchestrator := NewOrchestrator(clamAV, myJudge, semgrepAnalyzer, gvisorBox)

	sub, err := amqp.NewSubscriber(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	msgs, err := sub.Subscribe(context.Background(), topicName)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	go func() {
		// 1. Listen for errors on the subscriber itself
		// Some errors (like connection drops) are sent to the error channel
		// if you had one configured, but let's at least log the routine start.
		log.Println("Starting message consumption loop...")

		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					log.Printf("message channel closed")
					return
				}
				log.Printf("Received message: %s", msg.UUID)
				msg.Ack()
				processMessage(ctx, msg, orchestrator)
			case <-ctx.Done():
				log.Printf("Shutting down")
				sub.Close()
				return
			}
		}
	}()
	return nil
}
func processMessage(ctx context.Context, msg *message.Message, ao *AnalysisOrchestrator) error {
	log.Println("Stiglo!")
	fileName := string(msg.Payload)
	fileName = "clean.py"
	err := ao.AnalyzeFile(ctx, fileName)
	if err != nil {
		return err
	}
	return nil
}
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
