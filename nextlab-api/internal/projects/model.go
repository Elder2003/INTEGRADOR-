package projects

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not_found")
	ErrConflict            = errors.New("conflict")
	ErrStageHasSubmissions = errors.New("stage_has_submissions")
	ErrNoStages            = errors.New("no_stages")
	ErrWrongTipo           = errors.New("wrong_activity_type")
)

// ─── Investigacion ────────────────────────────────────────────────────────────

type InvestigacionStage struct {
	ID           uuid.UUID `json:"id"`
	ActividadID  uuid.UUID `json:"actividad_id"`
	Name         string    `json:"name"`
	DisplayOrder int       `json:"display_order"`
	Deadline     time.Time `json:"deadline"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type StageSubmission struct {
	ID               uuid.UUID  `json:"id"`
	StageID          uuid.UUID  `json:"stage_id"`
	StudentID        uuid.UUID  `json:"student_id"`
	FileKey          string     `json:"-"`
	OriginalFilename string     `json:"original_filename"`
	IsLate           bool       `json:"is_late"`
	Grade            *float64   `json:"grade"`
	GradeNote        string     `json:"grade_note"`
	GradedAt         *time.Time `json:"graded_at,omitempty"`
	GradedBy         *uuid.UUID `json:"graded_by,omitempty"`
	SubmittedAt      time.Time  `json:"submitted_at"`
}

// StageStatus is the computed status for a student's stage (no DB column).
type StageStatus string

const (
	StageStatusPending       StageStatus = "pending"
	StageStatusSubmitted     StageStatus = "submitted"
	StageStatusSubmittedLate StageStatus = "submitted_late"
)

type StageProgressEntry struct {
	StageID     uuid.UUID   `json:"stage_id"`
	Status      StageStatus `json:"status"`
	SubmittedAt *time.Time  `json:"submitted_at"`
	IsLate      bool        `json:"is_late"`
	Grade       *float64    `json:"grade"`
}

type StudentProgress struct {
	StudentID   uuid.UUID            `json:"student_id"`
	StudentName string               `json:"student_name"`
	Stages      []StageProgressEntry `json:"stages"`
}

type StageProgressResponse struct {
	ActividadID uuid.UUID            `json:"actividad_id"`
	Stages      []InvestigacionStage `json:"stages"`
	Students    []StudentProgress    `json:"students"`
}

// ─── Git Analysis ─────────────────────────────────────────────────────────────

type GitSubmission struct {
	ID          uuid.UUID `json:"id"`
	ActividadID uuid.UUID `json:"actividad_id"`
	StudentID   uuid.UUID `json:"student_id"`
	RepoURL     string    `json:"repo_url"`
	SubmittedAt time.Time `json:"submitted_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GitAnalysis struct {
	ID               uuid.UUID  `json:"id"`
	GitSubmissionID  uuid.UUID  `json:"git_submission_id"`
	Status           string     `json:"status"`
	CommitCount      *int       `json:"commit_count"`
	Authors          []string   `json:"authors"`
	FirstCommitAt    *time.Time `json:"first_commit_at"`
	LastCommitAt     *time.Time `json:"last_commit_at"`
	DeadlineRushPct  *float64   `json:"deadline_rush_pct"`
	ConsistencyScore *int       `json:"consistency_score"`
	Flags            []string   `json:"flags"`
	LLMAdvisory      string     `json:"llm_advisory"`
	LLMStatus        string     `json:"llm_status"`
	FailReason       string     `json:"fail_reason,omitempty"`
	Grade            *float64   `json:"grade"`
	OverrideReason   string     `json:"override_reason,omitempty"`
	GradedBy         *uuid.UUID `json:"graded_by,omitempty"`
	GradedAt         *time.Time `json:"graded_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type GitAnalysisJob struct {
	ID              uuid.UUID  `json:"id"`
	GitAnalysisID   uuid.UUID  `json:"git_analysis_id"`
	Status          string     `json:"status"`
	Retries         int        `json:"retries"`
	QueuedAt        time.Time  `json:"queued_at"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	// joined from git_analyses + git_submissions for worker use
	RepoURL         string     `json:"-"`
	ActividadID     uuid.UUID  `json:"-"`
}

type GitReportResponse struct {
	GitSubmissionID uuid.UUID    `json:"git_submission_id"`
	StudentID       uuid.UUID    `json:"student_id"`
	RepoURL         string       `json:"repo_url"`
	Analysis        *GitAnalysis `json:"analysis"`
	AnalysisStatus  string       `json:"analysis_status"`
}

// ─── Oral Defense ─────────────────────────────────────────────────────────────

type DefenseAppointment struct {
	ID          uuid.UUID  `json:"id"`
	ActividadID uuid.UUID  `json:"actividad_id"`
	StudentID   uuid.UUID  `json:"student_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	Grade       *float64   `json:"grade"`
	GradedBy    *uuid.UUID `json:"graded_by,omitempty"`
	GradedAt    *time.Time `json:"graded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
