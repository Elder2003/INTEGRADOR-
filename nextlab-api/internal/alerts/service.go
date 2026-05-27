package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo *Repository
	rdb  *redis.Client
}

func NewService(repo *Repository, rdb *redis.Client) *Service {
	return &Service{repo: repo, rdb: rdb}
}

// WriteAlert persists the alert and publishes it to the docente's Redis pub/sub channel.
func (s *Service) WriteAlert(ctx context.Context, a Alert) error {
	id, err := s.repo.InsertAlert(ctx, a)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	a.ID = id

	msg, err := json.Marshal(ssePayload(a))
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}
	// Fire-and-forget publish — delivery failure does not affect the DB record.
	_ = s.rdb.Publish(ctx, "alerts:"+a.DocenteID.String(), string(msg)).Err()
	return nil
}

// IssueStreamToken generates a short-lived one-time token for SSE authentication.
func (s *Service) IssueStreamToken(ctx context.Context, docenteID uuid.UUID) (string, error) {
	token := uuid.New().String()
	key := "sse_token:" + token
	if err := s.rdb.Set(ctx, key, docenteID.String(), 60*time.Second).Err(); err != nil {
		return "", fmt.Errorf("store sse token: %w", err)
	}
	return token, nil
}

// ValidateStreamToken atomically reads and deletes the token, returning the associated docenteID.
// Returns ErrForbidden if the token is missing or expired.
func (s *Service) ValidateStreamToken(ctx context.Context, token string) (uuid.UUID, error) {
	val, err := s.rdb.GetDel(ctx, "sse_token:"+token).Result()
	if err == redis.Nil || val == "" {
		return uuid.Nil, ErrForbidden
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("validate sse token: %w", err)
	}
	docenteID, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, ErrForbidden
	}
	return docenteID, nil
}

// ssePayload builds the JSON object written as SSE data.
func ssePayload(a Alert) map[string]any {
	return map[string]any{
		"alert_id":     a.ID,
		"actividad_id": a.ActividadID,
		"student_id":   a.StudentID,
		"type":         string(a.Type),
		"payload":      a.Payload,
		"advisory":     a.Advisory,
		"created_at":   a.CreatedAt,
	}
}
