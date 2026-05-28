package rag

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/config"
)

func RegisterRoutes(
	app *fiber.App,
	db *pgxpool.Pool,
	minioClient *minio.Client,
	cfg *config.Config,
	authMW fiber.Handler,
	roleGuard func(roles ...string) fiber.Handler,
	aiSvc *ai.Service,
) {
	// Ensure rag-docs bucket exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, ragBucket)
	if err != nil {
		slog.Error("failed to check rag-docs bucket", "err", err)
	} else if !exists {
		if err := minioClient.MakeBucket(ctx, ragBucket, minio.MakeBucketOptions{}); err != nil {
			slog.Error("failed to create rag-docs bucket", "err", err)
		} else {
			slog.Info("created minio bucket", "bucket", ragBucket)
		}
	}

	curRepo := NewCurricularRepository(db)
	docRepo := NewDocenteRepository(db)
	embedClient := NewEmbedClient(cfg)
	ingestSvc := NewIngestService(curRepo, docRepo, embedClient, minioClient, cfg)
	retrieveSvc := NewRetrieveService(curRepo, docRepo, embedClient, cfg)

	// Curricular RAG
	curr := app.Group("/rag/curricular", authMW)
	curr.Post("/ingest", roleGuard("admin"), ingestCurricularHandler(ingestSvc))
	curr.Post("/query", retrieveCurricularHandler(retrieveSvc))
	curr.Post("/solve", roleGuard("admin"), solveProblemHandler(curRepo, embedClient, aiSvc))

	// Docente RAG
	doc := app.Group("/rag/docente", authMW, roleGuard("docente"))
	doc.Post("/ingest", ingestDocenteHandler(ingestSvc, docRepo))
	doc.Post("/query", retrieveDocenteHandler(retrieveSvc, docRepo))
}
