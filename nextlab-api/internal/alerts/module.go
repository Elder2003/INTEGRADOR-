package alerts

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Module struct {
	repo *Repository
	svc  *Service
	rdb  *redis.Client
}

func NewModule(db *pgxpool.Pool, rdb *redis.Client) *Module {
	repo := NewRepository(db, rdb)
	svc := NewService(repo, rdb)
	return &Module{repo: repo, svc: svc, rdb: rdb}
}

// Service returns the AlertWriter-compatible service for injection into other modules.
func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) RegisterRoutes(app *fiber.App, jwtMiddleware fiber.Handler, roleGuard func(roles ...string) fiber.Handler) {
	docente := app.Group("/alerts", jwtMiddleware, roleGuard("docente", "co_docente"))
	docente.Post("/stream-token", StreamTokenHandler(m.svc))
	docente.Get("", GetAlertsHandler(m.repo))
	docente.Patch("/:id/read", MarkReadHandler(m.repo))

	// SSE stream uses short-lived token — no standard JWT middleware
	app.Get("/alerts/stream", SSEStreamHandler(m.svc, m.repo, m.rdb))
}
