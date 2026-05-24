package appeals

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// verifyChain recomputes each event_hash and returns true if all match.
func verifyChain(events []AuditEvent) bool {
	for _, e := range events {
		payloadBytes, err := json.Marshal(e.Payload)
		if err != nil {
			return false
		}
		prevHash := ""
		if e.PrevHash != nil {
			prevHash = *e.PrevHash
		}
		expected := ComputeHash(
			e.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			e.EventType,
			e.Sequence,
			string(payloadBytes),
			prevHash,
		)
		if expected != e.EventHash {
			return false
		}
	}
	return true
}

func GetEvidenceHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		callerID, ok := callerIDFromLocals(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		role := roleFromLocals(c)

		appeal, err := repo.GetAppeal(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if appeal == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found"})
		}

		if role != "admin" {
			owns, err := repo.VerifyDocenteOwnsAppealSession(c.Context(), id, callerID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
			}
			if !owns {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
			}
		}

		events, err := repo.GetAuditEvents(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		valid := verifyChain(events)

		return c.Status(fiber.StatusOK).JSON(EvidenceChain{
			AppealID: id,
			Valid:    valid,
			Events:   events,
		})
	}
}
