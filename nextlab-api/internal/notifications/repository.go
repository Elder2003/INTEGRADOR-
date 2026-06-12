package notifications

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Insert(ctx context.Context, studentID uuid.UUID, notifType string, payload json.RawMessage) (*StudentNotification, error) {
	var n StudentNotification
	err := r.db.QueryRow(ctx, `
		INSERT INTO student_notifications (student_id, type, payload)
		VALUES ($1, $2, $3)
		RETURNING id, student_id, type, payload, read_at, created_at
	`, studentID, notifType, payload).Scan(
		&n.ID, &n.StudentID, &n.Type, &n.Payload, &n.ReadAt, &n.CreatedAt,
	)
	return &n, err
}

func (r *Repository) GetPaginated(ctx context.Context, studentID uuid.UUID, cursor *uuid.UUID, limit int) ([]StudentNotification, bool, *uuid.UUID, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows []StudentNotification
	var q string
	var args []any

	if cursor == nil {
		q = `SELECT id, student_id, type, payload, read_at, created_at
		     FROM student_notifications
		     WHERE student_id = $1
		     ORDER BY created_at DESC, id DESC
		     LIMIT $2`
		args = []any{studentID, limit + 1}
	} else {
		q = `SELECT id, student_id, type, payload, read_at, created_at
		     FROM student_notifications
		     WHERE student_id = $1
		       AND (created_at, id) < (
		           SELECT created_at, id FROM student_notifications WHERE id = $2
		       )
		     ORDER BY created_at DESC, id DESC
		     LIMIT $3`
		args = []any{studentID, *cursor, limit + 1}
	}

	dbRows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, false, nil, err
	}
	defer dbRows.Close()

	for dbRows.Next() {
		var n StudentNotification
		if err := dbRows.Scan(&n.ID, &n.StudentID, &n.Type, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, false, nil, err
		}
		rows = append(rows, n)
	}
	if err := dbRows.Err(); err != nil {
		return nil, false, nil, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	var nextCursor *uuid.UUID
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		nextCursor = &last
	}

	return rows, hasMore, nextCursor, nil
}

func (r *Repository) MarkRead(ctx context.Context, id uuid.UUID, studentID uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE student_notifications
		SET read_at = NOW()
		WHERE id = $1 AND student_id = $2 AND read_at IS NULL
	`, id, studentID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) CountUnread(ctx context.Context, studentID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_notifications
		WHERE student_id = $1 AND read_at IS NULL
	`, studentID).Scan(&count)
	return count, err
}
