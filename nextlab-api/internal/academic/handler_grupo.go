package academic

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateGrupoHandler(db *pgxpool.Pool) fiber.Handler {
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

		var req CreateGrupoRequest
		if err := c.BodyParser(&req); err != nil || req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "name is required", CorrelationID: correlationID(c),
			})
		}

		grupo, err := CreateGrupo(c.Context(), db, cursoID, req.Name, req.Capacity)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error: "conflict", Message: "a grupo with that name already exists in this curso", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "grupo created",
			"event", "grupo_created", "grupo_id", grupo.ID, "curso_id", cursoID)
		return c.Status(fiber.StatusCreated).JSON(grupo)
	}
}

func ListGruposHandler(db *pgxpool.Pool) fiber.Handler {
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

		list, err := GetGruposByCurso(c.Context(), db, cursoID)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []Grupo{}
		}
		return c.JSON(list)
	}
}

func UpdateGrupoHandler(db *pgxpool.Pool) fiber.Handler {
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
		grupoID, err := uuid.Parse(c.Params("gid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid grupo id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		var req UpdateGrupoRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_request", Message: "invalid body", CorrelationID: correlationID(c),
			})
		}

		grupo, err := UpdateGrupo(c.Context(), db, grupoID, cursoID, req.Name, req.Capacity)
		if err != nil {
			return internalError(c)
		}
		if grupo == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "grupo not found", CorrelationID: correlationID(c),
			})
		}
		return c.JSON(grupo)
	}
}

func DeleteGrupoHandler(db *pgxpool.Pool) fiber.Handler {
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
		grupoID, err := uuid.Parse(c.Params("gid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid grupo id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		if err := DeleteGrupo(c.Context(), db, grupoID, cursoID); err != nil {
			return internalError(c)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func AssignStudentHandler(db *pgxpool.Pool) fiber.Handler {
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
		grupoID, err := uuid.Parse(c.Params("gid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid grupo id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		var req AssignStudentRequest
		if err := c.BodyParser(&req); err != nil || req.UserID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "user_id is required", CorrelationID: correlationID(c),
			})
		}

		enrolled, err := StudentEnrolledInCurso(c.Context(), db, req.UserID, cursoID)
		if err != nil {
			return internalError(c)
		}
		if !enrolled {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "student is not enrolled in this curso", CorrelationID: correlationID(c),
			})
		}

		asignacion, err := AssignStudentToGrupo(c.Context(), db, req.UserID, grupoID, cursoID, docenteID)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error: "conflict", Message: "student is already assigned to a grupo in this curso", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		return c.Status(fiber.StatusCreated).JSON(asignacion)
	}
}

func RemoveStudentHandler(db *pgxpool.Pool) fiber.Handler {
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
		grupoID, err := uuid.Parse(c.Params("gid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid grupo id", CorrelationID: correlationID(c),
			})
		}
		studentUID, err := uuid.Parse(c.Params("uid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid user id", CorrelationID: correlationID(c),
			})
		}

		if err := requireCursoOwnership(c, db, cursoID, docenteID); err != nil {
			return err
		}

		if err := RemoveStudentFromGrupo(c.Context(), db, studentUID, grupoID); err != nil {
			return internalError(c)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// requireCursoOwnership returns a fiber error response if the docente doesn't own the curso.
func requireCursoOwnership(c *fiber.Ctx, db *pgxpool.Pool, cursoID, docenteID uuid.UUID) error {
	curso, err := GetCursoByIDAndDocente(c.Context(), db, cursoID, docenteID)
	if err != nil {
		return internalError(c)
	}
	if curso == nil {
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "not_found", Message: "curso not found", CorrelationID: correlationID(c),
		})
	}
	return nil
}
