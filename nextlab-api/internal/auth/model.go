package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         string    `db:"role"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TokenHash  string     `db:"token_hash"`
	ReplacedBy *uuid.UUID `db:"replaced_by"`
	ExpiresAt  time.Time  `db:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type PasswordResetToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

type AccessTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	JTI  string `json:"jti"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type ErrorResponse struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
	RetryAfter    *int   `json:"retry_after,omitempty"`
}
