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
}
