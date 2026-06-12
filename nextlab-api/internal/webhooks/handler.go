package webhooks

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var validEvents = map[string]bool{
	"grade.published": true,
	"appeal.updated":  true,
	"session.opened":  true,
}

func CreateHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, err := parseCallerID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req CreateWebhookRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
		}
		if req.URL == "" || req.Secret == "" || len(req.Events) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_fields"})
		}
		if len(req.Secret) < 16 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "secret_too_short", "message": "Secret must be at least 16 characters",
			})
		}
		if err := ValidateURL(req.URL); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "invalid_url", "message": err.Error(),
			})
		}
		for _, evt := range req.Events {
			if !validEvents[evt] {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
					"error": "invalid_events", "message": "Unknown event type: " + evt,
				})
			}
		}

		wh := &WebhookConfig{
			URL:        req.URL,
			SecretHmac: req.Secret, // stored as plaintext in this implementation
			Events:     req.Events,
			CreatedBy:  &callerID,
		}
		if err := svc.repo.Create(c.Context(), wh); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"url":        wh.URL,
			"events":     wh.Events,
			"is_active":  true,
			"created_at": time.Now(),
		})
	}
}

func ListHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		webhooks, err := svc.repo.ListActive(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if webhooks == nil {
			webhooks = []WebhookConfig{}
		}
		return c.JSON(webhooks)
	}
}

func DeleteHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}
		if err := svc.repo.SoftDelete(c.Context(), id); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func TestHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		wh, err := svc.repo.GetByID(c.Context(), id)
		if err != nil || wh == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "webhook_not_found"})
		}

		testPayload := []byte(`{"event":"test","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `","data":{}}`)
		start := time.Now()
		statusCode, postErr := svc.doPost(wh.URL, Sign(testPayload, wh.SecretHmac), testPayload)
		latency := time.Since(start).Milliseconds()

		ok := postErr == nil && statusCode >= 200 && statusCode < 300
		resp := fiber.Map{"ok": ok, "latency_ms": latency}
		if ok {
			resp["status_code"] = statusCode
		} else {
			errMsg := "connection failed"
			if postErr != nil {
				errMsg = postErr.Error()
			}
			resp["error"] = errMsg
			resp["status_code"] = statusCode
		}
		return c.JSON(resp)
	}
}

func parseCallerID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("user_id").(string)
	return uuid.Parse(raw)
}
