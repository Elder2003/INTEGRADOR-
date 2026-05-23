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
	docente := app.Group("", jwtMiddleware, roleGuard("docente", "co_docente"))
	docente.Get("/actividades/:actividadId/analytics", ActivityAnalyticsHandler(m.repo))
	docente.Get("/cursos/:cursoId/analytics", CourseAnalyticsHandler(m.repo))
}
