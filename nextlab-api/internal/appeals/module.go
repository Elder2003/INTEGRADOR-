package appeals

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationPublisher is implemented by *notifications.Service.
type NotificationPublisher interface {
	Publish(studentID uuid.UUID, notifType string, payload any) error
}

// WebhookTrigger is implemented by *webhooks.Service.
type WebhookTrigger interface {
	TriggerAsync(eventType string, data any)
}

type Module struct {
	repo           *Repository
	notifSvc       NotificationPublisher
	webhookTrigger WebhookTrigger
}

func NewModule(db *pgxpool.Pool) *Module {
	return &Module{repo: NewRepository(db)}
}

// SetNotificationPublisher wires the notification service after construction.
func (m *Module) SetNotificationPublisher(p NotificationPublisher) {
	m.notifSvc = p
}

// SetWebhookTrigger wires the webhook trigger after construction.
func (m *Module) SetWebhookTrigger(w WebhookTrigger) {
	m.webhookTrigger = w
}

func (m *Module) RegisterRoutes(app *fiber.App, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler) {
	// US1 — filing & listing
	app.Post("/appeals", jwtMW, FileAppealHandler(m.repo))
	app.Get("/appeals/:id", jwtMW, GetAppealHandler(m.repo))
	app.Get("/appeals", jwtMW, ListAppealsHandler(m.repo))
	app.Get("/actividades/:aid/appeals", jwtMW, ListActivityAppealsHandler(m.repo))

	// US3 — review workflow (pass notifSvc for appeal_updated notifications)
	app.Put("/appeals/:id/review", jwtMW, ReviewAppealHandler(m.repo, m.notifSvc, m.webhookTrigger))
	app.Put("/appeals/:id/escalate", jwtMW, EscalateAppealHandler(m.repo, m.notifSvc, m.webhookTrigger))
	app.Put("/appeals/:id/admin-review", jwtMW, roleGuard("admin"), AdminReviewAppealHandler(m.repo, m.notifSvc, m.webhookTrigger))

	// US4 — evidence chain
	app.Get("/appeals/:id/evidence", jwtMW, GetEvidenceHandler(m.repo))
}
