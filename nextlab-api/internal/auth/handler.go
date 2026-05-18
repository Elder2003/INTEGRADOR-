package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ─── US1: Login ──────────────────────────────────────────────────────────────

func LoginHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_request",
				Message:       "email and password are required",
				CorrelationID: correlationID(c),
			})
		}

		ip := c.IP()
		blocked, retryAfter, err := svc.CheckRateLimit(c.Context(), ip, req.Email)
		if err != nil {
			return internalError(c)
		}
		if blocked {
			return c.Status(fiber.StatusTooManyRequests).JSON(ErrorResponse{
				Error:         "rate_limit_exceeded",
				Message:       "Too many login attempts. Try again in 60 seconds.",
				CorrelationID: correlationID(c),
				RetryAfter:    &retryAfter,
			})
		}

		resp, err := svc.Authenticate(c.Context(), req.Email, req.Password, ip)
		if err != nil {
			_ = svc.RecordFailedAttempt(c.Context(), ip, req.Email)
			if errors.Is(err, ErrAccountInactive) {
				return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
					Error:         "account_inactive",
					Message:       "This account has been deactivated",
					CorrelationID: correlationID(c),
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:         "invalid_credentials",
				Message:       "Invalid email or password",
				CorrelationID: correlationID(c),
			})
		}

		return c.JSON(resp)
	}
}

// ─── US2: Refresh ─────────────────────────────────────────────────────────────

func RefreshHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RefreshRequest
		if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_request",
				Message:       "refresh_token is required",
				CorrelationID: correlationID(c),
			})
		}

		resp, err := svc.RotateRefreshToken(c.Context(), req.RefreshToken)
		if err != nil {
			if errors.Is(err, ErrTokenReuseDetected) {
				return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
					Error:         "token_reuse_detected",
					Message:       "Security event: refresh token reused. Session has been terminated.",
					CorrelationID: correlationID(c),
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:         "invalid_token",
				Message:       "Refresh token is invalid or expired",
				CorrelationID: correlationID(c),
			})
		}

		return c.JSON(resp)
	}
}

// ─── US2: Logout ──────────────────────────────────────────────────────────────

func LogoutHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LogoutRequest
		if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_request",
				Message:       "refresh_token is required",
				CorrelationID: correlationID(c),
			})
		}

		userIDStr, _ := c.Locals("user_id").(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Error:         "unauthorized",
				Message:       "Missing or invalid authorization token",
				CorrelationID: correlationID(c),
			})
		}

		if err := svc.RevokeSession(c.Context(), userID, req.RefreshToken); err != nil {
			return internalError(c)
		}

		return c.JSON(fiber.Map{"message": "Logged out successfully"})
	}
}

// ─── US3: /me ─────────────────────────────────────────────────────────────────

func MeHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":    c.Locals("user_id"),
			"email": c.Locals("user_email"),
			"role":  c.Locals("user_role"),
		})
	}
}

// ─── US4: Password recovery ───────────────────────────────────────────────────

func ForgotPasswordHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req ForgotPasswordRequest
		if err := c.BodyParser(&req); err != nil || req.Email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_request",
				Message:       "email is required",
				CorrelationID: correlationID(c),
			})
		}

		rawToken, err := svc.GeneratePasswordResetToken(c.Context(), req.Email)
		if err != nil {
			return internalError(c)
		}

		resp := fiber.Map{"message": "If an account with that email exists, a reset token has been generated."}
		if rawToken != "" {
			resp["reset_token"] = rawToken
		}
		return c.JSON(resp)
	}
}

func ResetPasswordHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req ResetPasswordRequest
		if err := c.BodyParser(&req); err != nil || req.Token == "" || req.NewPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_request",
				Message:       "token and new_password are required",
				CorrelationID: correlationID(c),
			})
		}

		err := svc.ResetPassword(c.Context(), req.Token, req.NewPassword)
		if err != nil {
			if err.Error() == "password_too_short" {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
					Error:         "validation_error",
					Message:       "new_password must be at least 8 characters",
					CorrelationID: correlationID(c),
				})
			}
			if errors.Is(err, ErrInvalidToken) {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
					Error:         "invalid_token",
					Message:       "Reset token is invalid, expired, or already used",
					CorrelationID: correlationID(c),
				})
			}
			return internalError(c)
		}

		return c.JSON(fiber.Map{"message": "Password updated successfully"})
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
