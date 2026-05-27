package detection

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type detectionHandler struct {
	svc *DetectionService
}

func newDetectionHandler(svc *DetectionService) *detectionHandler {
	return &detectionHandler{svc: svc}
}

func (h *detectionHandler) report(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("sessionID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid session_id"})
	}

	docenteID, ok := userIDFromLocals(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	report, err := h.svc.GetReport(c.Context(), sessionID, docenteID)
	if err != nil {
		switch err {
		case ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
	}

	return c.Status(fiber.StatusOK).JSON(report)
}

func (h *detectionHandler) grade(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("sessionID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid session_id"})
	}

	docenteID, ok := userIDFromLocals(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req GradeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
	}

	result, err := h.svc.SubmitGrade(c.Context(), sessionID, docenteID, req)
	if err != nil {
		switch err {
		case ErrForbidden:
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case ErrNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "llm_result_not_found"})
		case ErrConflict:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "grade_already_submitted"})
		case ErrOverrideRequired:
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error":   "override_reason_required",
				"message": "Grade is outside the suggested range; provide an override_reason",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *detectionHandler) gradeLog(c *fiber.Ctx) error {
	sessionID, err := uuid.Parse(c.Params("sessionID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid session_id"})
	}

	callerID, ok := userIDFromLocals(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	role, _ := c.Locals("user_role").(string)
	if role != "admin" {
		owns, err := h.svc.repo.VerifyDocenteOwnsSession(c.Context(), sessionID, callerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
	}

	g, err := h.svc.repo.GetGradeDecision(c.Context(), sessionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
	}
	if g == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
	}

	var suggested string
	if g.SuggestedGradeRange != nil {
		suggested = *g.SuggestedGradeRange
	}
	result := GradeLogResult{
		SessionID:           sessionID,
		FinalGrade:          g.FinalGrade,
		SuggestedGradeRange: suggested,
		OverrideReason:      g.OverrideReason,
		DecidedBy:           g.DecidedBy,
		DecidedAt:           g.DecidedAt,
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func userIDFromLocals(c *fiber.Ctx) (uuid.UUID, bool) {
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
