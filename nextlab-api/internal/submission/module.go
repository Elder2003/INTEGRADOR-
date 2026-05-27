package submission

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/alerts"
	"github.com/Elder2003/nextlab-api/internal/config"
)

// RegisterRoutes wires all M4 file storage endpoints.
func RegisterRoutes(
	app *fiber.App,
	db *pgxpool.Pool,
	mc *minio.Client,
	cfg *config.Config,
	authMW fiber.Handler,
	roleGuard func(roles ...string) fiber.Handler,
	tracingSvc TracingService,
	alertWriter alerts.AlertWriter,
) {
	repo := &Repository{db: db}
	svc := &Service{repo: repo, mc: mc, bucket: cfg.MinioBucket, tracingSvc: tracingSvc, alertWriter: alertWriter}

	// ─── Student — submission lifecycle ──────────────────────────────────────
	studentMW := []fiber.Handler{authMW, roleGuard("estudiante")}
	app.Post("/actividades/:aid/submission", append(studentMW, UploadHandler(svc))...)
	app.Put("/actividades/:aid/submission", append(studentMW, ReplaceHandler(svc))...)
	app.Delete("/actividades/:aid/submission", append(studentMW, DeleteHandler(svc))...)
	app.Get("/actividades/:aid/submission", append(studentMW, GetSubmissionHandler(svc))...)
	app.Get("/actividades/:aid/submission/download", append(studentMW, StudentDownloadHandler(svc))...)

	// ─── Docente & Co-docente — submission review ─────────────────────────────
	docente := app.Group("/cursos", authMW, roleGuard("docente", "co_docente"))
	docente.Get("/:cid/actividades/:aid/submissions", ListSubmissionsHandler(svc))
	docente.Get("/:cid/actividades/:aid/config-changes", ListConfigChangesHandler(svc))

	downloads := app.Group("/submissions", authMW, roleGuard("docente", "co_docente"))
	downloads.Get("/:sid/download", DownloadHandler(svc))
}
