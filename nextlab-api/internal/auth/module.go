package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/config"
)

func RegisterRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) {
	svc := NewService(db, rdb, cfg)

	g := app.Group("/auth")

	// US1: Login
	g.Post("/login", LoginHandler(svc))

	// US2: Session management
	g.Post("/refresh", RefreshHandler(svc))
	g.Post("/logout", AuthMiddleware(cfg), LogoutHandler(svc))

	// US3: Identity
	g.Get("/me", AuthMiddleware(cfg), MeHandler())

	// US4: Password recovery
	g.Post("/forgot-password", ForgotPasswordHandler(svc))
	g.Post("/reset-password", ResetPasswordHandler(svc))
}
