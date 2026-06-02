package tracing

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// ─── Activity + enrollment checks ────────────────────────────────────────────

// GetActividadTraceInfo checks enrollment and whether trace is enabled for the activity.
func (r *Repository) GetActividadTraceInfo(ctx context.Context, actividadID, studentID uuid.UUID) (traceEnabled bool, enrolled bool, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT
			a.trace_enabled,
			EXISTS(
				SELECT 1 FROM inscripciones i
				JOIN cursos c ON c.id = i.curso_id
				WHERE c.id = a.curso_id
				  AND i.user_id = $2
				  AND i.is_active = true
			) AS enrolled
		FROM actividades a
		WHERE a.id = $1
	`, actividadID, studentID).Scan(&traceEnabled, &enrolled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	return traceEnabled, enrolled, err
}

// ─── Session CRUD ─────────────────────────────────────────────────────────────

// InsertSessionIdempotent inserts a new active session; does nothing if one already exists.
func (r *Repository) InsertSessionIdempotent(ctx context.Context, studentID, actividadID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO exam_sessions (student_id, actividad_id, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (student_id, actividad_id) DO NOTHING
	`, studentID, actividadID)
	return err
}

// GetSessionByStudentActivity returns the session for this student+activity, or nil if not found.
func (r *Repository) GetSessionByStudentActivity(ctx context.Context, studentID, actividadID uuid.UUID) (*ExamSession, error) {
	var s ExamSession
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, actividad_id, submission_id, status,
		       started_at, closed_at, last_heartbeat_at, last_event_hash, event_count
		FROM exam_sessions
		WHERE student_id = $1 AND actividad_id = $2
	`, studentID, actividadID).Scan(
		&s.ID, &s.StudentID, &s.ActividadID, &s.SubmissionID, &s.Status,
		&s.StartedAt, &s.ClosedAt, &s.LastHeartbeatAt, &s.LastEventHash, &s.EventCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetSessionByID returns a session by its primary key, or nil if not found.
func (r *Repository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (*ExamSession, error) {
	var s ExamSession
	err := r.db.QueryRow(ctx, `
		SELECT id, student_id, actividad_id, submission_id, status,
		       started_at, closed_at, last_heartbeat_at, last_event_hash, event_count
		FROM exam_sessions
		WHERE id = $1
	`, sessionID).Scan(
		&s.ID, &s.StudentID, &s.ActividadID, &s.SubmissionID, &s.Status,
		&s.StartedAt, &s.ClosedAt, &s.LastHeartbeatAt, &s.LastEventHash, &s.EventCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CloseSession transitions a session to a terminal status.
func (r *Repository) CloseSession(ctx context.Context, sessionID uuid.UUID, submissionID *uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE exam_sessions
		SET status = $2, submission_id = $3, closed_at = now()
		WHERE id = $1 AND status = 'active'
	`, sessionID, status, submissionID)
	return err
}

// ─── AppendEvent (atomic hash chain + sequence increment) ─────────────────────

// AppendEventResult carries the assigned sequence number and server timestamp.
type AppendEventResult struct {
	SequenceNum int
	ReceivedAt  time.Time
}

// AppendEvent inserts a trace event atomically, computing the hash inside the TX.
// computeHash receives (newSeq, prevHash, receivedAt) and returns the event hash.
// Returns ErrSessionClosed if the session is no longer active.
func (r *Repository) AppendEvent(ctx context.Context, sessionID uuid.UUID, eventType string, metadataJSON []byte, computeHash func(seq int, prevHash string, receivedAt time.Time) string) (*AppendEventResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	var eventCount int
	var lastEventHash *string
	err = tx.QueryRow(ctx, `
		SELECT status, event_count, last_event_hash FROM exam_sessions WHERE id = $1 FOR UPDATE
	`, sessionID).Scan(&status, &eventCount, &lastEventHash)
	if err != nil {
		return nil, err
	}
	if status != "active" {
		return nil, ErrSessionClosed
	}

	newSeq := eventCount + 1
	receivedAt := time.Now().UTC()

	var prevHash string
	if eventCount == 0 || lastEventHash == nil {
		prevHash = fmt.Sprintf("%x", sha256.Sum256([]byte(sessionID.String())))
	} else {
		prevHash = *lastEventHash
	}

	eventHash := computeHash(newSeq, prevHash, receivedAt)

	_, err = tx.Exec(ctx, `
		INSERT INTO trace_events (session_id, sequence_num, event_type, received_at, metadata, event_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sessionID, newSeq, eventType, receivedAt, metadataJSON, eventHash)
	if err != nil {
		return nil, err
	}

	heartbeatUpdate := ""
	if eventType == "heartbeat" {
		heartbeatUpdate = ", last_heartbeat_at = $4"
	}
	args := []any{sessionID, newSeq, eventHash}
	if eventType == "heartbeat" {
		args = append(args, receivedAt)
	}
	_, err = tx.Exec(ctx, `
		UPDATE exam_sessions
		SET event_count = $2, last_event_hash = $3`+heartbeatUpdate+`
		WHERE id = $1
	`, args...)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &AppendEventResult{SequenceNum: newSeq, ReceivedAt: receivedAt}, nil
}

// ─── Worker jobs ──────────────────────────────────────────────────────────────

// InsertWorkerJob creates a new pending job and returns its ID.
func (r *Repository) InsertWorkerJob(ctx context.Context, sessionID, submissionID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO worker_jobs (session_id, submission_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
	`, sessionID, submissionID).Scan(&id)
	return id, err
}

// SetJobStreamID updates a job row with the Redis stream message ID and marks it queued.
func (r *Repository) SetJobStreamID(ctx context.Context, jobID uuid.UUID, streamID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE worker_jobs SET stream_id = $2, status = 'queued' WHERE id = $1
	`, jobID, streamID)
	return err
}

// MarkJobDead permanently kills a worker job that has exhausted its retries.
func (r *Repository) MarkJobDead(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE worker_jobs SET status='dead', dead_at=NOW(), error=$2 WHERE id=$1
	`, jobID, errMsg)
	return err
}

// MarkJobStatus updates a job's status and optional timestamps.
func (r *Repository) MarkJobStatus(ctx context.Context, jobID uuid.UUID, status string, claimedAt *time.Time, doneAt *time.Time, errMsg *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE worker_jobs
		SET status = $2,
		    claimed_at    = COALESCE($3, claimed_at),
		    done_at       = COALESCE($4, done_at),
		    error         = COALESCE($5, error),
		    attempt_count = attempt_count + CASE WHEN $2 = 'processing' THEN 1 ELSE 0 END
		WHERE id = $1
	`, jobID, status, claimedAt, doneAt, errMsg)
	return err
}

// GetOrphanedJobs returns jobs stuck in pending/queued status for > 60 seconds.
func (r *Repository) GetOrphanedJobs(ctx context.Context) ([]WorkerJob, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, submission_id, stream_id, status, enqueued_at,
		       claimed_at, done_at, attempt_count, error, max_retries, dead_at
		FROM worker_jobs
		WHERE status IN ('pending', 'queued')
		  AND status != 'dead'
		  AND enqueued_at < NOW() - INTERVAL '60 seconds'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkerJobs(rows)
}

// ─── Deadline + timeout sweeps ────────────────────────────────────────────────

// GetDeadlinedSessions returns active sessions whose activity deadline has passed.
func (r *Repository) GetDeadlinedSessions(ctx context.Context) ([]ExamSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT es.id, es.student_id, es.actividad_id, es.submission_id, es.status,
		       es.started_at, es.closed_at, es.last_heartbeat_at, es.last_event_hash, es.event_count
		FROM exam_sessions es
		JOIN actividades a ON a.id = es.actividad_id
		WHERE es.status = 'active'
		  AND a.deadline < NOW()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// GetTimedOutSessions returns active sessions with no heartbeat for > 5 minutes, before deadline.
func (r *Repository) GetTimedOutSessions(ctx context.Context) ([]ExamSession, error) {
	rows, err := r.db.Query(ctx, `
		SELECT es.id, es.student_id, es.actividad_id, es.submission_id, es.status,
		       es.started_at, es.closed_at, es.last_heartbeat_at, es.last_event_hash, es.event_count
		FROM exam_sessions es
		JOIN actividades a ON a.id = es.actividad_id
		WHERE es.status = 'active'
		  AND a.deadline > NOW()
		  AND es.last_heartbeat_at < NOW() - INTERVAL '5 minutes'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// VerifyDocenteOwnsSession checks that the docente is owner of the curso containing the session's actividad.
func (r *Repository) VerifyDocenteOwnsSession(ctx context.Context, sessionID, docenteID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM exam_sessions es
			JOIN actividades a ON a.id = es.actividad_id
			JOIN cursos c ON c.id = a.curso_id
			WHERE es.id = $1 AND c.docente_id = $2
		)
	`, sessionID, docenteID).Scan(&exists)
	return exists, err
}

// ─── Trace retrieval (docente) ────────────────────────────────────────────────

// GetTraceBySubmission fetches the session and all trace events for a submission, role-scoped.
func (r *Repository) GetTraceBySubmission(ctx context.Context, submissionID, callerID uuid.UUID, role string) (*ExamSession, []TraceEvent, error) {
	var scopeJoin string
	switch role {
	case "docente":
		scopeJoin = `JOIN cursos c ON c.id = a.curso_id AND c.docente_id = $2`
	case "co_docente":
		scopeJoin = `JOIN codocente_grants g ON g.actividad_id = a.id AND g.codocente_id = $2 AND g.is_active = true AND g.revoked_at IS NULL`
	default:
		return nil, nil, ErrForbidden
	}

	var s ExamSession
	err := r.db.QueryRow(ctx, `
		SELECT es.id, es.student_id, es.actividad_id, es.submission_id, es.status,
		       es.started_at, es.closed_at, es.last_heartbeat_at, es.last_event_hash, es.event_count
		FROM exam_sessions es
		JOIN submissions sub ON sub.id = $1 AND sub.actividad_id = es.actividad_id AND sub.student_id = es.student_id
		JOIN actividades a ON a.id = es.actividad_id
		`+scopeJoin+`
		WHERE sub.id = $1
	`, submissionID, callerID).Scan(
		&s.ID, &s.StudentID, &s.ActividadID, &s.SubmissionID, &s.Status,
		&s.StartedAt, &s.ClosedAt, &s.LastHeartbeatAt, &s.LastEventHash, &s.EventCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrForbidden
	}
	if err != nil {
		return nil, nil, err
	}

	events, err := r.GetAllEvents(ctx, s.ID)
	return &s, events, err
}

// GetEventsPaginated returns a page of trace events after the given sequence number.
// Uses limit+1 trick to determine hasMore without an extra COUNT query.
func (r *Repository) GetEventsPaginated(ctx context.Context, sessionID uuid.UUID, afterSeq, limit int) ([]TraceEvent, bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, sequence_num, event_type, received_at, metadata, event_hash
		FROM trace_events
		WHERE session_id = $1 AND sequence_num > $2
		ORDER BY sequence_num ASC
		LIMIT $3
	`, sessionID, afterSeq, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var events []TraceEvent
	for rows.Next() {
		var e TraceEvent
		var metaBytes []byte
		if err := rows.Scan(&e.ID, &e.SessionID, &e.SequenceNum, &e.EventType, &e.ReceivedAt, &metaBytes, &e.EventHash); err != nil {
			return nil, false, err
		}
		if len(metaBytes) > 0 {
			_ = jsonUnmarshal(metaBytes, &e.Metadata)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}
	return events, hasMore, nil
}

// GetAllEvents returns all trace events for a session ordered by sequence number.
func (r *Repository) GetAllEvents(ctx context.Context, sessionID uuid.UUID) ([]TraceEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, sequence_num, event_type, received_at, metadata, event_hash
		FROM trace_events
		WHERE session_id = $1
		ORDER BY sequence_num ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TraceEvent
	for rows.Next() {
		var e TraceEvent
		var metaBytes []byte
		if err := rows.Scan(&e.ID, &e.SessionID, &e.SequenceNum, &e.EventType, &e.ReceivedAt, &metaBytes, &e.EventHash); err != nil {
			return nil, err
		}
		if len(metaBytes) > 0 {
			_ = jsonUnmarshal(metaBytes, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

func scanSessions(rows pgx.Rows) ([]ExamSession, error) {
	var sessions []ExamSession
	for rows.Next() {
		var s ExamSession
		if err := rows.Scan(
			&s.ID, &s.StudentID, &s.ActividadID, &s.SubmissionID, &s.Status,
			&s.StartedAt, &s.ClosedAt, &s.LastHeartbeatAt, &s.LastEventHash, &s.EventCount,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// ─── Offline batch insert (M17) ───────────────────────────────────────────────

type offlineBatchEvent struct {
	Type            string
	ClientTimestamp time.Time
	Metadata        map[string]any
}

// InsertBatch stores offline-buffered trace events with received_offline=true.
// Events that already exist (same session_id + client_timestamp + event_type where received_offline)
// are silently skipped. Returns the number of accepted and skipped events.
func (r *Repository) InsertBatch(ctx context.Context, sessionID uuid.UUID, events []offlineBatchEvent) (accepted, skipped int, err error) {
	for _, e := range events {
		metaJSON, _ := jsonMarshal(e.Metadata)
		tag, execErr := r.db.Exec(ctx, `
			INSERT INTO trace_events
				(session_id, sequence_num, event_type, received_at, client_timestamp, received_offline, metadata, event_hash)
			SELECT
				$1,
				COALESCE((SELECT MAX(sequence_num) FROM trace_events WHERE session_id = $1), 0) + 1,
				$2,
				NOW(),
				$3,
				TRUE,
				$4,
				encode(sha256(($1::text || $2 || $3::text)::bytea), 'hex')
			WHERE NOT EXISTS (
				SELECT 1 FROM trace_events
				WHERE session_id = $1
				  AND event_type = $2
				  AND client_timestamp = $3
				  AND received_offline = TRUE
			)
		`, sessionID, e.Type, e.ClientTimestamp, metaJSON)
		if execErr != nil {
			return accepted, skipped, execErr
		}
		if tag.RowsAffected() > 0 {
			accepted++
		} else {
			skipped++
		}
	}
	return accepted, skipped, nil
}

func scanWorkerJobs(rows pgx.Rows) ([]WorkerJob, error) {
	var jobs []WorkerJob
	for rows.Next() {
		var j WorkerJob
		if err := rows.Scan(
			&j.ID, &j.SessionID, &j.SubmissionID, &j.StreamID, &j.Status,
			&j.EnqueuedAt, &j.ClaimedAt, &j.DoneAt, &j.AttemptCount, &j.Error,
			&j.MaxRetries, &j.DeadAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}
