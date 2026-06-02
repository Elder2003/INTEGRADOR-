package notifications

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func StreamHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no")

		channel := "notifications:student:" + studentID.String()

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sub := svc.rdb.Subscribe(ctx, channel)
			defer sub.Close()

			ch := sub.Channel()
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case msg, ok := <-ch:
					if !ok {
						return
					}
					fmt.Fprintf(w, "event: notification\ndata: %s\n\n", msg.Payload)
					w.Flush()
				case <-ticker.C:
					fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
					w.Flush()
				case <-ctx.Done():
					return
				}
			}
		})

		return nil
	}
}

func ListHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		limit := c.QueryInt("limit", 20)
		if limit < 1 || limit > 100 {
			limit = 20
		}

		var cursor *uuid.UUID
		if cursorStr := c.Query("cursor"); cursorStr != "" {
			id, err := uuid.Parse(cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_cursor"})
			}
			cursor = &id
		}

		notifications, hasMore, nextCursor, err := repo.GetPaginated(c.Context(), studentID, cursor, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		unread, _ := repo.CountUnread(c.Context(), studentID)

		resp := fiber.Map{
			"data":         notifications,
			"has_more":     hasMore,
			"unread_count": unread,
		}
		if nextCursor != nil {
			resp["next_cursor"] = nextCursor.String()
		}

		return c.JSON(resp)
	}
}

func ReadHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		studentID, err := parseUserID(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_id"})
		}

		updated, err := repo.MarkRead(c.Context(), id, studentID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}
		if !updated {
			// Either not found or already read — idempotent
			return c.JSON(fiber.Map{"ok": true})
		}

		return c.JSON(fiber.Map{"ok": true})
	}
}

func parseUserID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("user_id").(string)
	return uuid.Parse(raw)
}

