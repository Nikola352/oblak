package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"oblak/internal/config"
	"oblak/internal/database"
	"oblak/internal/filestore"
	"oblak/internal/server"
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

	srv := server.New(db, minioClient)
	if err := srv.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
