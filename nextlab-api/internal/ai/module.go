package ai

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Elder2003/nextlab-api/internal/config"
)

// RegisterRoutes wires all AI router endpoints and starts the background mode cache.
func RegisterRoutes(
	app *fiber.App,
	db *pgxpool.Pool,
	cfg *config.Config,
	authMW fiber.Handler,
	roleGuard func(roles ...string) fiber.Handler,
) {
	repo := NewRepository(db)
	svc := NewService(repo, cfg)

	// Ensure a default mode row exists before starting.
	svc.SeedIfEmpty(context.Background())

	// Start background 3-second mode cache refresh.
	svc.StartCacheRefresh(context.Background())

	timeoutSecs := cfg.AIPrimaryTimeoutSecs
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}

	// Any authenticated user: dispatch + read config.
	internal := app.Group("/ai", authMW)
	internal.Post("/complete", routeHandler(svc, timeoutSecs))
	internal.Get("/config", getConfigHandler(svc))

	// Admin only: write config + history.
	admin := app.Group("/ai", authMW, roleGuard("admin"))
	admin.Put("/config", putConfigHandler(svc))
	admin.Get("/config/history", getConfigHistoryHandler(svc))
}
