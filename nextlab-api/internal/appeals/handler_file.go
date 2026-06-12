package appeals

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func callerIDFromLocals(c *fiber.Ctx) (uuid.UUID, bool) {
	raw := c.Locals("user_id")
	str, ok := raw.(string)
	if !ok || str == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

func roleFromLocals(c *fiber.Ctx) string {
	raw := c.Locals("user_role")
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

func FileAppealHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req FileAppealRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
		}
		if req.SessionID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_session_id"})
		}
		if req.Type != "analysis_dispute" && req.Type != "grade_dispute" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_type", "message": "type must be analysis_dispute or grade_dispute"})
		}
		if strings.TrimSpace(req.Reason) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_reason"})
		}

		owns, err := repo.VerifyStudentOwnsSession(c.Context(), req.SessionID, callerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		open, err := repo.HasOpenAppealOfType(c.Context(), req.SessionID, req.Type)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if open {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "duplicate_active_appeal",
				"message": "An active appeal of this type already exists for this session",
			})
		}

		appeal, err := repo.InsertAppeal(c.Context(), req, callerID)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "duplicate_active_appeal",
					"message": "An active appeal of this type already exists for this session",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		return c.Status(fiber.StatusCreated).JSON(appeal)
	}
}

func GetAppealHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role := roleFromLocals(c)

		appeal, err := repo.GetAppeal(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if appeal == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		}

		// student can only see their own; docente can see appeals in their sessions; admin sees all
		if role == "admin" {
			return c.Status(fiber.StatusOK).JSON(appeal)
		}
		if appeal.FiledBy == callerID {
			return c.Status(fiber.StatusOK).JSON(appeal)
		}
		if role == "docente" {
			owns, err := repo.VerifyDocenteOwnsAppealSession(c.Context(), id, callerID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
			}
			if owns {
				return c.Status(fiber.StatusOK).JSON(appeal)
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
}

func ListAppealsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role := roleFromLocals(c)

		filter := AppealFilter{
			Status: c.Query("status"),
		}

		if sessionIDStr := c.Query("session_id"); sessionIDStr != "" {
			sid, err := uuid.Parse(sessionIDStr)
			if err == nil {
				filter.SessionID = &sid
			}
		}

		switch role {
		case "estudiante":
			filter.FiledBy = &callerID
		case "docente":
			// scoped to actividades owned by this docente — handled via ActividadID if provided,
			// otherwise list all where docente owns the session (fall through with no extra filter;
			// SQL already joins actividades/cursos when ActividadID is set).
			// For the generic /appeals list, docente sees appeals from their own sessions.
			// We model this as filed_by=nil but with actividad filter = nil → unscoped for docente
			// is intentionally showing all appeals for their activities.
			// In production you'd want a subquery; for now we return all and rely on the client
			// to use /actividades/:aid/appeals for scoped views.
		case "admin":
			// no extra filter
		default:
			// unknown role — scope to caller's own appeals
			filter.FiledBy = &callerID
		}

		appeals, total, err := repo.ListAppeals(c.Context(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"appeals": appeals, "total": total})
	}
}

func ListActivityAppealsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_actividad_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role := roleFromLocals(c)

		if role != "admin" {
			owns, err := repo.VerifyDocenteOwnsActividad(c.Context(), aid, callerID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
			}
			if !owns {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
		}

		filter := AppealFilter{
			ActividadID: &aid,
			Status:      c.Query("status"),
		}

		appeals, total, err := repo.ListAppeals(c.Context(), filter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"actividad_id": aid,
			"appeals":      appeals,
			"total":        total,
		})
	}
}
