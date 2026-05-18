package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	RedisAddr       string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	APIPort         string
	LogLevel        string
	AppEnv          string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func Load() (*Config, error) {
	accessTTL, err := parseDuration("ACCESS_TOKEN_TTL", "15m")
	if err != nil {
		return nil, err
	}
	refreshTTL, err := parseDuration("REFRESH_TOKEN_TTL", "168h")
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DatabaseURL:     getenv("DATABASE_URL", ""),
		RedisAddr:       getenv("REDIS_ADDR", ""),
		MinioEndpoint:   getenv("MINIO_ENDPOINT", ""),
		MinioAccessKey:  getenv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey:  getenv("MINIO_SECRET_KEY", ""),
		APIPort:         getenv("API_PORT", "8080"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		AppEnv:          getenv("APP_ENV", "development"),
		JWTSecret:       getenv("JWT_SECRET", ""),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}

	required := map[string]string{
		"DATABASE_URL":   cfg.DatabaseURL,
		"REDIS_ADDR":     cfg.RedisAddr,
		"MINIO_ENDPOINT": cfg.MinioEndpoint,
		"MINIO_ACCESS_KEY": cfg.MinioAccessKey,
		"MINIO_SECRET_KEY": cfg.MinioSecretKey,
		"JWT_SECRET":     cfg.JWTSecret,
	}

	for name, value := range required {
		if value == "" {
			slog.Error("missing required environment variable", "var", name)
			return nil, fmt.Errorf("missing required environment variable: %s", name)
		}
	}

	if len(cfg.JWTSecret) < 32 {
		slog.Error("JWT_SECRET too short", "min_length", 32, "actual_length", len(cfg.JWTSecret))
		os.Exit(1)
	}

	return cfg, nil
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(envKey, defaultValue string) (time.Duration, error) {
	raw := getenv(envKey, defaultValue)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s=%q: %w", envKey, raw, err)
	}
	return d, nil
}
