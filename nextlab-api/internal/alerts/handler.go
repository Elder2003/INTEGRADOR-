package alerts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func errResp(c *fiber.Ctx, status int, code, message string) error {
	corrID, _ := c.Locals("correlation_id").(string)
	return c.Status(status).JSON(fiber.Map{
		"error":          code,
		"message":        message,
		"correlation_id": corrID,
	})
}

// StreamTokenHandler issues a short-lived one-time SSE auth token.
func StreamTokenHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := claimsDocenteID(c)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
		}
		token, err := svc.IssueStreamToken(c.Context(), docenteID)
		if err != nil {
			return errResp(c, fiber.StatusInternalServerError, "token_error", "could not issue stream token")
		}
		return c.JSON(fiber.Map{"token": token})
	}
}

// SSEStreamHandler opens a persistent SSE connection authenticated via a short-lived query-param token.
// On connect it replays alerts with id > Last-Event-ID, then streams live pub/sub events.
func SSEStreamHandler(svc *Service, repo *Repository, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Query("token")
		if token == "" {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "missing token")
		}
		docenteID, err := svc.ValidateStreamToken(c.Context(), token)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid or expired token")
		}

		sinceID := int64(0)
		if raw := c.Get("Last-Event-ID"); raw != "" {
			if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
				sinceID = n
			}
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")

		ctx := c.Context()
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			// 1. Replay backlog
			backlog, err := repo.GetAlertsSince(ctx, docenteID, sinceID, false, 500)
			if err != nil {
				slog.Error("sse backlog query failed", "err", err)
			}
			for _, a := range backlog {
				writeSSEEvent(w, a)
			}
			_ = w.Flush()

			// 2. Subscribe to live Redis channel
			sub := rdb.Subscribe(ctx, "alerts:"+docenteID.String())
			defer sub.Close()
			ch := sub.Channel()

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-ch:
					if !ok {
						return
					}
					var payload map[string]any
					if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
						continue
					}
					id, _ := payload["alert_id"].(float64)
					alertType, _ := payload["type"].(string)
					fmt.Fprintf(w, "id: %d\n", int64(id))
					fmt.Fprintf(w, "event: %s\n", alertType)
					fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
					_ = w.Flush()
				case <-ticker.C:
					// heartbeat to keep connection alive
					fmt.Fprintf(w, ": ping\n\n")
					_ = w.Flush()
				}
			}
		})
		return nil
	}
}

// GetAlertsHandler returns a paginated list of alerts for the authenticated docente.
func GetAlertsHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := claimsDocenteID(c)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
		}
		sinceID, _ := strconv.ParseInt(c.Query("since_id", "0"), 10, 64)
		unreadOnly := c.Query("unread_only") == "true"
		limit, _ := strconv.Atoi(c.Query("limit", "50"))

		alerts, err := repo.GetAlertsSince(c.Context(), docenteID, sinceID, unreadOnly, limit)
		if err != nil {
			return errResp(c, fiber.StatusInternalServerError, "query_error", err.Error())
		}
		return c.JSON(fiber.Map{"alerts": alerts, "total": len(alerts)})
	}
}

// MarkReadHandler marks a single alert as read.
func MarkReadHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, err := claimsDocenteID(c)
		if err != nil {
			return errResp(c, fiber.StatusUnauthorized, "unauthorized", "invalid token claims")
		}
		alertID, err := strconv.ParseInt(c.Params("id"), 10, 64)
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "alert id must be an integer")
		}
		if err := repo.MarkAlertRead(c.Context(), alertID, docenteID); err != nil {
			if err == ErrAlertNotFound {
				return errResp(c, fiber.StatusNotFound, "not_found", "alert not found")
			}
			return errResp(c, fiber.StatusInternalServerError, "update_error", err.Error())
		}
		return c.JSON(fiber.Map{"ok": true})
	}
}

// writeSSEEvent formats a single Alert as SSE lines.
func writeSSEEvent(w *bufio.Writer, a Alert) {
	data, _ := json.Marshal(ssePayload(a))
	fmt.Fprintf(w, "id: %d\n", a.ID)
	fmt.Fprintf(w, "event: %s\n", string(a.Type))
	fmt.Fprintf(w, "data: %s\n\n", string(data))
}

// claimsDocenteID extracts the authenticated user's UUID from Fiber locals set by JWT middleware.
func claimsDocenteID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Locals("userID")
	if raw == nil {
		return uuid.Nil, ErrForbidden
	}
	switch v := raw.(type) {
	case uuid.UUID:
		return v, nil
	case string:
		return uuid.Parse(v)
	default:
		return uuid.Nil, ErrForbidden
	}
}
