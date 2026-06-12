package baseline

import (
	"time"

	"github.com/google/uuid"
)

type StudentBehaviorBaseline struct {
	StudentID             uuid.UUID `db:"student_id"`
	AvgTabBlursPerHour    float64   `db:"avg_tab_blurs_per_hour"`
	StddevTabBlurs        float64   `db:"stddev_tab_blurs"`
	AvgClipboardPerHour   float64   `db:"avg_clipboard_per_hour"`
	StddevClipboard       float64   `db:"stddev_clipboard"`
	Last10TabBlurs        []float64 `db:"last_10_tab_blurs"`
	Last10Clipboard       []float64 `db:"last_10_clipboard"`
	TotalSessionsAnalyzed int       `db:"total_sessions_analyzed"`
	UpdatedAt             time.Time `db:"updated_at"`
}
