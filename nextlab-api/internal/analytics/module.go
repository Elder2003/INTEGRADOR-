package analytics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	repo *Repository
}

func NewModule(db *pgxpool.Pool) *Module {
	return &Module{repo: NewRepository(db)}
}

func (m *Module) RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler, roleGuard func(roles ...string) fiber.Handler) {
	mw := []fiber.Handler{jwtMiddleware, roleGuard("docente", "co_docente")}
	app.Get("/actividades/:actividadId/analytics", append(mw, ActivityAnalyticsHandler(m.repo))...)
	app.Get("/cursos/:cursoId/analytics", append(mw, CourseAnalyticsHandler(m.repo))...)

	// M17: Percentiles (US2)
	docenteMW := []fiber.Handler{jwtMiddleware, roleGuard("docente")}
	app.Get("/docente/cursos/:cid/actividades/:aid/analytics/percentiles", append(docenteMW, PercentilesHandler(m.repo))...)

	// M17: Admin analytics (US4)
	adminMW := []fiber.Handler{jwtMiddleware, roleGuard("admin")}
	app.Get("/admin/analytics/overview", append(adminMW, AdminOverviewHandler(m.repo))...)
	app.Get("/admin/analytics/integrity", append(adminMW, AdminIntegrityHandler(m.repo))...)
	app.Get("/admin/analytics/submissions", append(adminMW, AdminSubmissionsHandler(m.repo))...)
}
