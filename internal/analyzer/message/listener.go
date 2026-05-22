package message

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/analyzer/av"
	"oblak/internal/analyzer/dast"
	"oblak/internal/analyzer/llm"
	orchestrator2 "oblak/internal/analyzer/orchestrator"
	"oblak/internal/analyzer/sast"
	"oblak/internal/server/database"
	"oblak/internal/server/function"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func StartListener(ctx context.Context) error {
	queueName := "analyzer_data"
	amqpURI := "amqp://admin:admin@localhost:5672/"
	exchangeName := "analyzer"
	topicName := "analyzer"

	cfg := consumerConfig(amqpURI, exchangeName, queueName)

	var myJudge llm.JudgeLLM = llm.NewQwenJudge("http://localhost:11434")
	var semgrepAnalyzer sast.StaticAnalyzer = sast.NewSemgrepAnalyzer("/home/nikolavelemir/faks/rbs/oblak/.venv/bin/semgrep")
	var clamAV av.Antivirus = av.NewClamAV("tcp://localhost:3310")
	gvisorBox, err := dast.NewGVisorBox()

	db, err := database.Connect(context.Background(), "postgres://postgres:postgres@localhost:5433/oblak")
	var functionStore = function.NewStore(db)

	if err != nil {
		panic(err)
	}
	orchestrator := orchestrator2.NewOrchestrator(clamAV, myJudge, semgrepAnalyzer, gvisorBox)

	sub, err := amqp.NewSubscriber(cfg, watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	msgs, err := sub.Subscribe(context.Background(), topicName)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	go func() {
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
				err := processMessage(ctx, msg, orchestrator, functionStore)
				if err != nil {
					panic(err)
				}
				return
			case <-ctx.Done():
				log.Printf("Shutting down")
				err := sub.Close()
				if err != nil {
					panic(err)
				}
				return
			}
		}
	}()
	return nil
}
func processMessage(ctx context.Context, msg *message.Message, ao *orchestrator2.AnalysisOrchestrator, functionStore *function.Store) error {
	log.Println("Stiglo!")
	id := uuid.New()
	err := onLand(id, ctx, functionStore)
	if err != nil {
		return err
	}
	fileName := string(msg.Payload)
	fileName = "clean.tar.gz"
	verdict, err := ao.AnalyzeFile(ctx, fileName)

	if err != nil {
		return err
	}

	return updateFunctionStatus(id, verdict, ctx, functionStore)

}

func updateFunctionStatus(id uuid.UUID, verdict orchestrator2.AnalysisVerdict, ctx context.Context, store *function.Store) error {
	var status = function.StatusDetected
	if verdict == orchestrator2.MALICIOUS {
		status = function.StatusDetected
	}
	return store.UpdateFunctionStatus(ctx, id, status)
}
func onLand(id uuid.UUID, ctx context.Context, store *function.Store) error {
	var onLandStatus = function.StatusScanning
	return store.UpdateFunctionStatus(ctx, id, onLandStatus)
}
