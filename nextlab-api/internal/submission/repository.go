package submission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

// ─── Validation queries ───────────────────────────────────────────────────────

// StudentEnrolledInActividad checks that the student has an active inscripcion
// in the curso that owns the given actividad.
func (r *Repository) StudentEnrolledInActividad(ctx context.Context, studentID, actividadID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM inscripciones i
		JOIN cursos c ON c.id = i.curso_id
		JOIN actividades a ON a.curso_id = c.id
		WHERE a.id = $1
		  AND i.user_id = $2
		  AND i.is_active = true
		  AND c.is_active = true`,
		actividadID, studentID).Scan(&count)
	return count > 0, err
}

// GetActividadInfo fetches the fields needed to process a submission.
func (r *Repository) GetActividadInfo(ctx context.Context, actividadID uuid.UUID) (*ActividadInfo, error) {
	a := &ActividadInfo{}
	err := r.db.QueryRow(ctx, `
		SELECT a.id, a.tipo, a.deadline, a.late_submission_allowed, a.has_submissions, c.is_active
		FROM actividades a
		JOIN cursos c ON c.id = a.curso_id
		WHERE a.id = $1`,
		actividadID).Scan(
		&a.ID, &a.Tipo, &a.Deadline, &a.LateSubmissionAllowed, &a.HasSubmissions, &a.CursoIsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

// GetSubmissionByStudent returns the active submission for a student+actividad, or nil.
func (r *Repository) GetSubmissionByStudent(ctx context.Context, studentID, actividadID uuid.UUID) (*Submission, error) {
	s := &Submission{}
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, actividad_id, storage_key, original_filename,
		       file_size_bytes, mime_type, submitted_at, is_late,
		       edit_window_expires_at, last_replaced_at, created_at
		FROM submissions
		WHERE student_id = $1 AND actividad_id = $2`,
		studentID, actividadID).Scan(
		&s.ID, &s.StudentID, &s.ActividadID, &s.StorageKey, &s.OriginalFilename,
		&s.FileSizeBytes, &s.MimeType, &s.SubmittedAt, &s.IsLate,
		&s.EditWindowExpiresAt, &s.LastReplacedAt, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// GetSubmissionByID returns a submission by its primary key.
func (r *Repository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*Submission, error) {
	s := &Submission{}
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, actividad_id, storage_key, original_filename,
		       file_size_bytes, mime_type, submitted_at, is_late,
		       edit_window_expires_at, last_replaced_at, created_at
		FROM submissions WHERE id = $1`,
		id).Scan(
		&s.ID, &s.StudentID, &s.ActividadID, &s.StorageKey, &s.OriginalFilename,
		&s.FileSizeBytes, &s.MimeType, &s.SubmittedAt, &s.IsLate,
		&s.EditWindowExpiresAt, &s.LastReplacedAt, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ─── Write queries ────────────────────────────────────────────────────────────

// InsertSubmission inserts a new submission and marks has_submissions on the activity.
// Both writes happen in a single transaction.
func (r *Repository) InsertSubmission(ctx context.Context, s *Submission) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO submissions
		  (id, student_id, actividad_id, storage_key, original_filename,
		   file_size_bytes, mime_type, submitted_at, is_late, edit_window_expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.ID, s.StudentID, s.ActividadID, s.StorageKey, s.OriginalFilename,
		s.FileSizeBytes, s.MimeType, s.SubmittedAt, s.IsLate, s.EditWindowExpiresAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE actividades SET has_submissions = true WHERE id = $1 AND has_submissions = false`,
		s.ActividadID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateSubmission replaces the file metadata on an existing submission in a transaction.
func (r *Repository) UpdateSubmission(ctx context.Context, studentID, actividadID uuid.UUID,
	newKey, filename, mimeType string, size int64, submittedAt time.Time, isLate bool) error {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE submissions SET
		  storage_key      = $3,
		  original_filename= $4,
		  file_size_bytes  = $5,
		  mime_type        = $6,
		  submitted_at     = $7,
		  is_late          = $8,
		  last_replaced_at = NOW()
		WHERE student_id = $1 AND actividad_id = $2`,
		studentID, actividadID, newKey, filename, size, mimeType, submittedAt, isLate,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("submission not found")
	}
	return tx.Commit(ctx)
}

// DeleteSubmission removes the submission row and returns the storage key for MinIO cleanup.
func (r *Repository) DeleteSubmission(ctx context.Context, studentID, actividadID uuid.UUID) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `
		DELETE FROM submissions
		WHERE student_id = $1 AND actividad_id = $2
		RETURNING storage_key`,
		studentID, actividadID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return key, err
}

// ─── Docente / co-docente list queries ───────────────────────────────────────

// VerifyDocenteOwnsActividad returns true if the docente owns the curso
// that contains the given actividad.
func (r *Repository) VerifyDocenteOwnsActividad(ctx context.Context, actividadID, cursoID, docenteID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM actividades a
		JOIN cursos c ON c.id = a.curso_id
		WHERE a.id = $1 AND c.id = $2 AND c.docente_id = $3`,
		actividadID, cursoID, docenteID).Scan(&count)
	return count > 0, err
}

// VerifyCodocenteGrantForActividad returns true if the co_docente holds an active grant.
func (r *Repository) VerifyCodocenteGrantForActividad(ctx context.Context, actividadID, codocenteID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM codocente_grants
		WHERE actividad_id = $1 AND codocente_id = $2
		  AND is_active = true AND revoked_at IS NULL`,
		actividadID, codocenteID).Scan(&count)
	return count > 0, err
}

// GetSubmissionsByActividad returns all submissions for an activity.
// Access must already be verified by the caller.
func (r *Repository) GetSubmissionsByActividad(ctx context.Context, actividadID uuid.UUID) ([]SubmissionListItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id, u.id, u.name, u.email,
		       s.original_filename, s.file_size_bytes, s.mime_type,
		       s.submitted_at, s.is_late, s.edit_window_expires_at, s.last_replaced_at
		FROM submissions s
		JOIN users u ON u.id = s.student_id
		WHERE s.actividad_id = $1
		ORDER BY u.name ASC`,
		actividadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SubmissionListItem
	for rows.Next() {
		var item SubmissionListItem
		var expiresAt *time.Time
		err := rows.Scan(
			&item.ID,
			&item.Student.ID, &item.Student.Name, &item.Student.Email,
			&item.OriginalFilename, &item.FileSizeBytes, &item.MimeType,
			&item.SubmittedAt, &item.IsLate, &expiresAt, &item.LastReplacedAt,
		)
		if err != nil {
			return nil, err
		}
		item.EditWindowExpiresAt = expiresAt
		item.IsLocked = expiresAt == nil || time.Now().After(*expiresAt)
		list = append(list, item)
	}
	return list, rows.Err()
}

// VerifySubmissionBelongsToDocente checks that a submission's activity is in one of the docente's courses.
func (r *Repository) VerifySubmissionBelongsToDocente(ctx context.Context, submissionID, docenteID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions s
		JOIN actividades a ON a.id = s.actividad_id
		JOIN cursos c ON c.id = a.curso_id
		WHERE s.id = $1 AND c.docente_id = $2`,
		submissionID, docenteID).Scan(&count)
	return count > 0, err
}

// VerifySubmissionBelongsToCodocente checks the co_docente has an active grant for the activity.
func (r *Repository) VerifySubmissionBelongsToCodocente(ctx context.Context, submissionID, codocenteID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions s
		JOIN codocente_grants g ON g.actividad_id = s.actividad_id
		WHERE s.id = $1 AND g.codocente_id = $2
		  AND g.is_active = true AND g.revoked_at IS NULL`,
		submissionID, codocenteID).Scan(&count)
	return count > 0, err
}

// ─── Config change log ────────────────────────────────────────────────────────

// InsertConfigChange writes an audit log entry for an activity config change.
// Must be called within a transaction (tx is *pgxpool.Pool-compatible via pgx.Tx).
func (r *Repository) InsertConfigChange(ctx context.Context, tx pgx.Tx, actividadID, changedBy uuid.UUID, field, prev, next string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO actividad_config_changes
		  (actividad_id, changed_by, field_name, previous_value, new_value)
		VALUES ($1, $2, $3, $4, $5)`,
		actividadID, changedBy, field, prev, next,
	)
	return err
}

// UpdateLateSubmissionAllowed updates the flag and writes a config log entry atomically.
func (r *Repository) UpdateLateSubmissionAllowed(ctx context.Context, actividadID, docenteID uuid.UUID, newVal bool) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var prevVal bool
	err = tx.QueryRow(ctx, `
		SELECT a.late_submission_allowed FROM actividades a
		JOIN cursos c ON c.id = a.curso_id
		WHERE a.id = $1 AND c.docente_id = $2`,
		actividadID, docenteID).Scan(&prevVal)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("actividad not found or access denied")
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE actividades SET late_submission_allowed = $1, updated_at = NOW() WHERE id = $2`,
		newVal, actividadID,
	)
	if err != nil {
		return err
	}

	prev := "false"
	if prevVal {
		prev = "true"
	}
	next := "false"
	if newVal {
		next = "true"
	}
	if err := r.InsertConfigChange(ctx, tx, actividadID, docenteID, "late_submission_allowed", prev, next); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetConfigChanges returns the audit log for an activity, accessible by the owning docente or admin.
func (r *Repository) GetConfigChanges(ctx context.Context, actividadID, cursoID, requesterID uuid.UUID, role string) ([]ConfigChangeResponse, error) {
	if role == "admin" {
		return r.queryConfigChanges(ctx, actividadID)
	}
	// docente must own the curso
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM actividades a
		JOIN cursos c ON c.id = a.curso_id
		WHERE a.id = $1 AND c.id = $2 AND c.docente_id = $3`,
		actividadID, cursoID, requesterID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("forbidden")
	}
	return r.queryConfigChanges(ctx, actividadID)
}

func (r *Repository) queryConfigChanges(ctx context.Context, actividadID uuid.UUID) ([]ConfigChangeResponse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cc.id, u.id, u.name, cc.field_name, cc.previous_value, cc.new_value, cc.changed_at
		FROM actividad_config_changes cc
		JOIN users u ON u.id = cc.changed_by
		WHERE cc.actividad_id = $1
		ORDER BY cc.changed_at DESC`,
		actividadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ConfigChangeResponse
	for rows.Next() {
		var c ConfigChangeResponse
		if err := rows.Scan(
			&c.ID, &c.ChangedBy.ID, &c.ChangedBy.Name,
			&c.FieldName, &c.PreviousValue, &c.NewValue, &c.ChangedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
