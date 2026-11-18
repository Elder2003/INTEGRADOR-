package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/academic"
	"github.com/Elder2003/nextlab-api/internal/admin"
	"github.com/Elder2003/nextlab-api/internal/appeals"
	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/alerts"
	"github.com/Elder2003/nextlab-api/internal/analytics"
	"github.com/Elder2003/nextlab-api/internal/auth"
	"github.com/Elder2003/nextlab-api/internal/config"
	"github.com/Elder2003/nextlab-api/internal/detection"
	"github.com/Elder2003/nextlab-api/internal/health"
	"github.com/Elder2003/nextlab-api/internal/baseline"
	"github.com/Elder2003/nextlab-api/internal/mldetection"
	"github.com/Elder2003/nextlab-api/internal/notifications"
	"github.com/Elder2003/nextlab-api/internal/pdf"
	"github.com/Elder2003/nextlab-api/internal/webhooks"
	"github.com/Elder2003/nextlab-api/internal/projects"
	"github.com/Elder2003/nextlab-api/internal/rag"
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
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")
		},
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Authorization",
		AllowCredentials: false,
	}))
	app.Use(correlationIDMiddleware())

	// health checker initialized after embedClient is available (see below)
	var healthChecker *health.Checker

	app.Get("/health", func(c *fiber.Ctx) error {
		if healthChecker == nil {
			return c.Status(503).JSON(fiber.Map{"status": "starting"})
		}
		report := healthChecker.Check(c.Context())
		c.Status(report.StatusCode())
		return c.JSON(report)
	})

	// Alerts & Analytics (M11)
	alertsModule := alerts.NewModule(pgPool, redisClient)
	alertsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)
	analyticsModule := analytics.NewModule(pgPool)
	analyticsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// M17: Student notifications
	notifSvc := notifications.SetupRoutes(app, pgPool, redisClient, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// M17: Webhooks (US8)
	webhooksSvc := webhooks.SetupRoutes(app, pgPool, chainMW(auth.AuthMiddleware(cfg), auth.RoleGuard("admin")))

	// M17: PDF reports (US3)
	pdf.SetupRoutes(app, pgPool,
		chainMW(auth.AuthMiddleware(cfg), auth.RoleGuard("docente")),
		chainMW(auth.AuthMiddleware(cfg), auth.RoleGuard("admin")),
	)

	auth.RegisterRoutes(app, pgPool, redisClient, cfg)
	academic.RegisterRoutes(app, pgPool, redisClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)
	tracingSvc := tracing.RegisterRoutes(app, pgPool, redisClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)
	submission.RegisterRoutes(app, pgPool, minioClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard, tracingSvc, alertsModule.Service())

	// AI service created early so it can be shared with RAG and Detection
	aiRepo := ai.NewRepository(pgPool)
	aiSvc := ai.NewService(aiRepo, cfg)
	ai.RegisterRoutes(app, pgPool, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// Detection Engine (M8) — embedClient and rate limiter created before RAG routes
	detectionRepo := detection.NewDetectionRepository(pgPool)
	heuristicEngine := detection.NewHeuristicEngine(cfg)
	embedClient := rag.NewEmbedClient(cfg)
	if err := rag.ValidateDimension(ctx, embedClient, cfg.AIEmbedDimension); err != nil {
		slog.Error("FATAL embed dimension mismatch", "err", err)
		os.Exit(1)
	}
	aiRateLimiter := ai.NewAIRateLimiter(redisClient, cfg.AIRateLimitRequests, cfg.AIRateLimitWindowSec)
	healthChecker = health.NewChecker(pgPool, redisClient, minioClient, cfg.AIPrimaryURL, cfg.AIPrimaryModel, embedClient)
	// mlSvc is wired to health after detection setup (below)
	rag.RegisterRoutes(app, pgPool, minioClient, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard, aiSvc, aiRateLimiter)
	ragSvc := rag.NewRetrieveService(
		rag.NewCurricularRepository(pgPool),
		rag.NewDocenteRepository(pgPool),
		embedClient,
		cfg,
	)
	baselineSvc := baseline.NewService(baseline.NewRepository(pgPool))
	mlSvc := mldetection.SetupRoutes(app, pgPool, cfg.MLModelDir, cfg.DatabaseURL,
		chainMW(auth.AuthMiddleware(cfg), auth.RoleGuard("admin")))
	detectionSvc := detection.NewDetectionService(detectionRepo, heuristicEngine, aiSvc, ragSvc, minioClient, cfg.MinioBucket, cfg, alertsModule.Service())
	detectionSvc.SetNotificationPublisher(notifSvc)
	detectionSvc.SetBaselineService(baselineSvc)
	detectionSvc.SetWebhookTrigger(webhooksSvc)
	detectionSvc.SetMLRuntime(mlSvc.GetRuntime())
	// A trained ML artifact is optional in the lightweight presentation stack.
	// The routes remain available, but its absence must not make the demo unhealthy.
	if cfg.AppEnv != "demo" {
		healthChecker.SetMLMetaProvider(&mlHealthAdapter{svc: mlSvc})
	}
	detection.RegisterRoutes(app, detectionSvc, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// Project Management (M10)
	projectsModule := projects.NewModule(pgPool, minioClient, cfg.MinioBucket, aiSvc, cfg, alertsModule.Service())
	projectsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// Admin Modules (M12)
	adminModule := admin.NewModule(pgPool, minioClient, cfg.MinioBucket, cfg.AppEnv)
	adminModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)

	// Appeals & Final Audit (M13)
	appealsModule := appeals.NewModule(pgPool)
	appealsModule.SetNotificationPublisher(notifSvc)
	appealsModule.SetWebhookTrigger(webhooksSvc)
	appealsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)

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

// mlHealthAdapter adapts *mldetection.Service to health.MLMetaProvider.
type mlHealthAdapter struct{ svc *mldetection.Service }

func (a *mlHealthAdapter) GetActiveMeta() health.MLModelMeta {
	m := a.svc.GetActiveMeta()
	if m == nil {
		return health.MLModelMeta{}
	}
	return health.MLModelMeta{Available: true, Version: m.Version, TrainedAt: m.TrainedAt.Format("2006-01-02")}
}

// chainMW combines two middleware handlers into one for use as a single fiber.Handler.
func chainMW(first, second fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := first(c); err != nil {
			return err
		}
		if c.Response().StatusCode() >= 400 {
			return nil
		}
		return second(c)
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
