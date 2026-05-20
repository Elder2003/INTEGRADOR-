package tracing

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func openSessionHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_activity_id"})
		}
		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		session, isNew, err := svc.OpenSession(c.Context(), studentID, actividadID)
		if err != nil {
			switch {
			case errors.Is(err, ErrNotEnrolled):
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not_enrolled"})
			case errors.Is(err, ErrTraceNotApplicable):
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "trace_not_applicable"})
			case errors.Is(err, ErrSessionClosed):
				return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "session_closed"})
			default:
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
			}
		}

		resp := toSessionResponse(session)
		if isNew {
			return c.Status(fiber.StatusCreated).JSON(resp)
		}
		return c.Status(fiber.StatusOK).JSON(resp)
	}
}

func getSessionHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_activity_id"})
		}
		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		session, err := svc.repo.GetSessionByStudentActivity(c.Context(), studentID, actividadID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if session == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "session_not_found"})
		}
		return c.JSON(toSessionResponse(session))
	}
}

func toSessionResponse(s *ExamSession) ExamSessionResponse {
	return ExamSessionResponse{
		ID:          s.ID.String(),
		ActividadID: s.ActividadID.String(),
		Status:      s.Status,
		StartedAt:   s.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
		WSURL:       "/sessions/" + s.ID.String() + "/ws",
	}
}

func parseUserID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("user_id").(string)
	return uuid.Parse(raw)
}
