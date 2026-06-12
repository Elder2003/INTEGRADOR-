package notifications

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler) *Service {
	repo := NewRepository(db)
	svc := NewService(repo, rdb)

	studentMW := []fiber.Handler{jwtMW, roleGuard("estudiante")}
	app.Get("/student/notifications/stream", append(studentMW, StreamHandler(svc))...)
	app.Get("/student/notifications", append(studentMW, ListHandler(repo))...)
	app.Post("/student/notifications/:id/read", append(studentMW, ReadHandler(repo))...)

	return svc
}
