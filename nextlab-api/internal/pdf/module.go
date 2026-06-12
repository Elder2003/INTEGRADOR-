package pdf

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pdfTimeout = 15 * time.Second

func SetupRoutes(app *fiber.App, db *pgxpool.Pool, docenteAuthMW, adminAuthMW fiber.Handler) {
	svc := NewService(db)

	app.Get("/docente/cursos/:cid/actividades/:aid/report.pdf", docenteAuthMW, ActivityReportHandler(svc))
	app.Get("/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report.pdf", docenteAuthMW, DetectionReportHandler(svc))
	app.Get("/appeals/:id/audit-report.pdf", adminAuthMW, AppealAuditReportHandler(svc))
}
