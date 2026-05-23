package format

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/alerts"
)

type FormatService struct {
	repo        *Repository
	minio       *minio.Client
	bucket      string
	alertWriter alerts.AlertWriter
}

func NewFormatService(repo *Repository, minioClient *minio.Client, bucket string, alertWriter alerts.AlertWriter) *FormatService {
	return &FormatService{repo: repo, minio: minioClient, bucket: bucket, alertWriter: alertWriter}
}

// Run is called by the worker as phase 0 for a closed session.
// It is idempotent: if a format report already exists for the submission, it returns without re-inserting.
func (s *FormatService) Run(ctx context.Context, sessionID uuid.UUID) error {
	// Resolve session → submission_id + actividad_id
	var submissionID *uuid.UUID
	var actividadID uuid.UUID
	var storageKey string
	err := s.repo.db.QueryRow(ctx,
		`SELECT es.submission_id, es.actividad_id, sub.storage_key
		 FROM exam_sessions es
		 LEFT JOIN submissions sub ON sub.id = es.submission_id
		 WHERE es.id = $1`, sessionID,
	).Scan(&submissionID, &actividadID, &storageKey)
	if err != nil {
		return err
	}

	// No file was submitted
	if submissionID == nil {
		if err := s.repo.InsertNoSubmissionReport(ctx, sessionID); err != nil {
			slog.WarnContext(ctx, "format: could not store no_submission report", "session_id", sessionID, "err", err)
		}
		return nil
	}

	// Fetch file bytes from MinIO
	obj, err := s.minio.GetObject(ctx, s.bucket, storageKey, minio.GetObjectOptions{})
	if err != nil {
		slog.WarnContext(ctx, "format: could not fetch file from minio", "storage_key", storageKey, "err", err)
		return nil
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		slog.WarnContext(ctx, "format: could not read file bytes", "err", err)
		return nil
	}

	// Fetch format rules configured for the actividad
	rules, err := s.repo.GetFormatRulesForSession(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "format: could not fetch format rules", "session_id", sessionID, "err", err)
	}

	// Parse the file
	info, parseErr := ParseFile(data)

	rep := &FormatReport{
		SubmissionID: submissionID,
		SessionID:    sessionID,
		RuleResults:  []RuleResult{},
	}

	if parseErr != nil {
		rep.ParseStatus = "failed"
		rep.Compliant = false
		var pe *ParseError
		if errors.As(parseErr, &pe) {
			rep.FailReason = pe.Reason
		} else {
			rep.FailReason = "corrupt_file"
		}
		rep.FileType = info.FileType
	} else {
		ruleResults, compliant := EvaluateRules(info, rules)
		rep.ParseStatus = "success"
		rep.FileType = info.FileType
		rep.PageCount = info.PageCount
		rep.WordCount = info.WordCount
		rep.CharCount = info.CharCount
		rep.DocMetadata = info.DocMetadata
		rep.Compliant = compliant
		rep.RuleResults = ruleResults
	}

	if err := s.repo.InsertFormatReport(ctx, rep); err != nil {
		slog.ErrorContext(ctx, "format: failed to store format report", "session_id", sessionID, "err", err)
		return err
	}

	slog.InfoContext(ctx, "format report stored",
		"session_id", sessionID,
		"parse_status", rep.ParseStatus,
		"compliant", rep.Compliant,
	)

	if !rep.Compliant && s.alertWriter != nil {
		docenteID, _ := s.repo.GetDocenteForActividad(ctx, actividadID)
		aid := actividadID
		_ = s.alertWriter.WriteAlert(ctx, alerts.Alert{
			DocenteID:   docenteID,
			ActividadID: &aid,
			StudentID:   nil,
			Type:        alerts.AlertFormatViolation,
			Advisory:    true,
			Payload:     map[string]any{"submission_id": submissionID, "compliant": false, "parse_status": rep.ParseStatus},
		})
	}

	return nil
}
