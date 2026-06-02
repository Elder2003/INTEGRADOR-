package webhooks

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, cfg *WebhookConfig) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO webhook_configs (url, secret_hmac, events, is_active, created_by)
		VALUES ($1, $2, $3, TRUE, $4)
	`, cfg.URL, cfg.SecretHmac, cfg.Events, cfg.CreatedBy)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*WebhookConfig, error) {
	var wh WebhookConfig
	err := r.db.QueryRow(ctx, `
		SELECT id, url, secret_hmac, events, is_active, created_by, created_at, updated_at
		FROM webhook_configs WHERE id = $1
	`, id).Scan(&wh.ID, &wh.URL, &wh.SecretHmac, &wh.Events, &wh.IsActive, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &wh, err
}

func (r *Repository) ListActive(ctx context.Context) ([]WebhookConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, url, secret_hmac, events, is_active, created_by, created_at, updated_at
		FROM webhook_configs WHERE is_active = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookConfig
	for rows.Next() {
		var wh WebhookConfig
		if err := rows.Scan(&wh.ID, &wh.URL, &wh.SecretHmac, &wh.Events, &wh.IsActive, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, wh)
	}
	return out, rows.Err()
}

func (r *Repository) ListActiveForEvent(ctx context.Context, eventType string) ([]WebhookConfig, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, url, secret_hmac, events, is_active, created_by, created_at, updated_at
		FROM webhook_configs
		WHERE is_active = TRUE AND $1 = ANY(events)
	`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookConfig
	for rows.Next() {
		var wh WebhookConfig
		if err := rows.Scan(&wh.ID, &wh.URL, &wh.SecretHmac, &wh.Events, &wh.IsActive, &wh.CreatedBy, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, wh)
	}
	return out, rows.Err()
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE webhook_configs SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *Repository) InsertDeliveryLog(ctx context.Context, log *WebhookDeliveryLog) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO webhook_delivery_log (webhook_id, event_type, payload, status, attempt_count)
		VALUES ($1, $2, $3, 'pending', 1)
		RETURNING id
	`, log.WebhookID, log.EventType, json.RawMessage(log.Payload)).Scan(&id)
	return id, err
}

func (r *Repository) UpdateDeliveryLog(ctx context.Context, id uuid.UUID, status string, statusCode *int, attemptCount int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE webhook_delivery_log
		SET status = $2, status_code = $3, attempt_count = $4, last_attempt_at = NOW()
		WHERE id = $1
	`, id, status, statusCode, attemptCount)
	return err
}
