package main

import (
	"context"
	"log"
	"oblak/internal/function"
	"oblak/internal/invocation"
	"oblak/internal/server/events"
	"oblak/internal/server/extractor"
	"oblak/internal/server/httpserver"
	"oblak/internal/server/invocations"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"oblak/internal/server/config"
	"oblak/internal/server/database"
	"oblak/internal/server/filestore"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	ctx := context.Background()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	minioClient, err := filestore.Connect(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}

	bucketRegistry := filestore.NewBucketRegistry()
	bucketInitializer := filestore.NewBucketInitializer(minioClient, bucketRegistry)

	invocationLogStore := invocations.NewInvocationLogStore(minioClient, "oblak-logs")

	functionStore := function.NewStore(db)
	invocationStore := invocation.NewStore(db)

	quarantineBus := events.NewBus[events.QuarantineEvent]()
	extractionBus := events.NewBus[events.ExtractionEvent]()

	service := extractor.NewService(db, minioClient, functionStore, quarantineBus, extractionBus)
	service.StartQuarantineWorker()
	service.StartExtractionWorker()

	if err := bucketInitializer.InitializeBuckets(ctx); err != nil {
		log.Fatalf("Failed to initialize buckets: %v", err)
	}
	server := httpserver.New(db, minioClient, cfg.KeyEncryptionKey, quarantineBus, extractionBus, functionStore, invocationStore, invocationLogStore)
	if err := server.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
