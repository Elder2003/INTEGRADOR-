package ai

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// routeHandler handles POST /ai/complete.
// Validation failures return 400 BEFORE calling Dispatch so no log entry is created (FR-011).
func routeHandler(svc *Service, primaryTimeoutSecs int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			RequestID *string      `json:"request_id"`
			Model     string       `json:"model"`
			Messages  []ChatMessage `json:"messages"`
			MaxTokens int          `json:"max_tokens"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "messages required",
			})
		}
		// Validate messages before dispatching — no log entry created for invalid requests.
		if len(body.Messages) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "messages required",
			})
		}

		var reqID uuid.UUID
		if body.RequestID != nil && *body.RequestID != "" {
			parsed, err := uuid.Parse(*body.RequestID)
			if err == nil {
				reqID = parsed
			} else {
				reqID = uuid.New()
			}
		} else {
			reqID = uuid.New()
		}

		req := DispatchRequest{
			RequestID: reqID,
			Model:     body.Model,
			Messages:  body.Messages,
			MaxTokens: body.MaxTokens,
		}

		resp, entry, err := svc.Dispatch(c.Context(), req, primaryTimeoutSecs)
		if err != nil {
			mode := svc.modeCache.Load().mode
			switch mode {
			case "auto":
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":    "backend_unavailable",
					"message":  "all backends failed",
					"backends": []string{"local", "cloud"},
				})
			case "cloud":
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":   "backend_unavailable",
					"message": "cloud backend failed",
				})
			default:
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":   "backend_unavailable",
					"message": "primary backend failed",
				})
			}
		}

		c.Set("X-AI-Backend", entry.BackendUsed)
		c.Set("X-AI-Mode", entry.Mode)
		c.Set("X-Request-ID", entry.RequestID.String())
		return c.JSON(resp)
	}
}
