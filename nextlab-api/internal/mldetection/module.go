package mldetection

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(app *fiber.App, db *pgxpool.Pool, modelDir, dbDSN string, adminAuthMW fiber.Handler) *Service {
	repo := NewRepository(db)

	// Attempt to load the active model from DB on startup
	var runtime *ONNXRuntime
	if meta, err := repo.GetActive(context.Background()); err == nil && meta != nil {
		runtime, _ = Load(meta.FilePath, meta.FeatureNames)
	}

	svc := NewService(repo, runtime, modelDir, dbDSN)
	app.Post("/admin/detection/retrain", adminAuthMW, RetrainHandler(svc))
	return svc
}
