package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Elder2003/nextlab-api/internal/format"
	tracing "github.com/Elder2003/nextlab-api/internal/tracing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DetectionRepository struct {
	db *pgxpool.Pool
}

func NewDetectionRepository(db *pgxpool.Pool) *DetectionRepository {
	return &DetectionRepository{db: db}
}

func (r *DetectionRepository) InsertHeuristic(ctx context.Context, sessionID uuid.UUID, score int, flagged []string, eventCount int, temporalJSON []byte) (uuid.UUID, error) {
	var id uuid.UUID
	var temporal interface{}
	if len(temporalJSON) > 0 {
		temporal = temporalJSON
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO analysis_results
			(session_id, analysis_type, score, flagged_categories, event_count, temporal_metrics)
		VALUES ($1, 'heuristic', $2, $3, $4, $5)
		RETURNING id`,
		sessionID, score, flagged, eventCount, temporal,
	).Scan(&id)
	return id, err
}

func (r *DetectionRepository) InsertLLMResult(ctx context.Context, sessionID uuid.UUID, advisoryText, suggestedRange, backendUsed, llmStatus string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO analysis_results
			(session_id, analysis_type, advisory_text, suggested_grade_range, backend_used, llm_status)
		VALUES ($1, 'llm', $2, $3, $4, $5)
		RETURNING id`,
		sessionID, advisoryText, suggestedRange, backendUsed, llmStatus,
	).Scan(&id)
	return id, err
}

func (r *DetectionRepository) HeuristicExists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM analysis_results WHERE session_id=$1 AND analysis_type='heuristic'`,
		sessionID,
	).Scan(&count)
	return count > 0, err
}

func (r *DetectionRepository) LLMExists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM analysis_results WHERE session_id=$1 AND analysis_type='llm'`,
		sessionID,
	).Scan(&count)
	return count > 0, err
}

func (r *DetectionRepository) GetAnalysisResults(ctx context.Context, sessionID uuid.UUID) (*HeuristicResult, *LLMResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, analysis_type, created_at,
		       score, flagged_categories, event_count,
		       advisory_text, suggested_grade_range, backend_used, llm_status
		FROM analysis_results WHERE session_id=$1 ORDER BY analysis_type`,
		sessionID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var heuristic *HeuristicResult
	var llm *LLMResult

	for rows.Next() {
		var rowID uuid.UUID
		var aType string
		var createdAt time.Time
		var score *int
		var flagged []string
		var eventCount *int
		var advisoryText, suggestedRange, backendUsed, llmStatus *string

		if err := rows.Scan(&rowID, &aType, &createdAt, &score, &flagged, &eventCount,
			&advisoryText, &suggestedRange, &backendUsed, &llmStatus); err != nil {
			return nil, nil, err
		}

		if aType == "heuristic" && score != nil {
			heuristic = &HeuristicResult{
				SessionID:         sessionID,
				Score:             *score,
				FlaggedCategories: flagged,
				EventCount:        func() int { if eventCount != nil { return *eventCount }; return 0 }(),
				CreatedAt:         createdAt,
			}
		} else if aType == "llm" {
			llm = &LLMResult{
				ID:                  rowID,
				SessionID:           sessionID,
				AdvisoryText:        strDeref(advisoryText),
				SuggestedGradeRange: strDeref(suggestedRange),
				BackendUsed:         strDeref(backendUsed),
				LLMStatus:           strDeref(llmStatus),
				CreatedAt:           createdAt,
			}
		}
	}
	return heuristic, llm, rows.Err()
}

func (r *DetectionRepository) GetGradeDecision(ctx context.Context, sessionID uuid.UUID) (*GradeDecisionResult, error) {
	var g GradeDecisionResult
	err := r.db.QueryRow(ctx, `
		SELECT final_grade, suggested_grade_range, override_reason, decided_by, decided_at
		FROM grade_decisions WHERE session_id=$1`, sessionID,
	).Scan(&g.FinalGrade, &g.SuggestedGradeRange, &g.OverrideReason, &g.DecidedBy, &g.DecidedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

func (r *DetectionRepository) GradeDecisionExists(ctx context.Context, sessionID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM grade_decisions WHERE session_id=$1`, sessionID,
	).Scan(&count)
	return count > 0, err
}

func (r *DetectionRepository) InsertGradeDecision(ctx context.Context, sessionID, analysisResultID, decidedBy uuid.UUID, finalGrade float64, suggestedRange, overrideReason string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO grade_decisions
			(session_id, analysis_result_id, final_grade, suggested_grade_range, override_reason, decided_by)
		VALUES ($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''))`,
		sessionID, analysisResultID, finalGrade, suggestedRange, overrideReason, overrideReason,
	)
	return err
}

func (r *DetectionRepository) GetSessionForDetection(ctx context.Context, sessionID uuid.UUID) (actividadID, submissionID, studentID uuid.UUID, err error) {
	var subID *uuid.UUID
	err = r.db.QueryRow(ctx,
		`SELECT actividad_id, submission_id, student_id FROM exam_sessions WHERE id=$1`,
		sessionID,
	).Scan(&actividadID, &subID, &studentID)
	if err == pgx.ErrNoRows {
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, fmt.Errorf("session_not_found")
	}
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, err
	}
	if subID != nil {
		submissionID = *subID
	}
	return actividadID, submissionID, studentID, nil
}

func (r *DetectionRepository) GetSubmissionStorageKey(ctx context.Context, submissionID uuid.UUID) (storageKey, originalFilename string, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT storage_key, original_filename FROM submissions WHERE id=$1`,
		submissionID,
	).Scan(&storageKey, &originalFilename)
	return storageKey, originalFilename, err
}

func (r *DetectionRepository) GetCurricularScope(ctx context.Context, actividadID uuid.UUID) (carrera, codigoMateria string, semestre int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT c.carrera, c.codigo_materia, c.semestre
		FROM actividades a JOIN cursos c ON c.id = a.curso_id
		WHERE a.id=$1`, actividadID,
	).Scan(&carrera, &codigoMateria, &semestre)
	return carrera, codigoMateria, semestre, err
}

func (r *DetectionRepository) GetDocenteScope(ctx context.Context, actividadID uuid.UUID) (docenteID, cursoID uuid.UUID, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT c.docente_id, c.id
		FROM actividades a JOIN cursos c ON c.id = a.curso_id
		WHERE a.id=$1`, actividadID,
	).Scan(&docenteID, &cursoID)
	return docenteID, cursoID, err
}

func (r *DetectionRepository) GetTraceEvents(ctx context.Context, sessionID uuid.UUID) ([]tracing.TraceEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, sequence_num, event_type, received_at, metadata, event_hash
		FROM trace_events WHERE session_id=$1 ORDER BY sequence_num ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []tracing.TraceEvent
	for rows.Next() {
		var e tracing.TraceEvent
		if err := rows.Scan(&e.ID, &e.SessionID, &e.SequenceNum, &e.EventType,
			&e.ReceivedAt, &e.Metadata, &e.EventHash); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *DetectionRepository) VerifyDocenteOwnsSession(ctx context.Context, sessionID, docenteID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM exam_sessions es
		JOIN actividades a ON a.id = es.actividad_id
		JOIN cursos c ON c.id = a.curso_id
		WHERE es.id=$1 AND c.docente_id=$2`,
		sessionID, docenteID,
	).Scan(&count)
	return count > 0, err
}

// GetFormatReport fetches the format report for a session.
// It first looks up the submission_id from the session, then queries format_reports.
// Returns nil when the report is not yet available (pending).
func (r *DetectionRepository) GetFormatReport(ctx context.Context, sessionID uuid.UUID, submissionID uuid.UUID) (*format.FormatReport, error) {
	var query string
	var arg any
	if submissionID == uuid.Nil {
		// No submission — look for no_submission record keyed by session_id
		query = `SELECT id, submission_id, session_id, parse_status, fail_reason, file_type,
		                page_count, word_count, char_count, doc_metadata, compliant, rule_results, created_at
		         FROM format_reports WHERE session_id = $1 AND submission_id IS NULL LIMIT 1`
		arg = sessionID
	} else {
		query = `SELECT id, submission_id, session_id, parse_status, fail_reason, file_type,
		                page_count, word_count, char_count, doc_metadata, compliant, rule_results, created_at
		         FROM format_reports WHERE submission_id = $1`
		arg = submissionID
	}

	var rep format.FormatReport
	var subID *uuid.UUID
	var failReason, fileType *string
	var metaJSON, rulesJSON []byte

	err := r.db.QueryRow(ctx, query, arg).Scan(
		&rep.ID, &subID, &rep.SessionID, &rep.ParseStatus, &failReason, &fileType,
		&rep.PageCount, &rep.WordCount, &rep.CharCount, &metaJSON, &rep.Compliant, &rulesJSON, &rep.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rep.SubmissionID = subID
	if failReason != nil {
		rep.FailReason = *failReason
	}
	if fileType != nil {
		rep.FileType = *fileType
	}
	if metaJSON != nil {
		json.Unmarshal(metaJSON, &rep.DocMetadata) //nolint:errcheck
	}
	if rulesJSON != nil {
		json.Unmarshal(rulesJSON, &rep.RuleResults) //nolint:errcheck
	}
	if rep.RuleResults == nil {
		rep.RuleResults = []format.RuleResult{}
	}
	return &rep, nil
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
