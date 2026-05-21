package ai

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all append-only DB operations for the AI router.
// No UPDATE or DELETE methods exist — Constitution Principle I.
type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetCurrentMode returns the active routing mode from the most recent router_configs row.
// Returns "local" and no error when the table is empty (pre-seed state).
func (r *Repository) GetCurrentMode(ctx context.Context) (string, error) {
	var mode string
	err := r.db.QueryRow(ctx,
		`SELECT mode FROM router_configs ORDER BY changed_at DESC LIMIT 1`,
	).Scan(&mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return "local", nil
	}
	return mode, err
}

// GetCurrentConfig returns the full latest router_configs row.
func (r *Repository) GetCurrentConfig(ctx context.Context) (*RouterConfig, error) {
	cfg := &RouterConfig{}
	err := r.db.QueryRow(ctx,
		`SELECT id, mode, prev_mode, changed_by, changed_at
		 FROM router_configs ORDER BY changed_at DESC LIMIT 1`,
	).Scan(&cfg.ID, &cfg.Mode, &cfg.PrevMode, &cfg.ChangedBy, &cfg.ChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return cfg, err
}

// InsertModeChange appends a new router_configs row. prevMode may be empty for the seed row.
func (r *Repository) InsertModeChange(ctx context.Context, mode string, prevMode *string, changedBy *uuid.UUID) (*RouterConfig, error) {
	cfg := &RouterConfig{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO router_configs (mode, prev_mode, changed_by)
		VALUES ($1, $2, $3)
		RETURNING id, mode, prev_mode, changed_by, changed_at
	`, mode, prevMode, changedBy).Scan(
		&cfg.ID, &cfg.Mode, &cfg.PrevMode, &cfg.ChangedBy, &cfg.ChangedAt,
	)
	return cfg, err
}

// LogRequest appends an immutable ai_request_logs row.
// Caller must never include prompt or completion content in the log entry.
func (r *Repository) LogRequest(ctx context.Context, entry *AIRequestLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_request_logs
		  (request_id, mode, backend_used, fallback_used, fallback_reason,
		   latency_ms, status, error_detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.RequestID, entry.Mode, entry.BackendUsed, entry.FallbackUsed,
		entry.FallbackReason, entry.LatencyMs, entry.Status, entry.ErrorDetail)
	return err
}

// GetConfigHistory returns router_configs rows ordered by changed_at DESC with pagination.
func (r *Repository) GetConfigHistory(ctx context.Context, limit, offset int) ([]*RouterConfig, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, mode, prev_mode, changed_by, changed_at
		FROM router_configs
		ORDER BY changed_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*RouterConfig
	for rows.Next() {
		cfg := &RouterConfig{}
		if err := rows.Scan(&cfg.ID, &cfg.Mode, &cfg.PrevMode, &cfg.ChangedBy, &cfg.ChangedAt); err != nil {
			return nil, 0, err
		}
		entries = append(entries, cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM router_configs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
