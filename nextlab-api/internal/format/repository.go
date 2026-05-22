package format

import (
	"context"
	"encoding/json"

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

// InsertFormatReport stores a new format report. Idempotent via ON CONFLICT DO NOTHING on submission_id.
func (r *Repository) InsertFormatReport(ctx context.Context, rep *FormatReport) error {
	rulesJSON, err := json.Marshal(rep.RuleResults)
	if err != nil {
		return err
	}
	var metaJSON []byte
	if rep.DocMetadata != nil {
		metaJSON, err = json.Marshal(rep.DocMetadata)
		if err != nil {
			return err
		}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO format_reports
			(submission_id, session_id, parse_status, fail_reason, file_type,
			 page_count, word_count, char_count, doc_metadata, compliant, rule_results)
		VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), $6, $7, $8, $9, $10, $11)
		ON CONFLICT (submission_id) DO NOTHING`,
		rep.SubmissionID, rep.SessionID, rep.ParseStatus, rep.FailReason, rep.FileType,
		rep.PageCount, rep.WordCount, rep.CharCount, metaJSON, rep.Compliant, rulesJSON,
	)
	return err
}

// InsertNoSubmissionReport stores a format report for a session that had no file uploaded.
// Uses a partial unique index on session_id WHERE submission_id IS NULL for idempotency.
func (r *Repository) InsertNoSubmissionReport(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO format_reports
			(session_id, parse_status, fail_reason, compliant, rule_results)
		VALUES ($1, 'failed', 'no_submission', false, '[]'::jsonb)
		ON CONFLICT DO NOTHING`,
		sessionID,
	)
	return err
}

// GetFormatReportBySubmissionID returns the format report for a given submission, or nil if not found.
func (r *Repository) GetFormatReportBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*FormatReport, error) {
	return r.scanReport(ctx,
		`SELECT id, submission_id, session_id, parse_status, fail_reason, file_type,
		        page_count, word_count, char_count, doc_metadata, compliant, rule_results, created_at
		 FROM format_reports WHERE submission_id = $1`, submissionID)
}

// GetFormatReportBySessionID returns the format report keyed by session (for no_submission records).
func (r *Repository) GetFormatReportBySessionID(ctx context.Context, sessionID uuid.UUID) (*FormatReport, error) {
	return r.scanReport(ctx,
		`SELECT id, submission_id, session_id, parse_status, fail_reason, file_type,
		        page_count, word_count, char_count, doc_metadata, compliant, rule_results, created_at
		 FROM format_reports WHERE session_id = $1 LIMIT 1`, sessionID)
}

// GetFormatRulesForSession fetches the format_rules JSONB from the actividad linked to a session.
// Returns nil, nil when no rules are configured.
func (r *Repository) GetFormatRulesForSession(ctx context.Context, sessionID uuid.UUID) (*FormatRulesConfig, error) {
	var raw []byte
	err := r.db.QueryRow(ctx,
		`SELECT a.format_rules FROM actividades a
		 JOIN exam_sessions es ON es.actividad_id = a.id
		 WHERE es.id = $1`, sessionID,
	).Scan(&raw)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}
	var cfg FormatRulesConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *Repository) scanReport(ctx context.Context, query string, arg any) (*FormatReport, error) {
	var rep FormatReport
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
		rep.RuleResults = []RuleResult{}
	}
	return &rep, nil
}
