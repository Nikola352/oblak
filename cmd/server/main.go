package main

import (
	"context"
	"log"
	"oblak/internal/server/httpserver"

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

	if err := bucketInitializer.InitializeBuckets(ctx); err != nil {
		log.Fatalf("Failed to initialize buckets: %v", err)
	}
	server := httpserver.New(db, minioClient, cfg.KeyEncryptionKey)
	if err := server.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
