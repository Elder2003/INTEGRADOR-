package admin

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateFormatTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateFormatTemplateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid request body"})
		}
		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "name is required"})
		}

		callerID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized", Message: "invalid token"})
		}

		rulesJSON, err := json.Marshal(req.Rules)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid rules"})
		}

		ft, err := repo.InsertFormatTemplate(c.Context(), req.Name, req.Description, rulesJSON, callerID)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "conflict", Message: "template name already exists"})
			}
			slog.ErrorContext(c.Context(), "insert format template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.Status(fiber.StatusCreated).JSON(ft)
	}
}

func ListFormatTemplatesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		activeOnly, _ := strconv.ParseBool(c.Query("active_only", "false"))

		templates, err := repo.ListFormatTemplates(c.Context(), activeOnly)
		if err != nil {
			slog.ErrorContext(c.Context(), "list format templates error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(fiber.Map{
			"templates": templates,
			"total":     len(templates),
		})
	}
}

func GetFormatTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid template id"})
		}
		ft, err := repo.GetFormatTemplate(c.Context(), id)
		if err != nil {
			slog.ErrorContext(c.Context(), "get format template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if ft == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "template not found"})
		}
		return c.JSON(ft)
	}
}

func UpdateFormatTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid template id"})
		}
		var req UpdateFormatTemplateRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid request body"})
		}
		ft, err := repo.UpdateFormatTemplate(c.Context(), id, req)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "conflict", Message: "template name already exists"})
			}
			slog.ErrorContext(c.Context(), "update format template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if ft == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "template not found"})
		}
		return c.JSON(ft)
	}
}

func DeleteFormatTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid template id"})
		}

		inUse, err := repo.IsTemplateInUse(c.Context(), id)
		if err != nil {
			slog.ErrorContext(c.Context(), "is template in use error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if inUse {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "conflict", Message: "template is in use by one or more actividades"})
		}

		if err := repo.DeleteFormatTemplate(c.Context(), id); err != nil {
			slog.ErrorContext(c.Context(), "delete format template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func ApplyTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid actividad id"})
		}

		callerID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{Error: "unauthorized", Message: "invalid token"})
		}

		var body struct {
			TemplateID uuid.UUID `json:"template_id"`
		}
		if err := c.BodyParser(&body); err != nil || body.TemplateID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "template_id is required"})
		}

		owns, err := repo.VerifyDocenteOwnsActividad(c.Context(), aid, callerID)
		if err != nil {
			slog.ErrorContext(c.Context(), "verify docente owns actividad error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if !owns {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{Error: "forbidden", Message: "access denied"})
		}

		ft, err := repo.GetFormatTemplate(c.Context(), body.TemplateID)
		if err != nil {
			slog.ErrorContext(c.Context(), "get format template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if ft == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "template not found"})
		}

		rulesJSON, err := json.Marshal(ft.Rules)
		if err != nil {
			slog.ErrorContext(c.Context(), "marshal rules error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		if err := repo.ApplyTemplate(c.Context(), aid, ft.ID, callerID); err != nil {
			slog.ErrorContext(c.Context(), "apply template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if err := repo.UpdateActividadFormatRules(c.Context(), aid, rulesJSON); err != nil {
			slog.ErrorContext(c.Context(), "update actividad format rules error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(AppliedTemplate{
			FormatTemplateID: ft.ID,
			Name:             ft.Name,
			Rules:            ft.Rules,
		})
	}
}

func GetAppliedTemplateHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid actividad id"})
		}

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

		at, err := repo.GetAppliedTemplate(c.Context(), aid)
		if err != nil {
			slog.ErrorContext(c.Context(), "get applied template error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		return c.JSON(at)
	}
}
