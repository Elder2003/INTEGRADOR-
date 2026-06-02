package mldetection

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetActive(ctx context.Context) (*MLModel, error) {
	var m MLModel
	err := r.db.QueryRow(ctx, `
		SELECT id, version, trained_at, training_session_count,
		       file_path, feature_names, is_active, created_at
		FROM ml_models WHERE is_active = TRUE LIMIT 1
	`).Scan(&m.ID, &m.Version, &m.TrainedAt, &m.TrainingSessionCount,
		&m.FilePath, &m.FeatureNames, &m.IsActive, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *Repository) CountLabeledSessions(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ar.session_id)
		FROM analysis_results ar
		JOIN grade_decisions gd ON gd.session_id = ar.session_id
		WHERE gd.override_reason IS NOT NULL
		  AND ar.analysis_type = 'heuristic'
	`).Scan(&count)
	return count, err
}
