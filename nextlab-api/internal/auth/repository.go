package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetUserByEmail(ctx context.Context, db *pgxpool.Pool, email string) (*User, error) {
	u := &User{}
	err := db.QueryRow(ctx,
		`SELECT id, email, password_hash, role, is_active, google_sub, created_at, updated_at
		 FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.GoogleSub, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpsertGoogleUser creates or updates a student user record after Google OAuth verification.
func UpsertGoogleUser(ctx context.Context, db *pgxpool.Pool, email, googleSub string) (*User, error) {
	u := &User{}
	err := db.QueryRow(ctx,
		`INSERT INTO users (email, role, google_sub)
		 VALUES ($1, 'estudiante', $2)
		 ON CONFLICT (email) DO UPDATE
		   SET google_sub = EXCLUDED.google_sub, updated_at = NOW()
		 RETURNING id, email, password_hash, role, is_active, google_sub, created_at, updated_at`,
		email, googleSub).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.GoogleSub, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func CreateRefreshToken(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, tokenHash string, replacedBy *uuid.UUID, expiresAt time.Time) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := db.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, replaced_by, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, token_hash, replaced_by, expires_at, revoked_at, created_at`,
		userID, tokenHash, replacedBy, expiresAt).
		Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ReplacedBy, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func GetRefreshTokenByHash(ctx context.Context, db *pgxpool.Pool, tokenHash string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, replaced_by, expires_at, revoked_at, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ReplacedBy, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func RevokeRefreshToken(ctx context.Context, db *pgxpool.Pool, id uuid.UUID, replacedBy *uuid.UUID) error {
	_, err := db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW(), replaced_by = COALESCE($2, replaced_by)
		 WHERE id = $1`, id, replacedBy)
	return err
}

func RevokeAllUserRefreshTokens(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID) error {
	_, err := db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW()
		 WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func UpdateUserPassword(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, newHash string) error {
	_, err := db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newHash)
	return err
}

func CreatePasswordResetToken(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func GetPasswordResetTokenByHash(ctx context.Context, db *pgxpool.Pool, tokenHash string) (*PasswordResetToken, error) {
	prt := &PasswordResetToken{}
	err := db.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_reset_tokens WHERE token_hash = $1`, tokenHash).
		Scan(&prt.ID, &prt.UserID, &prt.TokenHash, &prt.ExpiresAt, &prt.UsedAt, &prt.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return prt, nil
}

func MarkPasswordResetTokenUsed(ctx context.Context, db *pgxpool.Pool, id uuid.UUID) error {
	_, err := db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}
