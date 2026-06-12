package appeals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// InsertAppeal opens a transaction, inserts the appeal, seeds 3 audit events, and commits.
func (r *Repository) InsertAppeal(ctx context.Context, req FileAppealRequest, filedBy uuid.UUID) (*Appeal, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var a Appeal
	err = tx.QueryRow(ctx, `
		INSERT INTO appeals (session_id, filed_by, type, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, session_id, filed_by, type, status, reason, created_at`,
		req.SessionID, filedBy, req.Type, req.Reason,
	).Scan(&a.ID, &a.SessionID, &a.FiledBy, &a.Type, &a.Status, &a.Reason, &a.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Gather session context for initial audit events.
	type sessionCtx struct {
		SubmittedAt       string
		OriginalFilename  string
		StudentID         uuid.UUID
		ActividadID       uuid.UUID
		HeuristicScore    *int
		SuggestedRange    *string
		LLMStatus         *string
	}
	var sc sessionCtx
	err = tx.QueryRow(ctx, `
		SELECT
			COALESCE(s.created_at::text, es.started_at::text),
			COALESCE(s.original_filename, ''),
			es.student_id,
			es.actividad_id,
			ar_h.score,
			ar_l.suggested_grade_range,
			ar_l.llm_status
		FROM exam_sessions es
		LEFT JOIN submissions s ON s.id = es.submission_id
		LEFT JOIN analysis_results ar_h ON ar_h.session_id = es.id AND ar_h.analysis_type = 'heuristic'
		LEFT JOIN analysis_results ar_l ON ar_l.session_id = es.id AND ar_l.analysis_type = 'llm'
		WHERE es.id = $1`, req.SessionID,
	).Scan(
		&sc.SubmittedAt, &sc.OriginalFilename, &sc.StudentID, &sc.ActividadID,
		&sc.HeuristicScore, &sc.SuggestedRange, &sc.LLMStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("session context: %w", err)
	}

	type eventDef struct {
		eventType string
		payload   map[string]any
	}
	events := []eventDef{
		{
			eventType: "submission_received",
			payload: map[string]any{
				"session_id":        req.SessionID.String(),
				"student_id":        sc.StudentID.String(),
				"actividad_id":      sc.ActividadID.String(),
				"submitted_at":      sc.SubmittedAt,
				"original_filename": sc.OriginalFilename,
			},
		},
		{
			eventType: "analysis_completed",
			payload: func() map[string]any {
				m := map[string]any{
					"session_id":           req.SessionID.String(),
					"heuristic_score":      0,
					"suggested_grade_range": "",
					"llm_status":           "pending",
				}
				if sc.HeuristicScore != nil {
					m["heuristic_score"] = *sc.HeuristicScore
				}
				if sc.SuggestedRange != nil {
					m["suggested_grade_range"] = *sc.SuggestedRange
				}
				if sc.LLMStatus != nil {
					m["llm_status"] = *sc.LLMStatus
				}
				return m
			}(),
		},
		{
			eventType: "appeal_filed",
			payload: map[string]any{
				"appeal_id": a.ID.String(),
				"type":      req.Type,
				"filed_by":  filedBy.String(),
				"reason":    req.Reason,
			},
		},
	}

	var prevHash string
	for i, ev := range events {
		seq := i + 1
		payloadBytes, _ := json.Marshal(ev.payload)
		payloadStr := string(payloadBytes)
		hash := ComputeHash(a.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), ev.eventType, seq, payloadStr, prevHash)

		var prevHashPtr *string
		if prevHash != "" {
			h := prevHash
			prevHashPtr = &h
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO audit_events (appeal_id, sequence, event_type, payload, prev_hash, event_hash, occurred_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			a.ID, seq, ev.eventType, payloadBytes, prevHashPtr, hash, a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("audit_event seq %d: %w", seq, err)
		}
		prevHash = hash
	}

	return &a, tx.Commit(ctx)
}

func (r *Repository) GetAppeal(ctx context.Context, id uuid.UUID) (*Appeal, error) {
	var a Appeal
	err := r.db.QueryRow(ctx, `
		SELECT id, session_id, filed_by, type, status, reason, created_at
		FROM appeals WHERE id=$1`, id,
	).Scan(&a.ID, &a.SessionID, &a.FiledBy, &a.Type, &a.Status, &a.Reason, &a.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (r *Repository) HasOpenAppeal(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM appeals WHERE session_id=$1 AND status IN ('pending','escalated'))`,
		sessionID,
	).Scan(&exists)
	return exists, err
}

// HasOpenAppealOfType checks if an active appeal of the given type exists for the session.
func (r *Repository) HasOpenAppealOfType(ctx context.Context, sessionID uuid.UUID, appealType string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM appeals WHERE session_id=$1 AND type=$2 AND status IN ('pending','escalated'))`,
		sessionID, appealType,
	).Scan(&exists)
	return exists, err
}

// ListAppeals returns appeals matching filter and the total count.
func (r *Repository) ListAppeals(ctx context.Context, filter AppealFilter) ([]Appeal, int, error) {
	where := []string{"1=1"}
	args := []any{}
	n := 1

	if filter.FiledBy != nil {
		where = append(where, fmt.Sprintf("a.filed_by=$%d", n))
		args = append(args, *filter.FiledBy)
		n++
	}
	if filter.SessionID != nil {
		where = append(where, fmt.Sprintf("a.session_id=$%d", n))
		args = append(args, *filter.SessionID)
		n++
	}
	if filter.ActividadID != nil {
		where = append(where, fmt.Sprintf("es.actividad_id=$%d", n))
		args = append(args, *filter.ActividadID)
		n++
	}
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("a.status=$%d", n))
		args = append(args, filter.Status)
		n++
	}

	needsJoin := filter.ActividadID != nil
	join := ""
	if needsJoin {
		join = "JOIN exam_sessions es ON es.id = a.session_id"
	}

	cond := strings.Join(where, " AND ")

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM appeals a %s WHERE %s`, join, cond)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`
		SELECT a.id, a.session_id, a.filed_by, a.type, a.status, a.reason, a.created_at
		FROM appeals a %s WHERE %s ORDER BY a.created_at DESC`, join, cond)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var appeals []Appeal
	for rows.Next() {
		var ap Appeal
		if err := rows.Scan(&ap.ID, &ap.SessionID, &ap.FiledBy, &ap.Type, &ap.Status, &ap.Reason, &ap.CreatedAt); err != nil {
			return nil, 0, err
		}
		appeals = append(appeals, ap)
	}
	if appeals == nil {
		appeals = []Appeal{}
	}
	return appeals, total, rows.Err()
}

func (r *Repository) UpdateAppealStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE appeals SET status=$1 WHERE id=$2`, status, id)
	return err
}

// InsertDecision inserts a decision and appends a decision_recorded audit event.
func (r *Repository) InsertDecision(ctx context.Context, appealID, decidedBy uuid.UUID, role, decision, justification string) (*AppealDecision, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var d AppealDecision
	err = tx.QueryRow(ctx, `
		INSERT INTO appeal_decisions (appeal_id, decided_by, role, decision, justification)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, appeal_id, decided_by, role, decision, justification, decided_at`,
		appealID, decidedBy, role, decision, justification,
	).Scan(&d.ID, &d.AppealID, &d.DecidedBy, &d.Role, &d.Decision, &d.Justification, &d.DecidedAt)
	if err != nil {
		return nil, err
	}

	// Get last event hash and sequence for this appeal.
	var lastSeq int
	var lastHash string
	err = tx.QueryRow(ctx,
		`SELECT sequence, event_hash FROM audit_events WHERE appeal_id=$1 ORDER BY sequence DESC LIMIT 1`,
		appealID,
	).Scan(&lastSeq, &lastHash)
	if err != nil {
		return nil, fmt.Errorf("last audit event: %w", err)
	}

	seq := lastSeq + 1
	payload := map[string]any{
		"appeal_id":     appealID.String(),
		"decided_by":    decidedBy.String(),
		"role":          role,
		"decision":      decision,
		"justification": justification,
	}
	payloadBytes, _ := json.Marshal(payload)
	hash := ComputeHash(d.DecidedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), "decision_recorded", seq, string(payloadBytes), lastHash)

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (appeal_id, sequence, event_type, payload, prev_hash, event_hash, occurred_at)
		VALUES ($1, $2, 'decision_recorded', $3, $4, $5, $6)`,
		appealID, seq, payloadBytes, lastHash, hash, d.DecidedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("audit_event decision_recorded: %w", err)
	}

	return &d, tx.Commit(ctx)
}

func (r *Repository) GetDecisions(ctx context.Context, appealID uuid.UUID) ([]AppealDecision, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, appeal_id, decided_by, role, decision, justification, decided_at
		FROM appeal_decisions WHERE appeal_id=$1 ORDER BY decided_at ASC`, appealID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []AppealDecision
	for rows.Next() {
		var d AppealDecision
		if err := rows.Scan(&d.ID, &d.AppealID, &d.DecidedBy, &d.Role, &d.Decision, &d.Justification, &d.DecidedAt); err != nil {
			return nil, err
		}
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}

func (r *Repository) GetAuditEvents(ctx context.Context, appealID uuid.UUID) ([]AuditEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, appeal_id, sequence, event_type, payload, prev_hash, event_hash, occurred_at
		FROM audit_events WHERE appeal_id=$1 ORDER BY sequence ASC`, appealID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var payloadBytes []byte
		if err := rows.Scan(&e.ID, &e.AppealID, &e.Sequence, &e.EventType, &payloadBytes, &e.PrevHash, &e.EventHash, &e.OccurredAt); err != nil {
			return nil, err
		}
		if len(payloadBytes) > 0 {
			json.Unmarshal(payloadBytes, &e.Payload) //nolint:errcheck
		}
		if e.Payload == nil {
			e.Payload = map[string]any{}
		}
		events = append(events, e)
	}
	if events == nil {
		events = []AuditEvent{}
	}
	return events, rows.Err()
}

func (r *Repository) VerifyStudentOwnsSession(ctx context.Context, sessionID, studentID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM exam_sessions WHERE id=$1 AND student_id=$2)`,
		sessionID, studentID,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) VerifyDocenteOwnsAppealSession(ctx context.Context, appealID, docenteID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM appeals ap
			JOIN exam_sessions es ON es.id = ap.session_id
			JOIN actividades a ON a.id = es.actividad_id
			JOIN cursos c ON c.id = a.curso_id
			WHERE ap.id=$1 AND c.docente_id=$2
		)`, appealID, docenteID,
	).Scan(&exists)
	return exists, err
}

// VerifyDocenteOwnsActividad checks whether docenteID owns any course for actividadID.
func (r *Repository) VerifyDocenteOwnsActividad(ctx context.Context, actividadID, docenteID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM actividades a
			JOIN cursos c ON c.id = a.curso_id
			WHERE a.id=$1 AND c.docente_id=$2
		)`, actividadID, docenteID,
	).Scan(&exists)
	return exists, err
}
