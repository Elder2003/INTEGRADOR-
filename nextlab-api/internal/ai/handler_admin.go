package ai

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// getConfigHandler handles GET /ai/config — returns current routing mode (any authenticated user).
func getConfigHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cfg, err := svc.GetCurrentConfig(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if cfg == nil {
			return c.JSON(fiber.Map{
				"mode":       "local",
				"prev_mode":  nil,
				"changed_by": nil,
				"changed_at": nil,
			})
		}
		return c.JSON(toConfigResponse(cfg))
	}
}

// putConfigHandler handles PUT /ai/config — changes routing mode (admin only).
func putConfigHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body struct {
			Mode string `json:"mode"`
		}
		if err := c.BodyParser(&body); err != nil || body.Mode == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_mode",
				"message": "must be local, auto, or cloud",
			})
		}
		if body.Mode != "local" && body.Mode != "auto" && body.Mode != "cloud" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "invalid_mode",
				"message": "must be local, auto, or cloud",
			})
		}

		callerID, _ := parseUserID(c)
		cfg, err := svc.SetMode(c.Context(), body.Mode, callerID)
		if errors.Is(err, ErrModeUnchanged) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "no_change",
				"message": "mode is already " + body.Mode,
			})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		return c.JSON(toConfigResponse(cfg))
	}
}

// getConfigHistoryHandler handles GET /ai/config/history — returns full mode change history (admin only).
func getConfigHistoryHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		limit := c.QueryInt("limit", 20)
		offset := c.QueryInt("offset", 0)
		if limit < 1 || limit > 100 {
			limit = 20
		}

		entries, total, err := svc.repo.GetConfigHistory(c.Context(), limit, offset)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		out := make([]fiber.Map, 0, len(entries))
		for _, e := range entries {
			out = append(out, toConfigResponse(e))
		}
		return c.JSON(fiber.Map{
			"entries": out,
			"total":   total,
		})
	}
}

func toConfigResponse(cfg *RouterConfig) fiber.Map {
	return fiber.Map{
		"id":         cfg.ID,
		"mode":       cfg.Mode,
		"prev_mode":  cfg.PrevMode,
		"changed_by": cfg.ChangedBy,
		"changed_at": cfg.ChangedAt,
	}
}

func parseUserID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("user_id").(string)
	return uuid.Parse(raw)
}
