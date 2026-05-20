package tracing

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Elder2003/nextlab-api/internal/auth"
	"github.com/Elder2003/nextlab-api/internal/config"
)

const (
	rateLimit   = 30
	rateWindow  = time.Minute
	readTimeout = 2 * time.Minute
)

// wsJWTMiddleware validates the ?token= query param before the WebSocket upgrade.
// The token is NEVER logged.
func wsJWTMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Query("token")
		if token == "" {
			return fiber.ErrUnauthorized
		}

		claims := &auth.AccessTokenClaims{}
		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !parsed.Valid {
			return fiber.ErrUnauthorized
		}

		c.Locals("user_id", claims.Sub)
		c.Locals("user_role", claims.Role)
		return c.Next()
	}
}

// makeWSHandler returns the per-connection WebSocket handler for /sessions/:sid/ws.
func makeWSHandler(hub *Hub, svc *Service) func(*websocket.Conn) {
	return func(c *websocket.Conn) {
		ctx := context.Background()

		sidStr := c.Params("sid")
		sessionID, err := uuid.Parse(sidStr)
		if err != nil {
			sendJSON(c, WSError{Type: "error", Code: "invalid_session", Message: "invalid session id"})
			c.Close()
			return
		}

		userID, err := uuid.Parse(c.Locals("user_id").(string))
		if err != nil {
			sendJSON(c, WSError{Type: "error", Code: "unauthorized", Message: "invalid user"})
			c.Close()
			return
		}

		// Verify the session belongs to this user and is active
		session, err := svc.repo.GetSessionByID(ctx, sessionID)
		if err != nil || session == nil || session.StudentID != userID || session.Status != "active" {
			sendJSON(c, WSError{Type: "error", Code: "session_not_found", Message: "session not found or not active"})
			c.Close()
			return
		}

		hub.Add(sidStr, c)
		gracefulClose := false
		defer func() {
			hub.Remove(sidStr, c)
			if !gracefulClose {
				svc.CloseSessionAbandoned(ctx, sessionID)
			}
		}()

		// Append session_open event (best effort)
		if _, err := svc.AppendEvent(ctx, sessionID, "session_open", nil); err != nil {
			slog.Error("failed to append session_open", "session_id", sessionID, "err", err)
		}

		sendJSON(c, WSConnected{
			Type:       "connected",
			SessionID:  sidStr,
			EventCount: session.EventCount,
		})

		// Per-connection rate limiter
		rateCount := 0
		rateReset := time.Now().Add(rateWindow)

		c.SetReadDeadline(time.Now().Add(readTimeout))

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			if mt != websocket.TextMessage {
				continue
			}

			c.SetReadDeadline(time.Now().Add(readTimeout))

			// Rate limiting window
			now := time.Now()
			if now.After(rateReset) {
				rateCount = 0
				rateReset = now.Add(rateWindow)
			}
			rateCount++
			if rateCount > rateLimit {
				sendJSON(c, WSError{Type: "error", Code: "rate_limit_exceeded", Message: "too many events"})
				continue
			}

			var event ClientEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				sendJSON(c, WSError{Type: "error", Code: "invalid_json", Message: "malformed message"})
				continue
			}
			if event.Type == "" {
				sendJSON(c, WSError{Type: "error", Code: "missing_type", Message: "event type required"})
				continue
			}

			// Client-initiated graceful close
			if event.Type == "session_close" {
				gracefulClose = true
				_, _ = svc.AppendEvent(ctx, sessionID, "session_close", nil)
				sendJSON(c, WSSessionClosed{Type: "session_closed", Reason: "client_close", ClosedAt: time.Now().UTC()})
				break
			}

			// Build metadata from full raw payload (minus protocol fields)
			var rawMap map[string]any
			_ = json.Unmarshal(msg, &rawMap)
			delete(rawMap, "type")
			delete(rawMap, "client_ts")

			ack, err := svc.AppendEvent(ctx, sessionID, event.Type, rawMap)
			if err != nil {
				if err == ErrSessionClosed {
					sendJSON(c, WSSessionClosed{Type: "session_closed", Reason: "session_already_closed", ClosedAt: time.Now().UTC()})
					break
				}
				sendJSON(c, WSError{Type: "error", Code: "internal_error", Message: "failed to record event"})
				continue
			}

			sendJSON(c, ack)
		}
	}
}

func sendJSON(c *websocket.Conn, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = c.WriteMessage(websocket.TextMessage, b)
}
