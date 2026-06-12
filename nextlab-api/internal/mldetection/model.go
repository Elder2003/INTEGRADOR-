package mldetection

import (
	"time"

	"github.com/google/uuid"
)

type MLModel struct {
	ID                   uuid.UUID `db:"id"`
	Version              int       `db:"version"`
	TrainedAt            time.Time `db:"trained_at"`
	TrainingSessionCount int       `db:"training_session_count"`
	FilePath             string    `db:"file_path"`
	FeatureNames         []string  `db:"feature_names"`
	IsActive             bool      `db:"is_active"`
	CreatedAt            time.Time `db:"created_at"`
}

var defaultFeatureNames = []string{
	"tab_blurs_rate",
	"clipboard_rate",
	"heartbeat_gap_avg",
	"session_duration_min",
	"burst_windows",
	"events_per_minute",
}
