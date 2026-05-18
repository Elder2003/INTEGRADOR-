package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/Elder2003/nextlab-api/internal/config"
)

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:         "unauthorized",
				Message:       "Missing or invalid authorization token",
				CorrelationID: correlationID(c),
			})
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims := &AccessTokenClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:         "unauthorized",
				Message:       "Missing or invalid authorization token",
				CorrelationID: correlationID(c),
			})
		}

		c.Locals("user_id", claims.Sub)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}

// RoleGuard returns a middleware that allows access only to users with one of the given roles.
// admin always passes. co_docente passes any guard that includes "docente".
func RoleGuard(roles ...string) fiber.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("user_role").(string)

		if role == "admin" {
			return c.Next()
		}
		if role == "co_docente" && allowed["docente"] {
			return c.Next()
		}
		if allowed[role] {
			return c.Next()
		}

		return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
			Error:         "forbidden",
			Message:       "Insufficient permissions for this resource",
			CorrelationID: correlationID(c),
		})
	}
}
