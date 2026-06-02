package ai

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// NewAIRateLimiter returns a Fiber middleware that limits AI endpoint calls per user.
// Uses Redis INCR/EXPIRE with a sliding window. Fail-open if Redis is unavailable.
func NewAIRateLimiter(rdb *redis.Client, requests int, windowSec int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			return c.Next()
		}

		key := "rl:ai:" + userID
		count, err := rdb.Incr(c.Context(), key).Result()
		if err != nil {
			return c.Next() // fail-open: Redis not available
		}
		if count == 1 {
			rdb.Expire(c.Context(), key, time.Duration(windowSec)*time.Second)
		}
		if int(count) > requests {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "Too many AI requests. Try again later.",
			})
		}
		return c.Next()
	}
}
