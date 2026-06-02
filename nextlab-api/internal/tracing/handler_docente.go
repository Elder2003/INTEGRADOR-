package tracing

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func getTraceHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		submissionID, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_submission_id"})
		}
		callerID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role, _ := c.Locals("user_role").(string)

		session, events, err := svc.GetTraceBySubmission(c.Context(), submissionID, callerID, role)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if session == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trace_not_found"})
		}

		items := make([]TraceEventItem, len(events))
		for i, e := range events {
			items[i] = TraceEventItem{
				SequenceNum: e.SequenceNum,
				EventType:   e.EventType,
				ReceivedAt:  e.ReceivedAt,
				Metadata:    e.Metadata,
				EventHash:   e.EventHash,
			}
		}

		return c.JSON(TraceResponse{
			SessionID:   session.ID.String(),
			StudentID:   session.StudentID.String(),
			ActividadID: session.ActividadID.String(),
			Status:      session.Status,
			StartedAt:   session.StartedAt,
			ClosedAt:    session.ClosedAt,
			EventCount:  session.EventCount,
			Events:      items,
		})
	}
}

func getTracePagedHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		submissionID, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_submission_id"})
		}
		callerID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role, _ := c.Locals("user_role").(string)

		cursorStr := c.Query("cursor", "0")
		cursor, err := parseIntQuery(cursorStr)
		if err != nil || cursor < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_cursor"})
		}

		limitStr := c.Query("limit", "100")
		limit, err := parseIntQuery(limitStr)
		if err != nil || limit < 1 || limit > 200 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_limit"})
		}

		session, _, err := svc.GetTraceBySubmission(c.Context(), submissionID, callerID, role)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if session == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trace_not_found"})
		}

		events, hasMore, err := svc.repo.GetEventsPaginated(c.Context(), session.ID, cursor, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		items := make([]TraceEventItem, len(events))
		for i, e := range events {
			items[i] = TraceEventItem{
				SequenceNum: e.SequenceNum,
				EventType:   e.EventType,
				ReceivedAt:  e.ReceivedAt,
				Metadata:    e.Metadata,
				EventHash:   e.EventHash,
			}
		}

		page := TraceEventsPage{
			SessionID: session.ID.String(),
			Data:      items,
			HasMore:   hasMore,
			Total:     session.EventCount,
		}
		if hasMore && len(items) > 0 {
			last := items[len(items)-1].SequenceNum
			page.NextCursor = &last
		}

		return c.JSON(page)
	}
}

func parseIntQuery(s string) (int, error) {
	n, err := strconv.Atoi(s)
	return n, err
}

func verifyTraceHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		submissionID, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_submission_id"})
		}
		callerID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role, _ := c.Locals("user_role").(string)

		// Resolve session from submission (role-scoped)
		session, _, err := svc.GetTraceBySubmission(c.Context(), submissionID, callerID, role)
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if session == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "trace_not_found"})
		}

		valid, eventCount, firstInvalid := svc.VerifyChain(c.Context(), session.ID)

		resp := VerifyResponse{Valid: valid, EventCount: eventCount}
		if !valid {
			resp.FirstInvalidSequence = &firstInvalid
		}
		return c.JSON(resp)
	}
}
