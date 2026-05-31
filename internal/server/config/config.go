package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port             string
	GinMode          string
	DatabaseURL      string
	MinioEndpoint    string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioUseSSL      bool
	KeyEncryptionKey string
	MaxTokenSize     int64
	TimeInterval     int64
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/oblak"),
		MinioEndpoint:    getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:   getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:   getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSSL:      getEnv("MINIO_USE_SSL", "false") == "true",
		KeyEncryptionKey: getEnv("KEY_ENCRYPTION_KEY", "secret-key"),
		MaxTokenSize:     getEnvInt64("KEY_ENCRYPTION_KEY", 10),
		TimeInterval:     getEnvInt64("TIME_INTERVAL", 5),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}
