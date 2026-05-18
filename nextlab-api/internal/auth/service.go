package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/Elder2003/nextlab-api/internal/config"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account inactive")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenReuseDetected = errors.New("token reuse detected")
)

// dummyHash is pre-computed to ensure constant-time comparison when email not found.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-constant-time-comparison"), 12)

type Service struct {
	db  *pgxpool.Pool
	rdb *redis.Client
	cfg *config.Config
}

func NewService(db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config) *Service {
	return &Service{db: db, rdb: rdb, cfg: cfg}
}

// ─── US1: Login ──────────────────────────────────────────────────────────────

func (s *Service) Authenticate(ctx context.Context, email, password, ip string) (*LoginResponse, error) {
	user, err := GetUserByEmail(ctx, s.db, email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		// Run dummy compare to maintain constant response time (anti-enumeration).
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		slog.WarnContext(ctx, "login failed: email not found",
			"event", "login_failed", "email", email, "ip", ip,
			"correlation_id", correlationIDFromCtx(ctx))
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		slog.WarnContext(ctx, "login failed: wrong password",
			"event", "login_failed", "email", email, "ip", ip,
			"correlation_id", correlationIDFromCtx(ctx))
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user, nil)
}

func (s *Service) issueTokenPair(ctx context.Context, user *User, replacedBy *uuid.UUID) (*LoginResponse, error) {
	jti := uuid.New().String()
	now := time.Now()
	exp := now.Add(s.cfg.AccessTokenTTL)

	claims := AccessTokenClaims{
		Sub:   user.ID.String(),
		Email: user.Email,
		Role:  user.Role,
		JTI:   jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	rawRT := make([]byte, 32)
	if _, err := rand.Read(rawRT); err != nil {
		return nil, err
	}
	rawRTStr := base64.RawURLEncoding.EncodeToString(rawRT)
	rtHash := hashToken(rawRTStr)

	if _, err := CreateRefreshToken(ctx, s.db, user.ID, rtHash, replacedBy, now.Add(s.cfg.RefreshTokenTTL)); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
		RefreshToken: rawRTStr,
	}, nil
}

// ─── Rate Limiting ────────────────────────────────────────────────────────────

func (s *Service) CheckRateLimit(ctx context.Context, ip, email string) (blocked bool, retryAfter int, err error) {
	_, coolKey := rateLimitKeys(ip, email)

	ttl, rErr := s.rdb.TTL(ctx, coolKey).Result()
	if rErr == nil && ttl > 0 {
		return true, int(ttl.Seconds()) + 1, nil
	}

	return false, 0, nil
}

func (s *Service) RecordFailedAttempt(ctx context.Context, ip, email string) error {
	key, coolKey := rateLimitKeys(ip, email)

	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if count == 1 {
		s.rdb.Expire(ctx, key, 60*time.Second)
	}
	if count > 5 {
		slog.WarnContext(ctx, "rate limit activated",
			"event", "rate_limit_hit", "ip", ip,
			"correlation_id", correlationIDFromCtx(ctx))
		s.rdb.Set(ctx, coolKey, "1", 60*time.Second)
	}
	return nil
}

func rateLimitKeys(ip, email string) (attemptKey, cooldownKey string) {
	h := sha256.Sum256([]byte(ip + ":" + email))
	suffix := hex.EncodeToString(h[:])[:16]
	return "rate_limit:" + suffix, "rate_cool:" + suffix
}

// ─── US2: Token rotation and logout ──────────────────────────────────────────

func (s *Service) RotateRefreshToken(ctx context.Context, rawToken string) (*LoginResponse, error) {
	rtHash := hashToken(rawToken)
	rt, err := GetRefreshTokenByHash(ctx, s.db, rtHash)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return nil, ErrInvalidToken
	}

	if rt.RevokedAt != nil {
		if rt.ReplacedBy != nil {
			// Reuse detected — cascade revoke the live replacement
			slog.WarnContext(ctx, "refresh token reuse detected",
				"event", "token_reuse_detected", "user_id", rt.UserID,
				"correlation_id", correlationIDFromCtx(ctx))
			_ = RevokeRefreshToken(ctx, s.db, *rt.ReplacedBy, nil)
			return nil, ErrTokenReuseDetected
		}
		return nil, ErrInvalidToken
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	user, err := getUserByID(ctx, s.db, rt.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, ErrInvalidToken
	}

	newResp, err := s.issueTokenPair(ctx, user, nil)
	if err != nil {
		return nil, err
	}

	// Look up new token id to set replaced_by on the old token
	newRTHash := hashToken(newResp.RefreshToken)
	newRT, err := GetRefreshTokenByHash(ctx, s.db, newRTHash)
	if err != nil || newRT == nil {
		return nil, errors.New("failed to locate new refresh token")
	}

	if err := RevokeRefreshToken(ctx, s.db, rt.ID, &newRT.ID); err != nil {
		return nil, err
	}

	return newResp, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID uuid.UUID, rawToken string) error {
	rtHash := hashToken(rawToken)
	rt, err := GetRefreshTokenByHash(ctx, s.db, rtHash)
	if err != nil {
		return err
	}
	if rt == nil || rt.UserID != userID {
		return nil // idempotent
	}
	if rt.RevokedAt != nil {
		return nil // already revoked
	}
	slog.InfoContext(ctx, "session revoked",
		"event", "session_revoked", "user_id", userID,
		"correlation_id", correlationIDFromCtx(ctx))
	return RevokeRefreshToken(ctx, s.db, rt.ID, nil)
}

// ─── US4: Password recovery ───────────────────────────────────────────────────

func (s *Service) GeneratePasswordResetToken(ctx context.Context, email string) (string, error) {
	user, err := GetUserByEmail(ctx, s.db, email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", nil // no-op, no enumeration
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawToken := "prt_" + base64.RawURLEncoding.EncodeToString(raw)
	tokenHash := hashToken(rawToken)

	if err := CreatePasswordResetToken(ctx, s.db, user.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password_too_short")
	}

	tokenHash := hashToken(rawToken)
	prt, err := GetPasswordResetTokenByHash(ctx, s.db, tokenHash)
	if err != nil {
		return err
	}
	if prt == nil || prt.UsedAt != nil || time.Now().After(prt.ExpiresAt) {
		return ErrInvalidToken
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	if err := UpdateUserPassword(ctx, s.db, prt.UserID, string(newHash)); err != nil {
		return err
	}
	if err := MarkPasswordResetTokenUsed(ctx, s.db, prt.ID); err != nil {
		return err
	}
	return RevokeAllUserRefreshTokens(ctx, s.db, prt.UserID)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func getUserByID(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) (*User, error) {
	u := &User{}
	err := db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, is_active, created_at, updated_at
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func correlationIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDKey{}).(string); ok {
		return v
	}
	return ""
}

type correlationIDKey struct{}
