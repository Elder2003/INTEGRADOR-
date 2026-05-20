package academic

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Docente — Actividades ────────────────────────────────────────────────────

func CreateActividadHandler(db *pgxpool.Pool) fiber.Handler {
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

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		var req CreateActividadRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_request", Message: "invalid body", CorrelationID: correlationID(c),
			})
		}
		if req.Title == "" || req.Tipo == "" || req.Deadline.IsZero() {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "title, tipo, and deadline are required", CorrelationID: correlationID(c),
			})
		}

		actividad, err := CreateActividad(c.Context(), db, cursoID, req.Title, req.Tipo, req.Deadline, req.Instructions)
		if err != nil {
			if isInvalidEnumValue(err) {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Error: "validation_error", Message: "invalid tipo value; allowed: practica, tarea, informe, examen, defensa_oral", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "actividad created",
			"event", "actividad_created", "actividad_id", actividad.ID, "curso_id", cursoID, "tipo", actividad.Tipo)
		return c.Status(fiber.StatusCreated).JSON(actividad)
	}
}

func ListActividadesHandler(db *pgxpool.Pool) fiber.Handler {
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

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		list, err := GetActividadesByCurso(c.Context(), db, cursoID)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []Actividad{}
		}
		return c.JSON(list)
	}
}

func GetActividadHandler(db *pgxpool.Pool) fiber.Handler {
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
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		actividad, err := GetActividadByIDAndCurso(c.Context(), db, actividadID, cursoID)
		if err != nil {
			return internalError(c)
		}
		if actividad == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found", CorrelationID: correlationID(c),
			})
		}
		return c.JSON(actividad)
	}
}

func UpdateActividadHandler(db *pgxpool.Pool) fiber.Handler {
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
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		existing, err := GetActividadByIDAndCurso(c.Context(), db, actividadID, cursoID)
		if err != nil {
			return internalError(c)
		}
		if existing == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found", CorrelationID: correlationID(c),
			})
		}

		var req UpdateActividadRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_request", Message: "invalid body", CorrelationID: correlationID(c),
			})
		}

		if err := GuardActivityMutation(existing, &req, false); err != nil {
			if err == ErrActivityFieldLocked {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
					Error: "field_locked", Message: "title and tipo cannot be changed after submissions exist", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		var updated *Actividad
		if existing.HasSubmissions {
			// Only deadline and instructions can be changed
			updated, err = UpdateActividad(c.Context(), db, actividadID, cursoID, req.Deadline, req.Instructions)
		} else {
			// All fields can change; use full update if any is provided
			title := existing.Title
			tipo := existing.Tipo
			deadline := existing.Deadline
			instructions := existing.Instructions
			if req.Title != nil {
				title = *req.Title
			}
			if req.Tipo != nil {
				tipo = *req.Tipo
			}
			if req.Deadline != nil {
				deadline = *req.Deadline
			}
			if req.Instructions != nil {
				instructions = *req.Instructions
			}
			updated, err = UpdateActividadFull(c.Context(), db, actividadID, cursoID, title, tipo, deadline, instructions)
		}
		if err != nil {
			if isInvalidEnumValue(err) {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Error: "validation_error", Message: "invalid tipo value", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}
		if updated == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found", CorrelationID: correlationID(c),
			})
		}

		slog.InfoContext(c.Context(), "actividad updated",
			"event", "actividad_updated", "actividad_id", actividadID, "docente_id", docenteID)
		return c.JSON(updated)
	}
}

func DeleteActividadHandler(db *pgxpool.Pool) fiber.Handler {
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
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		existing, err := GetActividadByIDAndCurso(c.Context(), db, actividadID, cursoID)
		if err != nil {
			return internalError(c)
		}
		if existing == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found", CorrelationID: correlationID(c),
			})
		}

		if err := GuardActivityMutation(existing, nil, true); err != nil {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error: "has_submissions", Message: "cannot delete actividad with existing submissions", CorrelationID: correlationID(c),
			})
		}

		if err := DeleteActividad(c.Context(), db, actividadID, cursoID); err != nil {
			return internalError(c)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ─── Student — Activity View ─────────────────────────────────────────────────

func StudentListCursosHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		studentID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		list, err := GetCursosByStudent(c.Context(), db, studentID)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []StudentCursoEntry{}
		}
		return c.JSON(list)
	}
}

func StudentListActividadesHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		studentID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		cursoID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: correlationID(c),
			})
		}

		enrolled, err := StudentEnrolledActiveCurso(c.Context(), db, studentID, cursoID)
		if err != nil {
			return internalError(c)
		}
		if !enrolled {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Error: "forbidden", Message: "not enrolled in this curso", CorrelationID: correlationID(c),
			})
		}

		list, err := GetActividadesByCursoForStudent(c.Context(), db, cursoID)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []StudentActividadEntry{}
		}
		return c.JSON(list)
	}
}
