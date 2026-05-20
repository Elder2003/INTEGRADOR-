package academic

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}

func isFKViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23503"
	}
	return false
}

func isInvalidEnumValue(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "22P02"
	}
	return false
}

func correlationID(c *fiber.Ctx) string {
	if v, ok := c.Locals("correlation_id").(string); ok {
		return v
	}
	return ""
}

func internalError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Error:         "internal_error",
		Message:       "An unexpected error occurred",
		CorrelationID: correlationID(c),
	})
}

func parseIntQuery(s string, out *int) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, err
	}
	*out = n
	return n, nil
}
