package alerts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewRepository(db *pgxpool.Pool, rdb *redis.Client) *Repository {
	return &Repository{db: db, rdb: rdb}
}

// InsertAlert persists an alert and returns its assigned BIGSERIAL id.
func (r *Repository) InsertAlert(ctx context.Context, a Alert) (int64, error) {
	payload, err := json.Marshal(a.Payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}
	var id int64
	err = r.db.QueryRow(ctx,
		`INSERT INTO alerts (docente_id, actividad_id, student_id, type, payload, advisory)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		a.DocenteID, a.ActividadID, a.StudentID, string(a.Type), payload, a.Advisory,
	).Scan(&id)
	return id, err
}

// GetAlertsSince returns alerts for a docente with id > sinceID, optionally filtered to unread only.
func (r *Repository) GetAlertsSince(ctx context.Context, docenteID uuid.UUID, sinceID int64, unreadOnly bool, limit int) ([]Alert, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, docente_id, actividad_id, student_id, type, payload, advisory, is_read, created_at
	          FROM alerts
	          WHERE docente_id = $1 AND id > $2`
	args := []any{docenteID, sinceID}
	if unreadOnly {
		query += " AND is_read = false"
	}
	query += " ORDER BY id ASC LIMIT $3"
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Alert
	for rows.Next() {
		var a Alert
		var rawPayload []byte
		if err := rows.Scan(&a.ID, &a.DocenteID, &a.ActividadID, &a.StudentID,
			&a.Type, &rawPayload, &a.Advisory, &a.IsRead, &a.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawPayload, &a.Payload); err != nil {
			a.Payload = map[string]any{}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// MarkAlertRead sets is_read=true for a single alert owned by docenteID.
func (r *Repository) MarkAlertRead(ctx context.Context, alertID int64, docenteID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE alerts SET is_read = true WHERE id = $1 AND docente_id = $2`,
		alertID, docenteID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlertNotFound
	}
	return nil
}

// GetDocenteForActividad returns the owning docente_id for a given actividad.
func (r *Repository) GetDocenteForActividad(ctx context.Context, actividadID uuid.UUID) (uuid.UUID, error) {
	var docenteID uuid.UUID
	err := r.db.QueryRow(ctx,
		`SELECT docente_id FROM actividades WHERE id = $1`, actividadID,
	).Scan(&docenteID)
	return docenteID, err
}
