package academic

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateCursoHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		var req CreateCursoRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_request", Message: "invalid body", CorrelationID: correlationID(c),
			})
		}
		if req.MateriaID == uuid.Nil || req.SemesterLabel == "" || req.Section == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "materia_id, semester_label, and section are required", CorrelationID: correlationID(c),
			})
		}

		active, err := MateriaIsActive(c.Context(), db, req.MateriaID)
		if err != nil {
			return internalError(c)
		}
		if !active {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "materia not found or inactive", CorrelationID: correlationID(c),
			})
		}

		curso, err := CreateCurso(c.Context(), db, req.MateriaID, docenteID, req.SemesterLabel, req.Section)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error: "conflict", Message: "a curso with that materia, semester_label, and section already exists", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "curso created",
			"event", "curso_created", "curso_id", curso.ID, "docente_id", docenteID)
		return c.Status(fiber.StatusCreated).JSON(curso)
	}
}

func ListCursosHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		list, err := GetCursosByDocente(c.Context(), db, docenteID)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []Curso{}
		}
		return c.JSON(list)
	}
}

func GetCursoHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		cursoID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: correlationID(c),
			})
		}

		curso, err := GetCursoByIDAndDocente(c.Context(), db, cursoID, docenteID)
		if err != nil {
			return internalError(c)
		}
		if curso == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "curso not found", CorrelationID: correlationID(c),
			})
		}
		return c.JSON(curso)
	}
}

func UpdateCursoHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		cursoID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: correlationID(c),
			})
		}

		var req UpdateCursoRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_request", Message: "invalid body", CorrelationID: correlationID(c),
			})
		}

		curso, err := UpdateCurso(c.Context(), db, cursoID, docenteID, req.SemesterLabel, req.Section)
		if err != nil {
			return internalError(c)
		}
		if curso == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "curso not found", CorrelationID: correlationID(c),
			})
		}
		return c.JSON(curso)
	}
}

func DeleteCursoHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		cursoID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: correlationID(c),
			})
		}

		hasSubmissions, err := CursoHasSubmissions(c.Context(), db, cursoID)
		if err != nil {
			return internalError(c)
		}
		if hasSubmissions {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error: "has_submissions", Message: "cannot delete curso with existing submissions", CorrelationID: correlationID(c),
			})
		}

		if err := DeactivateCurso(c.Context(), db, cursoID, docenteID); err != nil {
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "curso deactivated",
			"event", "curso_deactivated", "curso_id", cursoID, "docente_id", docenteID)
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func GetCursoRosterHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		cursoID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: correlationID(c),
			})
		}

		// Verify ownership
		curso, err := GetCursoByIDAndDocente(c.Context(), db, cursoID, docenteID)
		if err != nil {
			return internalError(c)
		}
		if curso == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "curso not found", CorrelationID: correlationID(c),
			})
		}

		roster, err := GetCursoRoster(c.Context(), db, cursoID)
		if err != nil {
			return internalError(c)
		}
		if roster == nil {
			roster = []RosterEntry{}
		}
		return c.JSON(roster)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func userID(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Locals("user_id").(string))
}

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Error:         "unauthorized",
		Message:       "Missing or invalid authorization token",
		CorrelationID: correlationID(c),
	})
}
