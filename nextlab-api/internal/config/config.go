package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL       string
	RedisAddr         string
	MinioEndpoint     string
	MinioAccessKey    string
	MinioSecretKey    string
	MinioBucket       string
	MinioUseSSL       bool
	APIPort           string
	LogLevel          string
	AppEnv            string
	JWTSecret         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	AIPrimaryURL         string
	AIPrimaryModel       string
	AICloudURL           string
	AICloudAPIKey        string
	AIPrimaryTimeoutSecs int

	AIEmbedURL       string
	AIEmbedModel     string
	AIEmbedDimension int
	RAGChunkTokens   int
	RAGChunkOverlap  int
	RAGTokenBudget   int

	HeuristicTabWeight      int
	HeuristicClipWeight     int
	HeuristicGapWeight      int
	HeuristicHighThreshold  int
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
		DatabaseURL:        getenv("DATABASE_URL", ""),
		RedisAddr:          getenv("REDIS_ADDR", ""),
		MinioEndpoint:      getenv("MINIO_ENDPOINT", ""),
		MinioAccessKey:     getenv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey:     getenv("MINIO_SECRET_KEY", ""),
		MinioBucket:        getenv("MINIO_BUCKET", "nextlab"),
		MinioUseSSL:        getenv("MINIO_USE_SSL", "false") == "true",
		APIPort:            getenv("API_PORT", "8080"),
		LogLevel:           getenv("LOG_LEVEL", "info"),
		AppEnv:             getenv("APP_ENV", "development"),
		JWTSecret:          getenv("JWT_SECRET", ""),
		AccessTokenTTL:     accessTTL,
		RefreshTokenTTL:    refreshTTL,
		GoogleClientID:     getenv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getenv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getenv("GOOGLE_REDIRECT_URL", ""),

		AIPrimaryURL:         getenv("AI_PRIMARY_URL", ""),
		AIPrimaryModel:       getenv("AI_PRIMARY_MODEL", "llama3"),
		AICloudURL:           getenv("AI_CLOUD_URL", ""),
		AICloudAPIKey:        getenv("AI_CLOUD_API_KEY", ""),
		AIPrimaryTimeoutSecs: parseInt(getenv("AI_PRIMARY_TIMEOUT_SECS", "30")),
	}

	cfg.AIEmbedURL       = getenv("AI_EMBED_URL", cfg.AICloudURL)
	cfg.AIEmbedModel     = getenv("AI_EMBED_MODEL", "text-embedding-3-small")
	cfg.AIEmbedDimension = parseInt(getenv("AI_EMBED_DIMENSION", "1536"))
	cfg.RAGChunkTokens   = parseInt(getenv("RAG_CHUNK_TOKENS", "512"))
	cfg.RAGChunkOverlap  = parseInt(getenv("RAG_CHUNK_OVERLAP", "64"))
	cfg.RAGTokenBudget   = parseInt(getenv("RAG_TOKEN_BUDGET", "2048"))

	// Detection Engine (M8) — configurable per institution to tune false-positive rates
	cfg.HeuristicTabWeight     = parseInt(getenv("HEURISTIC_TAB_WEIGHT", "5"))
	cfg.HeuristicClipWeight    = parseInt(getenv("HEURISTIC_CLIP_WEIGHT", "3"))
	cfg.HeuristicGapWeight     = parseInt(getenv("HEURISTIC_GAP_WEIGHT", "2"))
	cfg.HeuristicHighThreshold = parseInt(getenv("HEURISTIC_HIGH_THRESHOLD", "60"))

	required := map[string]string{
		"DATABASE_URL":          cfg.DatabaseURL,
		"REDIS_ADDR":            cfg.RedisAddr,
		"MINIO_ENDPOINT":        cfg.MinioEndpoint,
		"MINIO_ACCESS_KEY":      cfg.MinioAccessKey,
		"MINIO_SECRET_KEY":      cfg.MinioSecretKey,
		"JWT_SECRET":            cfg.JWTSecret,
		"GOOGLE_CLIENT_ID":      cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET":  cfg.GoogleClientSecret,
		"GOOGLE_REDIRECT_URL":   cfg.GoogleRedirectURL,
		"AI_PRIMARY_URL":        cfg.AIPrimaryURL,
	}

	for name, value := range required {
		if value == "" {
			slog.Error("missing required environment variable", "var", name)
			return nil, fmt.Errorf("missing required environment variable: %s", name)
		}
	}

	// Cloud AI vars are only required when mode is auto/cloud — warn if absent.
	if cfg.AICloudURL == "" {
		slog.Warn("AI_CLOUD_URL not set; cloud/auto routing mode will fail if activated")
	}
	if cfg.AICloudAPIKey == "" {
		slog.Warn("AI_CLOUD_API_KEY not set; cloud/auto routing mode will fail if activated")
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

func parseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func parseDuration(envKey, defaultValue string) (time.Duration, error) {
	raw := getenv(envKey, defaultValue)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s=%q: %w", envKey, raw, err)
	}
	return d, nil
}
