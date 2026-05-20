package submission

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ─── GET /cursos/:cid/actividades/:aid/submissions ────────────────────────────

func ListSubmissionsHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error: "unauthorized", Message: "Missing or invalid authorization token", CorrelationID: submCorrelationID(c),
			})
		}

		role, _ := c.Locals("user_role").(string)

		cursoID, err := uuid.Parse(c.Params("cid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: submCorrelationID(c),
			})
		}

		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		items, err := svc.ListSubmissions(c.Context(), actividadID, cursoID, callerID, role)
		if err != nil {
			return mapDocenteError(c, err)
		}
		if items == nil {
			items = []SubmissionListItem{}
		}

		return c.JSON(SubmissionListResponse{
			ActividadID: actividadID,
			Submissions: items,
			Total:       len(items),
		})
	}
}

// ─── GET /submissions/:sid/download ──────────────────────────────────────────

func DownloadHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error: "unauthorized", Message: "Missing or invalid authorization token", CorrelationID: submCorrelationID(c),
			})
		}

		role, _ := c.Locals("user_role").(string)

		submissionID, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid submission id", CorrelationID: submCorrelationID(c),
			})
		}

		sub, err := svc.ResolveSubmissionForDownload(c.Context(), submissionID, callerID, role)
		if err != nil {
			return mapDocenteError(c, err)
		}
		if sub == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "submission not found", CorrelationID: submCorrelationID(c),
			})
		}

		result, err := svc.StreamFile(c.Context(), sub)
		if err != nil {
			return mapDocenteError(c, err)
		}
		defer result.Object.Close()

		slog.InfoContext(c.Context(), "submission downloaded",
			"event", "submission_downloaded", "submission_id", submissionID, "caller_id", callerID, "role", role)

		c.Set("Content-Type", result.ContentType)
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, result.OriginalFilename))
		c.Set("Content-Length", fmt.Sprintf("%d", result.Size))
		return c.SendStream(result.Object, int(result.Size))
	}
}

// ─── GET /cursos/:cid/actividades/:aid/config-changes ─────────────────────────

func ListConfigChangesHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		callerID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error: "unauthorized", Message: "Missing or invalid authorization token", CorrelationID: submCorrelationID(c),
			})
		}

		role, _ := c.Locals("user_role").(string)

		cursoID, err := uuid.Parse(c.Params("cid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid curso id", CorrelationID: submCorrelationID(c),
			})
		}

		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		changes, err := svc.repo.GetConfigChanges(c.Context(), actividadID, cursoID, callerID, role)
		if err != nil {
			if err.Error() == "forbidden" {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Error: "forbidden", Message: "access denied", CorrelationID: submCorrelationID(c),
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
				Error: "internal_error", Message: "An unexpected error occurred", CorrelationID: submCorrelationID(c),
			})
		}
		if changes == nil {
			changes = []ConfigChangeResponse{}
		}

		return c.JSON(ConfigChangesResponse{
			ActividadID: actividadID,
			Changes:     changes,
		})
	}
}

// ─── Error mapper ─────────────────────────────────────────────────────────────

func mapDocenteError(c *fiber.Ctx, err error) error {
	cid := submCorrelationID(c)
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotEnrolled):
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: "forbidden", Message: "access denied", CorrelationID: cid,
		})
	case errors.Is(err, ErrSubmissionNotFound), errors.Is(err, ErrActivityNotFound):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "not_found", Message: "resource not found", CorrelationID: cid,
		})
	case errors.Is(err, ErrStorageUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(ErrorResponse{
			Error: "storage_unavailable", Message: "file storage is temporarily unavailable", CorrelationID: cid,
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error: "internal_error", Message: "An unexpected error occurred", CorrelationID: cid,
		})
	}
}
