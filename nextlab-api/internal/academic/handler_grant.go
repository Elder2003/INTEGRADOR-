package academic

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateGrantHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}

		owned, err := ActividadOwnedByDocente(c.Context(), db, actividadID, docenteID)
		if err != nil {
			return internalError(c)
		}
		if !owned {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found or not owned by you", CorrelationID: correlationID(c),
			})
		}

		var req CreateGrantRequest
		if err := c.BodyParser(&req); err != nil || req.CodocenteID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "codocente_id is required", CorrelationID: correlationID(c),
			})
		}

		isCodocente, err := UserHasRole(c.Context(), db, req.CodocenteID, "co_docente")
		if err != nil {
			return internalError(c)
		}
		if !isCodocente {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "validation_error", Message: "target user is not a co_docente", CorrelationID: correlationID(c),
			})
		}

		grant, err := CreateGrant(c.Context(), db, actividadID, req.CodocenteID, docenteID)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error: "conflict", Message: "grant already active for this co_docente on this actividad", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "grant issued",
			"event", "grant_issued", "actividad_id", actividadID, "codocente_id", req.CodocenteID, "granted_by", docenteID)
		return c.Status(fiber.StatusCreated).JSON(grant)
	}
}

func RevokeGrantHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}
		codocenteUID, err := uuid.Parse(c.Params("uid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid user id", CorrelationID: correlationID(c),
			})
		}

		owned, err := ActividadOwnedByDocente(c.Context(), db, actividadID, docenteID)
		if err != nil {
			return internalError(c)
		}
		if !owned {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "actividad not found or not owned by you", CorrelationID: correlationID(c),
			})
		}

		if err := RevokeGrant(c.Context(), db, actividadID, codocenteUID, docenteID); err != nil {
			return internalError(c)
		}

		slog.InfoContext(c.Context(), "grant revoked",
			"event", "grant_revoked", "actividad_id", actividadID, "codocente_id", codocenteUID, "revoked_by", docenteID)
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func CodocenteGetActividadHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		codocenteID, err := userID(c)
		if err != nil {
			return unauthorized(c)
		}

		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: correlationID(c),
			})
		}

		if err := ValidateCodocenteAccess(c.Context(), db, actividadID, codocenteID); err != nil {
			if err == ErrGrantNotFound {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Error: "forbidden", Message: "no active grant for this actividad", CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		actividad, err := GetActividadByID(c.Context(), db, actividadID)
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
