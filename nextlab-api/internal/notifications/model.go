package notifications

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type StudentNotification struct {
	ID        uuid.UUID       `db:"id"         json:"id"`
	StudentID uuid.UUID       `db:"student_id" json:"-"`
	Type      string          `db:"type"       json:"type"`
	Payload   json.RawMessage `db:"payload"    json:"payload"`
	ReadAt    *time.Time      `db:"read_at"    json:"read_at"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// Publisher is the interface that other packages use to send notifications
// without importing the full notifications package (avoids circular imports).
type Publisher interface {
	Publish(studentID uuid.UUID, notifType string, payload any) error
}
