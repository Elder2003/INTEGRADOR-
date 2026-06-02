package webhooks

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(app *fiber.App, db *pgxpool.Pool, adminAuthMW fiber.Handler) *Service {
	svc := NewService(NewRepository(db))

	app.Post("/admin/integrations/webhooks", adminAuthMW, CreateHandler(svc))
	app.Get("/admin/integrations/webhooks", adminAuthMW, ListHandler(svc))
	app.Delete("/admin/integrations/webhooks/:id", adminAuthMW, DeleteHandler(svc))
	app.Post("/admin/integrations/webhooks/:id/test", adminAuthMW, TestHandler(svc))

	return svc
}
