package appeals

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ReviewAppealHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		appeal, err := repo.GetAppeal(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if appeal == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		}
		if appeal.Status != "pending" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already_resolved"})
		}

		owns, err := repo.VerifyDocenteOwnsAppealSession(c.Context(), id, callerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		var req ReviewRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
		}
		if req.Decision != "upheld" && req.Decision != "rejected" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_decision", "message": "decision must be upheld or rejected"})
		}
		if req.Justification == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_justification"})
		}

		// Auto-escalate grade_dispute where docente owns the session (conflict of interest)
		if appeal.Type == "grade_dispute" {
			if err := repo.UpdateAppealStatus(c.Context(), id, "escalated"); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
			}
			appeal.Status = "escalated"
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"appeal": appeal})
		}

		decision, err := repo.InsertDecision(c.Context(), id, callerID, "docente", req.Decision, req.Justification)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if err := repo.UpdateAppealStatus(c.Context(), id, "resolved"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		appeal.Status = "resolved"

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"appeal": appeal, "decision": decision})
	}
}

func EscalateAppealHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		appeal, err := repo.GetAppeal(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if appeal == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		}
		if appeal.Status != "pending" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already_resolved"})
		}

		owns, err := repo.VerifyDocenteOwnsAppealSession(c.Context(), id, callerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		var req EscalateRequest
		c.BodyParser(&req) //nolint:errcheck — notes are optional

		if err := repo.UpdateAppealStatus(c.Context(), id, "escalated"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		appeal.Status = "escalated"

		return c.Status(fiber.StatusOK).JSON(appeal)
	}
}

func AdminReviewAppealHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		// roleGuard("admin") already enforces admin-only; this is a belt-and-suspenders check
		if roleFromLocals(c) != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		appeal, err := repo.GetAppeal(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if appeal == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		}
		if appeal.Status != "escalated" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "already_resolved"})
		}

		var req ReviewRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_body"})
		}
		if req.Decision != "upheld" && req.Decision != "rejected" && req.Decision != "overridden" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_decision"})
		}
		if req.Justification == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_justification"})
		}

		decision, err := repo.InsertDecision(c.Context(), id, callerID, "admin", req.Decision, req.Justification)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if err := repo.UpdateAppealStatus(c.Context(), id, "admin_resolved"); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		appeal.Status = "admin_resolved"

		return c.Status(fiber.StatusOK).JSON(fiber.Map{"appeal": appeal, "decision": decision})
	}
}
