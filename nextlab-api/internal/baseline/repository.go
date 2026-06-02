package baseline

import (
	"context"
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

func (r *Repository) Get(ctx context.Context, studentID uuid.UUID) (*StudentBehaviorBaseline, error) {
	var b StudentBehaviorBaseline
	err := r.db.QueryRow(ctx, `
		SELECT student_id, avg_tab_blurs_per_hour, stddev_tab_blurs,
		       avg_clipboard_per_hour, stddev_clipboard,
		       last_10_tab_blurs, last_10_clipboard,
		       total_sessions_analyzed, updated_at
		FROM student_behavior_baseline
		WHERE student_id = $1
	`, studentID).Scan(
		&b.StudentID, &b.AvgTabBlursPerHour, &b.StddevTabBlurs,
		&b.AvgClipboardPerHour, &b.StddevClipboard,
		&b.Last10TabBlurs, &b.Last10Clipboard,
		&b.TotalSessionsAnalyzed, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *Repository) Upsert(ctx context.Context, b *StudentBehaviorBaseline) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO student_behavior_baseline
			(student_id, avg_tab_blurs_per_hour, stddev_tab_blurs,
			 avg_clipboard_per_hour, stddev_clipboard,
			 last_10_tab_blurs, last_10_clipboard,
			 total_sessions_analyzed, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		ON CONFLICT (student_id) DO UPDATE SET
			avg_tab_blurs_per_hour  = EXCLUDED.avg_tab_blurs_per_hour,
			stddev_tab_blurs        = EXCLUDED.stddev_tab_blurs,
			avg_clipboard_per_hour  = EXCLUDED.avg_clipboard_per_hour,
			stddev_clipboard        = EXCLUDED.stddev_clipboard,
			last_10_tab_blurs       = EXCLUDED.last_10_tab_blurs,
			last_10_clipboard       = EXCLUDED.last_10_clipboard,
			total_sessions_analyzed = EXCLUDED.total_sessions_analyzed,
			updated_at              = NOW()
	`,
		b.StudentID, b.AvgTabBlursPerHour, b.StddevTabBlurs,
		b.AvgClipboardPerHour, b.StddevClipboard,
		b.Last10TabBlurs, b.Last10Clipboard,
		b.TotalSessionsAnalyzed,
	)
	return err
}
