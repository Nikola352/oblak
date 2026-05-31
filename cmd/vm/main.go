package main

import (
	"log"
	"oblak/internal/function"
	"oblak/internal/invocation"
	"oblak/internal/server/database"
	"oblak/internal/server/filestore"
	"oblak/internal/util"
	"oblak/internal/vm/config"
	"oblak/internal/vm/queue"
	"oblak/internal/vm/service"
	"oblak/internal/vm/vm"

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

	functionStore := function.NewStore(db)
	invocationStore := invocation.NewStore(db)

	buildService := service.NewEnvironmentPrepareService(functionStore, vm.NewEnvironmentPrepareRunner(minioClient))
	executeService := service.NewExecutionService(
		invocationStore,
		vm.NewExecutionRunner(minioClient),
	)

	buildHandler := queue.NewBuildHandler(buildService, invocationStore)
	executeHandler := queue.NewExecuteHandler(executeService, invocationStore)

	router, err := queue.NewVmRouter(*cfg, buildHandler, executeHandler, functionStore, invocationStore)
	if err != nil {
		log.Fatalf("router setup: %v", err)
	}
	defer func() { _ = router.Close() }()

	if err := router.Run(ctx); err != nil {
		log.Fatalf("router: %v", err)
	}
}
