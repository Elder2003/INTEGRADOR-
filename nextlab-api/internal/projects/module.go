package projects

import (
	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/alerts"
	"github.com/Elder2003/nextlab-api/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

// Module wires together all project management services and handlers.
type Module struct {
	investigacion *InvestigacionService
	gitAnalysis   *GitAnalysisService
	defensa       *DefensaService
	repo          *Repository
}

func NewModule(db *pgxpool.Pool, mc *minio.Client, bucket string, aiSvc *ai.Service, cfg *config.Config, alertWriter alerts.AlertWriter) *Module {
	repo := NewRepository(db)
	return &Module{
		investigacion: newInvestigacionService(repo, mc, bucket, alertWriter),
		gitAnalysis:   newGitAnalysisService(repo, aiSvc, cfg),
		defensa:       newDefensaService(repo),
		repo:          repo,
	}
}

// Repository exposes the shared repo for use by the worker.
func (m *Module) Repository() *Repository {
	return m.repo
}

// RegisterRoutes wires all M10 project management routes.
func (m *Module) RegisterRoutes(
	app *fiber.App,
	authMiddleware fiber.Handler,
	roleGuard func(...string) fiber.Handler,
) {
	// ── US1: Investigacion Stages ─────────────────────────────────────────────

	// Docente-only stage management
	docente := app.Group("/actividades/:actividadId/stages",
		authMiddleware,
		roleGuard("docente"),
	)
	docente.Post("", CreateStageHandler(m.investigacion))
	docente.Put("/:stageId", UpdateStageHandler(m.investigacion))
	docente.Delete("/:stageId", DeleteStageHandler(m.investigacion))
	docente.Put("/:stageId/submissions/:studentId/grade", GradeStageSubmissionHandler(m.investigacion))

	// Read-only stage list: both docente and student
	app.Get("/actividades/:actividadId/stages",
		authMiddleware,
		ListStagesHandler(m.investigacion),
	)

	// Stage file submission: students only
	app.Post("/actividades/:actividadId/stages/:stageId/submit",
		authMiddleware,
		roleGuard("estudiante"),
		SubmitToStageHandler(m.investigacion),
	)

	// Progress overview: docente only
	app.Get("/actividades/:actividadId/stage-progress",
		authMiddleware,
		roleGuard("docente"),
		GetStageProgressHandler(m.investigacion),
	)

	// Publication gate: docente only
	app.Post("/actividades/:actividadId/publish",
		authMiddleware,
		roleGuard("docente"),
		ValidatePublishHandler(m.repo),
	)

	// ── US2: Git Analysis ─────────────────────────────────────────────────────

	// Git URL submission: students only
	app.Post("/actividades/:actividadId/git-submit",
		authMiddleware,
		roleGuard("estudiante"),
		GitSubmitHandler(m.gitAnalysis),
	)

	// Git report: docente only (students get 403 by role guard)
	app.Get("/actividades/:actividadId/git-report/:studentId",
		authMiddleware,
		roleGuard("docente"),
		GetGitReportHandler(m.gitAnalysis),
	)

	// Git grade: docente only
	app.Post("/actividades/:actividadId/git-grade/:studentId",
		authMiddleware,
		roleGuard("docente"),
		GitGradeHandler(m.gitAnalysis),
	)

	// ── US3: Oral Defense ─────────────────────────────────────────────────────

	// Schedule: docente only
	app.Put("/actividades/:actividadId/defensa/:studentId/schedule",
		authMiddleware,
		roleGuard("docente"),
		ScheduleDefenseHandler(m.defensa),
	)

	// View appointment: both roles (student sees own only — enforced in service)
	app.Get("/actividades/:actividadId/defensa/:studentId",
		authMiddleware,
		GetDefenseAppointmentHandler(m.defensa),
	)

	// Grade defense: docente only
	app.Put("/actividades/:actividadId/defensa/:studentId/grade",
		authMiddleware,
		roleGuard("docente"),
		GradeDefenseHandler(m.defensa),
	)
}
