package tracing

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OfflineBatchEvent struct {
	Type            string         `json:"type"`
	ClientTimestamp int64          `json:"client_timestamp"` // Unix milliseconds
	Metadata        map[string]any `json:"metadata"`
}

type batchRequest struct {
	Events []OfflineBatchEvent `json:"events"`
}

type batchResponse struct {
	Accepted          int `json:"accepted"`
	SkippedDuplicates int `json:"skipped_duplicates"`
}

// batchTraceHandler handles POST /student/sessions/:sid/trace/batch.
// Accepts offline-buffered trace events and stores them with received_offline=true.
func batchTraceHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sid, err := uuid.Parse(c.Params("sid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_session_id"})
		}

		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req batchRequest
		if err := c.BodyParser(&req); err != nil || len(req.Events) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_payload"})
		}
		if len(req.Events) > 1000 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "batch_too_large", "max": 1000})
		}

		// Verify the session belongs to this student
		session, err := svc.repo.GetSessionByID(c.Context(), sid)
		if err != nil || session == nil || session.StudentID != studentID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "session_not_found"})
		}

		// Convert to internal type
		internal := make([]offlineBatchEvent, len(req.Events))
		for i, e := range req.Events {
			clientTS := time.UnixMilli(e.ClientTimestamp).UTC()
			internal[i] = offlineBatchEvent{
				Type:            e.Type,
				ClientTimestamp: clientTS,
				Metadata:        e.Metadata,
			}
		}

		accepted, skipped, err := svc.repo.InsertBatch(c.Context(), sid, internal)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		return c.Status(fiber.StatusAccepted).JSON(batchResponse{
			Accepted:          accepted,
			SkippedDuplicates: skipped,
		})
	}
}

