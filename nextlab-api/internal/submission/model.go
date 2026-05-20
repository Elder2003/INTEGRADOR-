package submission

import (
	"time"

	"github.com/google/uuid"
)

// ─── Domain entities ──────────────────────────────────────────────────────────

type Submission struct {
	ID                   uuid.UUID  `json:"id"`
	StudentID            uuid.UUID  `json:"student_id"`
	ActividadID          uuid.UUID  `json:"actividad_id"`
	StorageKey           string     `json:"-"`
	OriginalFilename     string     `json:"original_filename"`
	FileSizeBytes        int64      `json:"file_size_bytes"`
	MimeType             string     `json:"mime_type"`
	SubmittedAt          time.Time  `json:"submitted_at"`
	IsLate               bool       `json:"is_late"`
	EditWindowExpiresAt  *time.Time `json:"edit_window_expires_at"`
	LastReplacedAt       *time.Time `json:"last_replaced_at"`
	CreatedAt            time.Time  `json:"created_at"`
}

// IsLocked returns true when the submission can no longer be modified by the student.
func (s *Submission) IsLocked() bool {
	if s.EditWindowExpiresAt == nil {
		return true
	}
	return time.Now().After(*s.EditWindowExpiresAt)
}

type ActivityConfigChange struct {
	ID            uuid.UUID `json:"id"`
	ActividadID   uuid.UUID `json:"actividad_id"`
	ChangedBy     uuid.UUID `json:"changed_by"`
	FieldName     string    `json:"field_name"`
	PreviousValue string    `json:"previous_value"`
	NewValue      string    `json:"new_value"`
	ChangedAt     time.Time `json:"changed_at"`
}

// ─── Response types ───────────────────────────────────────────────────────────

type SubmissionResponse struct {
	ID                  uuid.UUID  `json:"id"`
	ActividadID         uuid.UUID  `json:"actividad_id"`
	OriginalFilename    string     `json:"original_filename"`
	FileSizeBytes       int64      `json:"file_size_bytes"`
	MimeType            string     `json:"mime_type"`
	SubmittedAt         time.Time  `json:"submitted_at"`
	IsLate              bool       `json:"is_late"`
	EditWindowExpiresAt *time.Time `json:"edit_window_expires_at"`
	IsLocked            bool       `json:"is_locked"`
	LastReplacedAt      *time.Time `json:"last_replaced_at,omitempty"`
}

func toResponse(s *Submission) SubmissionResponse {
	return SubmissionResponse{
		ID:                  s.ID,
		ActividadID:         s.ActividadID,
		OriginalFilename:    s.OriginalFilename,
		FileSizeBytes:       s.FileSizeBytes,
		MimeType:            s.MimeType,
		SubmittedAt:         s.SubmittedAt,
		IsLate:              s.IsLate,
		EditWindowExpiresAt: s.EditWindowExpiresAt,
		IsLocked:            s.IsLocked(),
		LastReplacedAt:      s.LastReplacedAt,
	}
}

type StudentInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type SubmissionListItem struct {
	ID                  uuid.UUID  `json:"id"`
	Student             StudentInfo `json:"student"`
	OriginalFilename    string     `json:"original_filename"`
	FileSizeBytes       int64      `json:"file_size_bytes"`
	MimeType            string     `json:"mime_type"`
	SubmittedAt         time.Time  `json:"submitted_at"`
	IsLate              bool       `json:"is_late"`
	IsLocked            bool       `json:"is_locked"`
	LastReplacedAt      *time.Time `json:"last_replaced_at,omitempty"`
	EditWindowExpiresAt *time.Time `json:"edit_window_expires_at,omitempty"`
}

type SubmissionListResponse struct {
	ActividadID uuid.UUID            `json:"actividad_id"`
	Submissions []SubmissionListItem `json:"submissions"`
	Total       int                  `json:"total"`
}

type ConfigChangeActor struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ConfigChangeResponse struct {
	ID            uuid.UUID         `json:"id"`
	ChangedBy     ConfigChangeActor `json:"changed_by"`
	FieldName     string            `json:"field_name"`
	PreviousValue string            `json:"previous_value"`
	NewValue      string            `json:"new_value"`
	ChangedAt     time.Time         `json:"changed_at"`
}

type ConfigChangesResponse struct {
	ActividadID uuid.UUID              `json:"actividad_id"`
	Changes     []ConfigChangeResponse `json:"changes"`
}

// ─── Internal types used by the service ──────────────────────────────────────

type ActividadInfo struct {
	ID                   uuid.UUID
	Tipo                 string
	Deadline             time.Time
	LateSubmissionAllowed bool
	HasSubmissions       bool
	CursoIsActive        bool
	TraceEnabled         bool
}

// ─── Errors ───────────────────────────────────────────────────────────────────

type ErrorResponse struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}
