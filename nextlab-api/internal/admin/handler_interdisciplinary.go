package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SetScopesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid actividad id"})
		}

		var body struct {
			Scopes []InterdisciplinaryScopeInput `json:"scopes"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid request body"})
		}
		if body.Scopes == nil {
			body.Scopes = []InterdisciplinaryScopeInput{}
		}

		scopes, err := repo.SetScopes(c.Context(), aid, body.Scopes)
		if err != nil {
			slog.ErrorContext(c.Context(), "set scopes error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(fiber.Map{
			"actividad_id": aid,
			"scopes":       scopes,
		})
	}
}

func GetScopesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid actividad id"})
		}

		role, _ := c.Locals("user_role").(string)
		if role == "docente" {
			callerID, err := uuid.Parse(c.Locals("user_id").(string))
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized", Message: "invalid token"})
			}
			owns, err := repo.VerifyDocenteOwnsActividad(c.Context(), aid, callerID)
			if err != nil {
				slog.ErrorContext(c.Context(), "verify docente owns actividad error", "err", err)
				return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
			}
			if !owns {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{Error: "forbidden", Message: "access denied"})
			}
		}

		scopes, err := repo.GetScopes(c.Context(), aid)
		if err != nil {
			slog.ErrorContext(c.Context(), "get scopes error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(fiber.Map{
			"actividad_id": aid,
			"scopes":       scopes,
		})
	}
}

func DeleteScopesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid actividad id"})
		}

		if err := repo.DeleteScopes(c.Context(), aid); err != nil {
			slog.ErrorContext(c.Context(), "delete scopes error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
