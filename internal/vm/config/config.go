package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL           string
	MinioEndpoint         string
	MinioAccessKey        string
	MinioSecretKey        string
	MinioUseSSL           bool
	AmqpUri               string
	AmqpVmExchangeName    string
	BuildQueueName        string
	BuildDLQName          string
	ExecuteQueueName      string
	ExecuteDLQName        string
	MaxConcurrentBuilds   int
	MaxConcurrentExecutes int
	MaxConcurrentVMs      int
}

func Load() *Config {
	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/oblak"),
		MinioEndpoint:         getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:        getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:        getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:           getEnv("MINIO_USE_SSL", "false") == "true",
		AmqpUri:               getEnv("AMQP_URI", "amqp://admin:admin@localhost:5672/"),
		AmqpVmExchangeName:    getEnv("AMQP_VM_EXCHANGE_NAME", "vm"),
		BuildQueueName:        getEnv("BUILD_QUEUE_NAME", "build"),
		BuildDLQName:          getEnv("BUILD_DLQ_NAME", "build_dlq"),
		ExecuteQueueName:      getEnv("EXECUTE_QUEUE_NAME", "execute"),
		ExecuteDLQName:        getEnv("EXECUTE_DLQ_NAME", "execute_dlq"),
		MaxConcurrentBuilds:   getEnvInt("MAX_CONCURRENT_BUILDS", 3),
		MaxConcurrentExecutes: getEnvInt("MAX_CONCURRENT_EXECUTES", 20),
		MaxConcurrentVMs:      getEnvInt("MAX_CONCURRENT_VMS", 20),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
