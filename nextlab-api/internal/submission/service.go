package submission

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"

	"github.com/Elder2003/nextlab-api/internal/alerts"
)

// TracingService is implemented by *tracing.Service; defined here to avoid import cycles.
type TracingService interface {
	CloseSessionWithSubmission(ctx context.Context, studentID, actividadID, submissionID uuid.UUID)
}


const maxFileSizeBytes = 52_428_800 // 50 MB

var (
	ErrSubmissionExists        = errors.New("submission_exists")
	ErrLateSubmissionsDisabled = errors.New("late_submissions_disabled")
	ErrInvalidActivityType     = errors.New("invalid_activity_type")
	ErrFileTooLarge            = errors.New("file_too_large")
	ErrSubmissionNotFound      = errors.New("submission_not_found")
	ErrSubmissionLocked        = errors.New("submission_locked")
	ErrStorageUnavailable      = errors.New("storage_unavailable")
	ErrActivityNotFound        = errors.New("activity_not_found")
	ErrNotEnrolled             = errors.New("not_enrolled")
	ErrForbidden               = errors.New("forbidden")
)

var validTipos = map[string]bool{
	"practica": true, "tarea": true, "informe": true, "examen": true, "defensa_oral": true,
}

// Service handles all submission business logic.
type Service struct {
	repo        *Repository
	mc          *minio.Client
	bucket      string
	tracingSvc  TracingService
	alertWriter alerts.AlertWriter
}

// StreamResult carries an open MinIO object reader plus metadata for download handlers.
type StreamResult struct {
	Object           io.ReadCloser
	Size             int64
	ContentType      string
	OriginalFilename string
}

// ─── Student operations ───────────────────────────────────────────────────────

// UploadFile validates, stores in MinIO, and records the submission atomically.
func (s *Service) UploadFile(ctx context.Context, studentID, actividadID uuid.UUID, header *multipart.FileHeader) (*SubmissionResponse, error) {
	enrolled, err := s.repo.StudentEnrolledInActividad(ctx, studentID, actividadID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrNotEnrolled
	}

	act, err := s.repo.GetActividadInfo(ctx, actividadID)
	if err != nil {
		return nil, err
	}
	if act == nil || !act.CursoIsActive {
		return nil, ErrActivityNotFound
	}
	if !validTipos[act.Tipo] {
		return nil, ErrInvalidActivityType
	}

	existing, err := s.repo.GetSubmissionByStudent(ctx, studentID, actividadID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSubmissionExists
	}

	if header.Size > maxFileSizeBytes {
		return nil, ErrFileTooLarge
	}

	now := time.Now().UTC()
	isLate := now.After(act.Deadline)

	if isLate && !act.LateSubmissionAllowed {
		return nil, ErrLateSubmissionsDisabled
	}

	var editWindowExpiresAt *time.Time
	if !isLate {
		window := now.Add(30 * time.Minute)
		if !act.LateSubmissionAllowed && window.After(act.Deadline) {
			window = act.Deadline
		}
		editWindowExpiresAt = &window
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	ext := filepath.Ext(header.Filename)
	storageKey := fmt.Sprintf("submissions/%s/%s/%s%s", actividadID, studentID, uuid.New(), ext)

	f, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = s.mc.PutObject(ctx, s.bucket, storageKey, f, header.Size, minio.PutObjectOptions{
		ContentType:          mimeType,
		ServerSideEncryption: encrypt.NewSSE(),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	sub := &Submission{
		ID:                  uuid.New(),
		StudentID:           studentID,
		ActividadID:         actividadID,
		StorageKey:          storageKey,
		OriginalFilename:    header.Filename,
		FileSizeBytes:       header.Size,
		MimeType:            mimeType,
		SubmittedAt:         now,
		IsLate:              isLate,
		EditWindowExpiresAt: editWindowExpiresAt,
	}
	if err := s.repo.InsertSubmission(ctx, sub); err != nil {
		_ = s.mc.RemoveObject(ctx, s.bucket, storageKey, minio.RemoveObjectOptions{})
		if isPgUniqueViolation(err) {
			return nil, ErrSubmissionExists
		}
		return nil, err
	}

	if act.TraceEnabled && s.tracingSvc != nil {
		s.tracingSvc.CloseSessionWithSubmission(ctx, studentID, actividadID, sub.ID)
	}

	if s.alertWriter != nil {
		aid := actividadID
		sid := studentID
		_ = s.alertWriter.WriteAlert(ctx, alerts.Alert{
			DocenteID:   act.DocenteID,
			ActividadID: &aid,
			StudentID:   &sid,
			Type:        alerts.AlertSubmissionReceived,
			Payload:     map[string]any{"submission_id": sub.ID, "filename": header.Filename, "submitted_at": now},
		})
	}

	resp := toResponse(sub)
	return &resp, nil
}

// ReplaceFile replaces the stored file within the edit window.
func (s *Service) ReplaceFile(ctx context.Context, studentID, actividadID uuid.UUID, header *multipart.FileHeader) (*SubmissionResponse, error) {
	existing, err := s.repo.GetSubmissionByStudent(ctx, studentID, actividadID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrSubmissionNotFound
	}
	if existing.IsLocked() {
		return nil, ErrSubmissionLocked
	}
	if header.Size > maxFileSizeBytes {
		return nil, ErrFileTooLarge
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	ext := filepath.Ext(header.Filename)
	newKey := fmt.Sprintf("submissions/%s/%s/%s%s", actividadID, studentID, uuid.New(), ext)

	f, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = s.mc.PutObject(ctx, s.bucket, newKey, f, header.Size, minio.PutObjectOptions{
		ContentType:          mimeType,
		ServerSideEncryption: encrypt.NewSSE(),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	now := time.Now().UTC()

	act, err := s.repo.GetActividadInfo(ctx, actividadID)
	if err != nil {
		_ = s.mc.RemoveObject(ctx, s.bucket, newKey, minio.RemoveObjectOptions{})
		return nil, err
	}
	isLate := act != nil && now.After(act.Deadline)

	oldKey := existing.StorageKey
	if err := s.repo.UpdateSubmission(ctx, studentID, actividadID, newKey, header.Filename, mimeType, header.Size, now, isLate); err != nil {
		_ = s.mc.RemoveObject(ctx, s.bucket, newKey, minio.RemoveObjectOptions{})
		return nil, err
	}

	if act != nil && act.TraceEnabled && s.tracingSvc != nil {
		s.tracingSvc.CloseSessionWithSubmission(ctx, studentID, actividadID, existing.ID)
	}

	_ = s.mc.RemoveObject(ctx, s.bucket, oldKey, minio.RemoveObjectOptions{})

	updated := &Submission{
		ID:                  existing.ID,
		StudentID:           studentID,
		ActividadID:         actividadID,
		StorageKey:          newKey,
		OriginalFilename:    header.Filename,
		FileSizeBytes:       header.Size,
		MimeType:            mimeType,
		SubmittedAt:         now,
		IsLate:              isLate,
		EditWindowExpiresAt: existing.EditWindowExpiresAt,
	}
	resp := toResponse(updated)
	return &resp, nil
}

// DeleteOwnSubmission removes the submission if within the edit window.
func (s *Service) DeleteOwnSubmission(ctx context.Context, studentID, actividadID uuid.UUID) error {
	existing, err := s.repo.GetSubmissionByStudent(ctx, studentID, actividadID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrSubmissionNotFound
	}
	if existing.IsLocked() {
		return ErrSubmissionLocked
	}

	oldKey := existing.StorageKey
	if _, err := s.repo.DeleteSubmission(ctx, studentID, actividadID); err != nil {
		return err
	}

	_ = s.mc.RemoveObject(ctx, s.bucket, oldKey, minio.RemoveObjectOptions{})
	return nil
}

// ─── Shared — file streaming ──────────────────────────────────────────────────

// StreamFile opens the MinIO object for proxied download.
func (s *Service) StreamFile(ctx context.Context, sub *Submission) (*StreamResult, error) {
	obj, err := s.mc.GetObject(ctx, s.bucket, sub.StorageKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil, ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	ct := stat.ContentType
	if ct == "" {
		ct = sub.MimeType
	}

	return &StreamResult{
		Object:           obj,
		Size:             stat.Size,
		ContentType:      ct,
		OriginalFilename: sub.OriginalFilename,
	}, nil
}

// ─── Docente operations ───────────────────────────────────────────────────────

// ListSubmissions returns all submissions for an activity, scoped by caller role.
func (s *Service) ListSubmissions(ctx context.Context, actividadID, cursoID, callerID uuid.UUID, role string) ([]SubmissionListItem, error) {
	switch role {
	case "docente":
		ok, err := s.repo.VerifyDocenteOwnsActividad(ctx, actividadID, cursoID, callerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	case "co_docente":
		ok, err := s.repo.VerifyCodocenteGrantForActividad(ctx, actividadID, callerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	default:
		return nil, ErrForbidden
	}
	return s.repo.GetSubmissionsByActividad(ctx, actividadID)
}

// ResolveSubmissionForDownload fetches a submission by ID scoped to the caller's role.
func (s *Service) ResolveSubmissionForDownload(ctx context.Context, submissionID, callerID uuid.UUID, role string) (*Submission, error) {
	switch role {
	case "docente":
		ok, err := s.repo.VerifySubmissionBelongsToDocente(ctx, submissionID, callerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	case "co_docente":
		ok, err := s.repo.VerifySubmissionBelongsToCodocente(ctx, submissionID, callerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	default:
		return nil, ErrForbidden
	}
	return s.repo.GetSubmissionByID(ctx, submissionID)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
