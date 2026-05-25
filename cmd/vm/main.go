package main

import (
	"fmt"
	"log"
	"oblak/internal/server/database"
	"oblak/internal/server/filestore"
	"oblak/internal/util"
	"oblak/internal/vm/config"
	"oblak/internal/vm/queue"
	"oblak/internal/vm/service"
	"oblak/internal/vm/vm"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	cfg := config.Load()

	ctx, _ := util.SigtermCancelledContext()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	minioClient, err := filestore.Connect(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}

	buildService := service.NewEnvironmentPrepareService(
		db,
		minioClient,
		vm.NewEnvironmentPrepareRunner(minioClient),
	)
	executeService := service.NewExecutionService(
		db,
		minioClient,
		vm.NewExecutionRunner(minioClient),
	)

	buildHandler := queue.NewBuildHandler(buildService)
	executeHandler := queue.NewExecuteHandler(executeService)

	logger := watermill.NewStdLogger(true, false)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		log.Fatalf("router: %v", err)
	}

	router.AddMiddleware(
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     30 * time.Second,
			Multiplier:      2,
			Logger:          logger,
		}.Middleware,
		middleware.Recoverer,
	)

	for i := range cfg.MaxConcurrentBuilds {
		sub, err := amqp.NewSubscriber(queue.PrepareEnvConsumerConfig(*cfg), logger)
		if err != nil {
			log.Fatalf("build subscriber: %v", err)
		}
		router.AddConsumerHandler(fmt.Sprintf("build-%d", i), cfg.BuildQueueName, sub, buildHandler.Handle)
	}

	for i := range cfg.MaxConcurrentExecutes {
		sub, err := amqp.NewSubscriber(queue.ExecuteConsumerConfig(*cfg), logger)
		if err != nil {
			log.Fatalf("execute subscriber: %v", err)
		}
		router.AddConsumerHandler(fmt.Sprintf("execute-%d", i), cfg.ExecuteQueueName, sub, executeHandler.Handle)
	}

	if err := router.Run(ctx); err != nil {
		log.Fatalf("router: %v", err)
	}
}
