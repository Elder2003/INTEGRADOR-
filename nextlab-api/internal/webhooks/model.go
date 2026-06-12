package webhooks

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WebhookConfig struct {
	ID         uuid.UUID       `db:"id"         json:"id"`
	URL        string          `db:"url"        json:"url"`
	SecretHmac string          `db:"secret_hmac" json:"-"`
	Events     []string        `db:"events"     json:"events"`
	IsActive   bool            `db:"is_active"  json:"is_active"`
	CreatedBy  *uuid.UUID      `db:"created_by" json:"created_by,omitempty"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at" json:"updated_at"`
}

type WebhookDeliveryLog struct {
	ID             uuid.UUID       `db:"id"`
	WebhookID      uuid.UUID       `db:"webhook_id"`
	EventType      string          `db:"event_type"`
	Payload        json.RawMessage `db:"payload"`
	StatusCode     *int            `db:"status_code"`
	AttemptCount   int             `db:"attempt_count"`
	Status         string          `db:"status"`
	LastAttemptAt  time.Time       `db:"last_attempt_at"`
	CreatedAt      time.Time       `db:"created_at"`
}

type CreateWebhookRequest struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}
