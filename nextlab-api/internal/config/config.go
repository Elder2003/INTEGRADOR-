package config

import (
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	DatabaseURL   string
	RedisAddr     string
	MinioEndpoint string
	MinioAccessKey string
	MinioSecretKey string
	APIPort       string
	LogLevel      string
	AppEnv        string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:    getenv("DATABASE_URL", ""),
		RedisAddr:      getenv("REDIS_ADDR", ""),
		MinioEndpoint:  getenv("MINIO_ENDPOINT", ""),
		MinioAccessKey: getenv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey: getenv("MINIO_SECRET_KEY", ""),
		APIPort:        getenv("API_PORT", "8080"),
		LogLevel:       getenv("LOG_LEVEL", "info"),
		AppEnv:         getenv("APP_ENV", "development"),
	}

	// Validate required fields
	required := map[string]string{
		"DATABASE_URL":       cfg.DatabaseURL,
		"REDIS_ADDR":         cfg.RedisAddr,
		"MINIO_ENDPOINT":     cfg.MinioEndpoint,
		"MINIO_ACCESS_KEY":   cfg.MinioAccessKey,
		"MINIO_SECRET_KEY":   cfg.MinioSecretKey,
	}

	for name, value := range required {
		if value == "" {
			slog.Error("missing required environment variable", "var", name)
			return nil, fmt.Errorf("missing required environment variable: %s", name)
		}
	}

	return cfg, nil
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
