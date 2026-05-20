package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"

	"github.com/Elder2003/nextlab-api/internal/academic"
)

const oauthStateTTL = 60 * time.Second

func (s *Service) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  s.cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// GoogleLoginHandler redirects the user to the Google OAuth consent screen.
func GoogleLoginHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state, err := generateOAuthState()
		if err != nil {
			return internalError(c)
		}

		ctx := c.Context()
		if err := svc.rdb.Set(ctx, "oauth_state:"+state, "1", oauthStateTTL).Err(); err != nil {
			slog.ErrorContext(ctx, "failed to store oauth state", "err", err)
			return internalError(c)
		}

		url := svc.oauthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline)
		return c.Redirect(url, fiber.StatusFound)
	}
}

// GoogleCallbackHandler handles the OAuth callback, verifies identity, and issues tokens.
func GoogleCallbackHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		state := c.Query("state")
		code := c.Query("code")

		if state == "" || code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_state",
				Message:       "Missing state or code parameter",
				CorrelationID: correlationID(c),
			})
		}

		ctx := c.Context()

		// Validate and consume CSRF state atomically
		val, err := svc.rdb.GetDel(ctx, "oauth_state:"+state).Result()
		if err != nil || val == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "invalid_state",
				Message:       "OAuth state is missing, expired, or already used",
				CorrelationID: correlationID(c),
			})
		}

		// Exchange authorization code for tokens
		oauthToken, err := svc.oauthConfig().Exchange(context.Background(), code)
		if err != nil {
			slog.WarnContext(ctx, "oauth code exchange failed", "err", err, "correlation_id", correlationID(c))
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "oauth_exchange_failed",
				Message:       "Failed to exchange authorization code",
				CorrelationID: correlationID(c),
			})
		}

		rawIDToken, ok := oauthToken.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "oauth_exchange_failed",
				Message:       "No ID token returned by Google",
				CorrelationID: correlationID(c),
			})
		}

		// Verify ID token signature and audience
		payload, err := idtoken.Validate(context.Background(), rawIDToken, svc.cfg.GoogleClientID)
		if err != nil {
			slog.WarnContext(ctx, "id token validation failed", "err", err, "correlation_id", correlationID(c))
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "oauth_exchange_failed",
				Message:       "ID token validation failed",
				CorrelationID: correlationID(c),
			})
		}

		email, _ := payload.Claims["email"].(string)
		googleSub := payload.Subject

		if email == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:         "oauth_exchange_failed",
				Message:       "Email not returned by Google",
				CorrelationID: correlationID(c),
			})
		}

		// Verify student is provisioned by admin
		profile, err := academic.GetStudentProfileByEmail(ctx, svc.db, email)
		if err != nil {
			return internalError(c)
		}
		if profile == nil {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Error:         "account_not_provisioned",
				Message:       "This email has not been provisioned. Contact your administrator.",
				CorrelationID: correlationID(c),
			})
		}
		if !profile.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
				Error:         "account_inactive",
				Message:       "This student account has been deactivated",
				CorrelationID: correlationID(c),
			})
		}

		// Create or update user record
		user, err := UpsertGoogleUser(ctx, svc.db, email, googleSub)
		if err != nil {
			return internalError(c)
		}

		// Sync course enrollments based on current carrera+semester
		if err := academic.EnrollmentSync(ctx, svc.db, user.ID, profile.CarreraID, profile.Semester); err != nil {
			slog.ErrorContext(ctx, "enrollment sync failed",
				"err", err, "user_id", user.ID, "correlation_id", correlationID(c))
			return internalError(c)
		}

		resp, err := svc.issueTokenPair(ctx, user, nil)
		if err != nil {
			return internalError(c)
		}

		slog.InfoContext(ctx, "google oauth login",
			"event", "google_login", "email", email, "user_id", user.ID, "correlation_id", correlationID(c))
		return c.JSON(resp)
	}
}

func generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
