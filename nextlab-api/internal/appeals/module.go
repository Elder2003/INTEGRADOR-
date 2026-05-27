package appeals

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

func (m *Module) RegisterRoutes(app *fiber.App, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler) {
	// US1 — filing & listing
	app.Post("/appeals", jwtMW, FileAppealHandler(m.repo))
	app.Get("/appeals/:id", jwtMW, GetAppealHandler(m.repo))
	app.Get("/appeals", jwtMW, ListAppealsHandler(m.repo))
	app.Get("/actividades/:aid/appeals", jwtMW, ListActivityAppealsHandler(m.repo))

	// US3 — review workflow
	app.Put("/appeals/:id/review", jwtMW, ReviewAppealHandler(m.repo))
	app.Put("/appeals/:id/escalate", jwtMW, EscalateAppealHandler(m.repo))
	app.Put("/appeals/:id/admin-review", jwtMW, roleGuard("admin"), AdminReviewAppealHandler(m.repo))

	// US4 — evidence chain
	app.Get("/appeals/:id/evidence", jwtMW, GetEvidenceHandler(m.repo))
}
