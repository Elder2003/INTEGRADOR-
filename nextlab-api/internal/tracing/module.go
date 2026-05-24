package tracing

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/config"
)

// RegisterRoutes wires all M5 tracing endpoints.
func RegisterRoutes(
	app *fiber.App,
	db *pgxpool.Pool,
	rdb *redis.Client,
	cfg *config.Config,
	authMW fiber.Handler,
	roleGuard func(roles ...string) fiber.Handler,
) *Service {
	repo := NewRepository(db, rdb)
	hub := NewHub()
	svc := NewService(repo, rdb)

	// ─── Student — session lifecycle ──────────────────────────────────────────
	studentMW := []fiber.Handler{authMW, roleGuard("estudiante")}
	app.Post("/actividades/:aid/session", append(studentMW, openSessionHandler(svc))...)
	app.Get("/actividades/:aid/session", append(studentMW, getSessionHandler(svc))...)

	// ─── Student — WebSocket event stream ─────────────────────────────────────
	// JWT validated via query param before upgrade (research.md Decision 2)
	app.Use("/sessions/:sid/ws", wsJWTMiddleware(cfg))
	app.Get("/sessions/:sid/ws", websocket.New(makeWSHandler(hub, svc)))

	// ─── Docente & Co-docente — trace review ──────────────────────────────────
	trace := app.Group("/submissions", authMW, roleGuard("docente", "co_docente"))
	trace.Get("/:sid/trace", getTraceHandler(svc))
	trace.Get("/:sid/trace/verify", verifyTraceHandler(svc))

	return svc
}
