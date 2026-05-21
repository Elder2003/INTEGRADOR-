package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/academic"
	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/auth"
	"github.com/Elder2003/nextlab-api/internal/config"
	"github.com/Elder2003/nextlab-api/internal/health"
	"github.com/Elder2003/nextlab-api/internal/submission"
	"github.com/Elder2003/nextlab-api/internal/tracing"
	embeddedmigrations "github.com/Elder2003/nextlab-api/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	// Connect to PostgreSQL (pool for concurrent bcrypt-heavy requests)
	pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	// Run migrations
	d, err := iofs.New(embeddedmigrations.FS, ".")
	if err != nil {
		slog.Error("failed to load migrations", "err", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize migrations", "err", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	version, _, _ := m.Version()
	slog.Info("migrations applied", slog.Uint64("version", uint64(version)), slog.String("correlation_id", "system"))

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctxTimeout).Err(); err != nil {
		slog.Error("failed to connect to redis", "addr", cfg.RedisAddr, "err", err)
		os.Exit(1)
	}

	// Connect to MinIO
	minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		slog.Error("failed to initialize minio", "err", err)
		os.Exit(1)
	}

	// Ensure the submissions bucket exists
	exists, err := minioClient.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		slog.Error("failed to check minio bucket", "bucket", cfg.MinioBucket, "err", err)
		os.Exit(1)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
			slog.Error("failed to create minio bucket", "bucket", cfg.MinioBucket, "err", err)
			os.Exit(1)
		}
	}
	slog.Info("minio ready", slog.String("bucket", cfg.MinioBucket), slog.String("correlation_id", "system"))

	app := fiber.New()
	app.Use(correlationIDMiddleware())

	healthChecker := health.NewChecker(pgPool, redisClient, minioClient)

	app.Get("/health", func(c *fiber.Ctx) error {
		report := healthChecker.Check(c.Context())
		c.Status(report.StatusCode())
		return c.JSON(report)
	})

	auth.RegisterRoutes(app, pgPool, redisClient, cfg)
	academic.RegisterRoutes(app, pgPool, redisClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)
	tracingSvc := tracing.RegisterRoutes(app, pgPool, redisClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)
	submission.RegisterRoutes(app, pgPool, minioClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard, tracingSvc)
	ai.RegisterRoutes(app, pgPool, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)

	slog.Info("server started",
		slog.String("service", "api"),
		slog.String("port", cfg.APIPort),
		slog.String("correlation_id", "system"),
	)

	if err := app.Listen(":" + cfg.APIPort); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func correlationIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		correlationID := generateCorrelationID()
		c.Locals("correlation_id", correlationID)
		return c.Next()
	}
}

func generateCorrelationID() string {
	return fmt.Sprintf("%d-%d", time.Now().Unix(), time.Now().Nanosecond())
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
