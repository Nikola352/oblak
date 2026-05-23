package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"oblak/internal/analyzer/audit"
	"oblak/internal/analyzer/av"
	"oblak/internal/analyzer/config"
	"oblak/internal/analyzer/dast"
	"oblak/internal/analyzer/llm"
	orchestrator2 "oblak/internal/analyzer/orchestrator"
	"oblak/internal/analyzer/sast"
	"oblak/internal/server/database"
	"oblak/internal/server/filestore"
	"oblak/internal/server/function"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type ListenerDependencies struct {
	BucketRegistry *filestore.BucketRegistry
	FileStore      *orchestrator2.FileStore
	FunctionStore  *function.Store
	Orchestrator   *orchestrator2.AnalysisOrchestrator
}

func StartListener(ctx context.Context) error {

	deps, err := wireDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed wiring application components: %w", err)
	}

	const (
		queueName    = "analyzer_data"
		amqpURI      = "amqp://admin:admin@localhost:5672/"
		exchangeName = "analyzer"
		topicName    = "analyzer"
	)

	cfg := consumerConfig(amqpURI, exchangeName, queueName)

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
				err := processMessage(ctx, msg, deps)
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
func processMessage(ctx context.Context, msg *message.Message, deps *ListenerDependencies) error {
	log.Println("Message received from queue, processing payload!")
	payload, err := unmarshalMessage(msg)
	if err != nil {
		return err
	}
	err = onLand(payload.FunctionId, ctx, deps.FunctionStore)
	if err != nil {
		return err
	}
	fileName := payload.Path
	verdict, err := deps.Orchestrator.AnalyzeFile(ctx, fileName)

	if err != nil {
		return err
	}

	err = updateFunctionStatus(payload.FunctionId, verdict, ctx, deps.FunctionStore)
	if err != nil {
		return err
	}
	err = deps.FileStore.Move(ctx, fileName, deps.BucketRegistry.Name(filestore.FunctionsBucket))

	return err
}
func unmarshalMessage(msg *message.Message) (*FunctionMessage, error) {
	var msgData FunctionMessage
	if err := json.Unmarshal(msg.Payload, &msgData); err != nil {
		log.Printf("[QUEUE ERROR] Failed parsing JSON payload payload structures: %v", err)
		return nil, fmt.Errorf("malformed function message metadata format: %w", err)
	}
	return &msgData, nil
}
func updateFunctionStatus(id uuid.UUID, verdict orchestrator2.AnalysisVerdict, ctx context.Context, store *function.Store) error {
	var status = function.StatusReady
	if verdict == orchestrator2.MALICIOUS {
		status = function.StatusDetected
	}
	return store.UpdateFunctionStatus(ctx, id, status)
}
func onLand(id uuid.UUID, ctx context.Context, store *function.Store) error {
	var onLandStatus = function.StatusScanning
	return store.UpdateFunctionStatus(ctx, id, onLandStatus)
}

func wireDependencies(ctx context.Context) (*ListenerDependencies, error) {
	bucketRegistry := filestore.NewBucketRegistry()
	var myJudge llm.JudgeLLM = llm.NewQwenJudge(config.Cfg.OllamaURL)
	var semgrepAnalyzer sast.StaticAnalyzer = sast.NewSemgrepAnalyzer(config.Cfg.SastBinaryPath)

	var clamAV av.Antivirus = av.NewClamAV(config.Cfg.AntivirusURL)
	var auditor audit.DependencyAuditor = audit.NewPipAuditor(config.Cfg.AuditBinaryPath)

	gvisorBox, err := dast.NewGVisorBox()
	if err != nil {
		return nil, fmt.Errorf("failed bootstrapping gvisor runtime container layout: %w", err)
	}

	fileStore := orchestrator2.NewFileStore(config.Cfg.Minio.Endpoint, config.Cfg.Minio.AccessKey, config.Cfg.Minio.SecretKey, bucketRegistry.Name(filestore.QuarantineBucket))

	db, err := database.Connect(ctx, config.Cfg.DbConnectionString)
	if err != nil {
		return nil, fmt.Errorf("database connection initialization failed: %w", err)
	}
	functionStore := function.NewStore(db)

	// 3. Construct Composite Orchestrator Aggregator
	orchestrator := orchestrator2.NewOrchestrator(clamAV, myJudge, semgrepAnalyzer, gvisorBox, auditor, fileStore)

	return &ListenerDependencies{
		Orchestrator:   orchestrator,
		FunctionStore:  functionStore,
		FileStore:      fileStore,
		BucketRegistry: bucketRegistry,
	}, nil
}
