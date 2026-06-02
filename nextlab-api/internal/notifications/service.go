package notifications

import (
	"context"
	"encoding/json"
	"log/slog"

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

// Publish stores a notification in the DB and broadcasts it via Redis pub/sub.
// This method satisfies the Publisher interface; it is fire-and-forget safe to call
// from goroutines (errors are logged, not returned to callers).
func (s *Service) Publish(studentID uuid.UUID, notifType string, payload any) error {
	ctx := context.Background()

	raw, err := json.Marshal(payload)
	if err != nil {
		slog.Error("notifications: marshal payload", "err", err)
		return err
	}

	notif, err := s.repo.Insert(ctx, studentID, notifType, raw)
	if err != nil {
		slog.Error("notifications: insert", "err", err)
		return err
	}

	msg, _ := json.Marshal(notif)
	channel := "notifications:student:" + studentID.String()
	if err := s.rdb.Publish(ctx, channel, msg).Err(); err != nil {
		// Not fatal — notification is persisted; client will see it on next poll.
		slog.Warn("notifications: redis publish failed", "channel", channel, "err", err)
	}

	return nil
}
