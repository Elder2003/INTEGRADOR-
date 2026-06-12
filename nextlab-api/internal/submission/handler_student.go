package submission

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func studentID(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Locals("user_id").(string))
}

func submUnauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Error:         "unauthorized",
		Message:       "Missing or invalid authorization token",
		CorrelationID: submCorrelationID(c),
	})
}

func submCorrelationID(c *fiber.Ctx) string {
	if v, ok := c.Locals("correlation_id").(string); ok {
		return v
	}
	return ""
}

func submInternalError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error:         "internal_error",
		Message:       "An unexpected error occurred",
		CorrelationID: submCorrelationID(c),
	})
}

func parseAID(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Params("aid"))
}

// ─── POST /actividades/:aid/submission ───────────────────────────────────────

func UploadHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}

		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		header, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "missing_file", Message: "multipart field 'file' is required", CorrelationID: submCorrelationID(c),
			})
		}

		resp, err := svc.UploadFile(c.Context(), sid, aid, header)
		if err != nil {
			return mapStudentError(c, err)
		}

		slog.InfoContext(c.Context(), "submission uploaded",
			"event", "submission_uploaded", "student_id", sid, "actividad_id", aid, "is_late", resp.IsLate)
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

// ─── GET /actividades/:aid/submission ────────────────────────────────────────

func GetSubmissionHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}

		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		sub, err := svc.repo.GetSubmissionByStudent(c.Context(), sid, aid)
		if err != nil {
			return submInternalError(c)
		}
		if sub == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "no submission found for this activity", CorrelationID: submCorrelationID(c),
			})
		}

		return c.JSON(toResponse(sub))
	}
}

// ─── PUT /actividades/:aid/submission ────────────────────────────────────────

func ReplaceHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}

		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		header, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "missing_file", Message: "multipart field 'file' is required", CorrelationID: submCorrelationID(c),
			})
		}

		resp, err := svc.ReplaceFile(c.Context(), sid, aid, header)
		if err != nil {
			return mapStudentError(c, err)
		}

		slog.InfoContext(c.Context(), "submission replaced",
			"event", "submission_replaced", "student_id", sid, "actividad_id", aid)
		return c.JSON(resp)
	}
}

// ─── DELETE /actividades/:aid/submission ─────────────────────────────────────

func DeleteHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}

		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		if err := svc.DeleteOwnSubmission(c.Context(), sid, aid); err != nil {
			return mapStudentError(c, err)
		}

		slog.InfoContext(c.Context(), "submission deleted",
			"event", "submission_deleted", "student_id", sid, "actividad_id", aid)
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ─── POST /actividades/:aid/submission/exam ──────────────────────────────────

func ExamAnswerHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}
		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		var body struct {
			Answer string `json:"answer"`
		}
		if err := c.BodyParser(&body); err != nil || body.Answer == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "missing_answer", Message: "field 'answer' is required", CorrelationID: submCorrelationID(c),
			})
		}

		resp, err := svc.SubmitExamAnswer(c.Context(), sid, aid, body.Answer)
		if err != nil {
			return mapStudentError(c, err)
		}
		return c.Status(fiber.StatusCreated).JSON(resp)
	}
}

// ─── GET /actividades/:aid/submission/download ────────────────────────────────

func StudentDownloadHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := studentID(c)
		if err != nil {
			return submUnauthorized(c)
		}

		aid, err := parseAID(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error: "invalid_id", Message: "invalid actividad id", CorrelationID: submCorrelationID(c),
			})
		}

		sub, err := svc.repo.GetSubmissionByStudent(c.Context(), sid, aid)
		if err != nil {
			return submInternalError(c)
		}
		if sub == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
				Error: "not_found", Message: "no submission found for this activity", CorrelationID: submCorrelationID(c),
			})
		}

		result, err := svc.StreamFile(c.Context(), sub)
		if err != nil {
			return mapStudentError(c, err)
		}
		defer result.Object.Close()

		c.Set("Content-Type", result.ContentType)
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, result.OriginalFilename))
		c.Set("Content-Length", fmt.Sprintf("%d", result.Size))
		return c.SendStream(result.Object, int(result.Size))
	}
}

// ─── Error mapper ─────────────────────────────────────────────────────────────

func mapStudentError(c *fiber.Ctx, err error) error {
	cid := submCorrelationID(c)
	switch {
	case errors.Is(err, ErrNotEnrolled), errors.Is(err, ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: "forbidden", Message: "access denied", CorrelationID: cid,
		})
	case errors.Is(err, ErrSubmissionExists):
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error: "submission_exists", Message: "a submission already exists for this activity", CorrelationID: cid,
		})
	case errors.Is(err, ErrLateSubmissionsDisabled):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
			Error: "late_submissions_disabled", Message: "the deadline has passed and late submissions are not allowed", CorrelationID: cid,
		})
	case errors.Is(err, ErrInvalidActivityType):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
			Error: "invalid_activity_type", Message: "this activity type does not accept file submissions", CorrelationID: cid,
		})
	case errors.Is(err, ErrFileTooLarge):
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error: "file_too_large", Message: "file exceeds the 50 MB limit", CorrelationID: cid,
		})
	case errors.Is(err, ErrSubmissionNotFound), errors.Is(err, ErrActivityNotFound):
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error: "not_found", Message: "resource not found", CorrelationID: cid,
		})
	case errors.Is(err, ErrSubmissionLocked):
		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error: "submission_locked", Message: "the edit window has expired", CorrelationID: cid,
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
