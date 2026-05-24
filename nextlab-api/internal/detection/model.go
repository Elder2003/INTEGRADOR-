package detection

import (
	"errors"
	"time"

	"github.com/Elder2003/nextlab-api/internal/format"
	"github.com/google/uuid"
)

var (
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not_found")
	ErrConflict         = errors.New("conflict")
	ErrOverrideRequired = errors.New("override_reason_required")
)

type HeuristicResult struct {
	SessionID         uuid.UUID `json:"session_id"`
	Score             int       `json:"score"`
	FlaggedCategories []string  `json:"flagged_categories"`
	EventCount        int       `json:"event_count"`
	CreatedAt         time.Time `json:"created_at"`
}

type LLMResult struct {
	ID                   uuid.UUID `json:"id"`
	SessionID            uuid.UUID `json:"session_id"`
	AdvisoryText         string    `json:"advisory_text"`
	SuggestedGradeRange  string    `json:"suggested_grade_range"`
	BackendUsed          string    `json:"backend_used"`
	LLMStatus            string    `json:"llm_status"`
	CreatedAt            time.Time `json:"created_at"`
}

type DetectionReport struct {
	SessionID     uuid.UUID            `json:"session_id"`
	StudentID     uuid.UUID            `json:"student_id"`
	ActividadID   uuid.UUID            `json:"actividad_id"`
	FormatReport  *format.FormatReport `json:"format_report"`
	Heuristic     *HeuristicResult     `json:"heuristic"`
	LLM           *LLMResult           `json:"llm"`
	LLMStatus     string               `json:"llm_status"`
	GradeDecision *GradeDecisionResult `json:"grade_decision"`
}

type GradeDecisionResult struct {
	FinalGrade           float64   `json:"final_grade"`
	SuggestedGradeRange  string    `json:"suggested_grade_range,omitempty"`
	OverrideReason       string    `json:"override_reason,omitempty"`
	DecidedBy            uuid.UUID `json:"decided_by"`
	DecidedAt            time.Time `json:"decided_at"`
}

type GradeRequest struct {
	FinalGrade     float64 `json:"final_grade"`
	OverrideReason string  `json:"override_reason,omitempty"`
}

type GradeLogResult struct {
	SessionID           uuid.UUID  `json:"session_id"`
	FinalGrade          float64    `json:"final_grade"`
	SuggestedGradeRange string     `json:"suggested_grade_range"`
	OverrideReason      *string    `json:"override_reason,omitempty"`
	DecidedBy           uuid.UUID  `json:"decided_by"`
	DecidedAt           time.Time  `json:"decided_at"`
}
