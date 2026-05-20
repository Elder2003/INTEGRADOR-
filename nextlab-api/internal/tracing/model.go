package tracing

import (
	"time"

	"github.com/google/uuid"
)

// ─── DB entities ─────────────────────────────────────────────────────────────

type ExamSession struct {
	ID               uuid.UUID  `db:"id"`
	StudentID        uuid.UUID  `db:"student_id"`
	ActividadID      uuid.UUID  `db:"actividad_id"`
	SubmissionID     *uuid.UUID `db:"submission_id"`
	Status           string     `db:"status"`
	StartedAt        time.Time  `db:"started_at"`
	ClosedAt         *time.Time `db:"closed_at"`
	LastHeartbeatAt  *time.Time `db:"last_heartbeat_at"`
	LastEventHash    *string    `db:"last_event_hash"`
	EventCount       int        `db:"event_count"`
}

type TraceEvent struct {
	ID          uuid.UUID      `db:"id"`
	SessionID   uuid.UUID      `db:"session_id"`
	SequenceNum int            `db:"sequence_num"`
	EventType   string         `db:"event_type"`
	ReceivedAt  time.Time      `db:"received_at"`
	Metadata    map[string]any `db:"metadata"`
	EventHash   string         `db:"event_hash"`
}

type WorkerJob struct {
	ID           uuid.UUID  `db:"id"`
	SessionID    uuid.UUID  `db:"session_id"`
	SubmissionID uuid.UUID  `db:"submission_id"`
	StreamID     *string    `db:"stream_id"`
	Status       string     `db:"status"`
	EnqueuedAt   time.Time  `db:"enqueued_at"`
	ClaimedAt    *time.Time `db:"claimed_at"`
	DoneAt       *time.Time `db:"done_at"`
	AttemptCount int        `db:"attempt_count"`
	Error        *string    `db:"error"`
}

// ─── API response types ───────────────────────────────────────────────────────

type ExamSessionResponse struct {
	ID          string `json:"id"`
	ActividadID string `json:"actividad_id"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	WSURL       string `json:"ws_url"`
}

type TraceEventItem struct {
	SequenceNum int            `json:"sequence_num"`
	EventType   string         `json:"event_type"`
	ReceivedAt  time.Time      `json:"received_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	EventHash   string         `json:"event_hash"`
}

type TraceResponse struct {
	SessionID   string           `json:"session_id"`
	StudentID   string           `json:"student_id"`
	ActividadID string           `json:"actividad_id"`
	Status      string           `json:"status"`
	StartedAt   time.Time        `json:"started_at"`
	ClosedAt    *time.Time       `json:"closed_at,omitempty"`
	EventCount  int              `json:"event_count"`
	Events      []TraceEventItem `json:"events"`
}

type VerifyResponse struct {
	Valid                bool `json:"valid"`
	EventCount           int  `json:"event_count"`
	FirstInvalidSequence *int `json:"first_invalid_sequence,omitempty"`
}

// ─── WebSocket message types ──────────────────────────────────────────────────

type ClientEvent struct {
	Type     string         `json:"type"`
	ClientTS *int64         `json:"client_ts,omitempty"`
	Extra    map[string]any `json:"-"`
}

type TraceAck struct {
	Type        string    `json:"type"`
	SequenceNum int       `json:"sequence_num"`
	ReceivedAt  time.Time `json:"received_at"`
}

type WSError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type WSSessionClosed struct {
	Type     string    `json:"type"`
	Reason   string    `json:"reason"`
	ClosedAt time.Time `json:"closed_at"`
}

type WSConnected struct {
	Type       string `json:"type"`
	SessionID  string `json:"session_id"`
	EventCount int    `json:"event_count"`
}
