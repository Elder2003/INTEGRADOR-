package alerts

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type AlertType string

const (
	AlertSubmissionReceived AlertType = "submission_received"
	AlertLateSubmission     AlertType = "late_submission"
	AlertFormatViolation    AlertType = "format_violation"
	AlertIntegrityFlag      AlertType = "integrity_flag"
	AlertGitAnalysisDone    AlertType = "git_analysis_done"
)

type Alert struct {
	ID          int64
	DocenteID   uuid.UUID
	ActividadID *uuid.UUID
	StudentID   *uuid.UUID
	Type        AlertType
	Payload     map[string]any
	Advisory    bool
	IsRead      bool
	CreatedAt   time.Time
}

// AlertWriter is a narrow interface injected into submission handlers and async workers
// to avoid importing the full alerts package from those packages.
type AlertWriter interface {
	WriteAlert(ctx context.Context, a Alert) error
}

var (
	ErrAlertNotFound = errors.New("alert not found")
	ErrForbidden     = errors.New("forbidden")
)
