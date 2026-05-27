package analytics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func errResp(c *fiber.Ctx, status int, code, message string) error {
	corrID, _ := c.Locals("correlation_id").(string)
	return c.Status(status).JSON(fiber.Map{
		"error":          code,
		"message":        message,
		"correlation_id": corrID,
	})
}

func claimsDocenteID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Locals("user_id")
	if raw == nil {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	switch v := raw.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, fiber.ErrUnauthorized
	}
}

// ActivityAnalyticsHandler returns live analytics for a single activity.
func ActivityAnalyticsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := claimsDocenteID(c)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
		}
		actividadID, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "actividadId must be a UUID")
		}

		// Verify ownership and fetch tipo (actividades.docente_id doesn't exist; join to cursos)
		var ownerID uuid.UUID
		var tipo string
		dbErr := repo.db.QueryRow(c.Context(),
			`SELECT c.docente_id, a.tipo FROM actividades a JOIN cursos c ON c.id = a.curso_id WHERE a.id = $1`,
			actividadID,
		).Scan(&ownerID, &tipo)
		if dbErr != nil {
			return errResp(c, fiber.StatusNotFound, "not_found", "actividad not found")
		}
		if ownerID != docenteID {
			return errResp(c, fiber.StatusForbidden, "forbidden", "access denied")
		}

		aa, err := repo.ActivityAnalyticsQuery(c.Context(), actividadID, tipo)
		if err != nil {
			return errResp(c, fiber.StatusInternalServerError, "query_error", err.Error())
		}
		return c.JSON(aa)
	}
}

// CourseAnalyticsHandler returns live analytics aggregated across all activities in a course.
func CourseAnalyticsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := claimsDocenteID(c)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
		}
		cursoID, err := uuid.Parse(c.Params("cursoId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "cursoId must be a UUID")
		}

		// Verify ownership
		var ownerID uuid.UUID
		dbErr := repo.db.QueryRow(c.Context(),
			`SELECT docente_id FROM cursos WHERE id = $1`, cursoID,
		).Scan(&ownerID)
		if dbErr != nil {
			return errResp(c, fiber.StatusNotFound, "not_found", "curso not found")
		}
		if ownerID != docenteID {
			return errResp(c, fiber.StatusForbidden, "forbidden", "access denied")
		}

		ca, err := repo.CourseAnalyticsQuery(c.Context(), cursoID, docenteID)
		if err != nil {
			return errResp(c, fiber.StatusInternalServerError, "query_error", err.Error())
		}
		return c.JSON(ca)
	}
}
