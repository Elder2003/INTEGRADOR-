package tracing

import (
	"errors"
	"strconv"
	"time"

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

func listActiveSessionsHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_activity_id"})
		}
		docenteID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		rows, err := svc.repo.GetActiveSessionsByActivity(c.Context(), actividadID, docenteID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		entries := make([]ActiveSessionEntry, len(rows))
		for i, row := range rows {
			entries[i] = ActiveSessionEntry{
				SessionID: row.ID.String(),
				Student: StudentInfo{
					ID:    row.StudentID.String(),
					Name:  row.StudentName,
					Email: row.StudentEmail,
				},
				StartedAt:  row.StartedAt,
				EventCount: row.EventCount,
			}
		}

		return c.JSON(ActiveSessionsListResponse{
			ActividadID: actividadID.String(),
			Sessions:    entries,
		})
	}
}

func getSessionEventsHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_session_id"})
		}
		docenteID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		owns, err := svc.repo.VerifyDocenteOwnsSession(c.Context(), sessionID, docenteID)
		if err != nil || !owns {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}

		events, err := svc.repo.GetAllEvents(c.Context(), sessionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		var sinceTime time.Time
		if s := c.Query("since"); s != "" {
			sinceTime, _ = time.Parse(time.RFC3339Nano, s)
		}

		items := make([]LiveEventItem, 0, len(events))
		for _, e := range events {
			if !sinceTime.IsZero() && !e.ReceivedAt.After(sinceTime) {
				continue
			}
			item := LiveEventItem{
				ID:        e.ID.String(),
				SessionID: e.SessionID.String(),
				Type:      e.EventType,
				Timestamp: e.ReceivedAt,
			}
			if v, ok := e.Metadata["content"].(string); ok {
				item.Content = &v
			}
			items = append(items, item)
		}

		return c.JSON(LiveEventsResponse{
			SessionID: sessionID.String(),
			Events:    items,
			Total:     len(items),
		})
	}
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
