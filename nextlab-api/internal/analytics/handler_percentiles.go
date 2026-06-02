package analytics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// PercentilesHandler handles GET /docente/cursos/:cid/actividades/:aid/analytics/percentiles?student_id=<uuid>
func PercentilesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_actividad_id"})
		}

		studentIDStr := c.Query("student_id")
		if studentIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "student_id_required"})
		}
		studentID, err := uuid.Parse(studentIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_student_id"})
		}

		result, totalGraded, err := repo.GetPercentiles(c.Context(), actividadID, studentID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if result == nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error":   "insufficient_data",
				"message": fiber.Map{"required": 5, "current": totalGraded},
			})
		}

		return c.JSON(result)
	}
}

// AdminOverviewHandler handles GET /admin/analytics/overview
func AdminOverviewHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		period := c.Query("period", "")
		result, err := repo.GetAdminOverview(c.Context(), period)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		return c.JSON(result)
	}
}

// AdminIntegrityHandler handles GET /admin/analytics/integrity
func AdminIntegrityHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		period := c.Query("period", "")
		result, err := repo.GetIntegrityByCarrera(c.Context(), period)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		return c.JSON(fiber.Map{"period": period, "by_carrera": result})
	}
}

// AdminSubmissionsHandler handles GET /admin/analytics/submissions
func AdminSubmissionsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		period := c.Query("period", "")
		result, err := repo.GetSubmissionsByMateria(c.Context(), period)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		return c.JSON(fiber.Map{"period": period, "by_materia": result})
	}
}
