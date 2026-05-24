package appeals

import (
	"time"

	"github.com/google/uuid"
)

type Appeal struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	FiledBy   uuid.UUID `json:"filed_by"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type AppealDecision struct {
	ID            uuid.UUID `json:"id"`
	AppealID      uuid.UUID `json:"appeal_id"`
	DecidedBy     uuid.UUID `json:"decided_by"`
	Role          string    `json:"role"`
	Decision      string    `json:"decision"`
	Justification string    `json:"justification"`
	DecidedAt     time.Time `json:"decided_at"`
}

type AuditEvent struct {
	ID         uuid.UUID      `json:"id"`
	AppealID   uuid.UUID      `json:"appeal_id"`
	Sequence   int            `json:"sequence"`
	EventType  string         `json:"event_type"`
	Payload    map[string]any `json:"payload"`
	PrevHash   *string        `json:"prev_hash"`
	EventHash  string         `json:"event_hash"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type FileAppealRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
}

type ReviewRequest struct {
	Decision      string `json:"decision"`
	Justification string `json:"justification"`
}

type EscalateRequest struct {
	Notes string `json:"notes,omitempty"`
}

type EvidenceChain struct {
	AppealID uuid.UUID    `json:"appeal_id"`
	Valid    bool         `json:"valid"`
	Events   []AuditEvent `json:"events"`
}

type AppealFilter struct {
	FiledBy     *uuid.UUID
	SessionID   *uuid.UUID
	ActividadID *uuid.UUID
	Status      string
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
