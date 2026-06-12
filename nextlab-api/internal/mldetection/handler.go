package mldetection

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

func RetrainHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := svc.Retrain(c.Context(), 50); err != nil {
			var insuf *InsufficientDataError
			if errors.As(err, &insuf) {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
					"error":   "insufficient_training_data",
					"message": err.Error(),
					"required": insuf.Required,
					"available": insuf.Available,
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "training_failed", "message": err.Error()})
		}
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message":            "Training job started",
			"estimated_minutes":  5,
		})
	}
}
