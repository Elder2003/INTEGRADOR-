# Data Model: Appeals & Final Audit (M13)

## Migration 000013_appeals_final_audit.up.sql

```sql
-- Appeals: one per exam session; immutable after insert
CREATE TABLE appeals (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID        NOT NULL REFERENCES exam_sessions(id),
    filed_by    UUID        NOT NULL REFERENCES users(id),
    type        TEXT        NOT NULL CHECK (type IN ('analysis_dispute', 'grade_dispute')),
    status      TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'escalated', 'resolved', 'admin_resolved')),
    reason      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id)
);

CREATE INDEX appeals_filed_by_idx ON appeals(filed_by);
CREATE INDEX appeals_status_idx   ON appeals(status);

-- Appeal decisions: append-only log of review outcomes
CREATE TABLE appeal_decisions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id     UUID        NOT NULL REFERENCES appeals(id),
    decided_by    UUID        NOT NULL REFERENCES users(id),
    role          TEXT        NOT NULL CHECK (role IN ('docente', 'admin')),
    decision      TEXT        NOT NULL CHECK (decision IN ('upheld', 'rejected', 'overridden')),
    justification TEXT        NOT NULL,
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX appeal_decisions_appeal_idx ON appeal_decisions(appeal_id);

-- Audit events: hash-chained evidence chain per appeal
CREATE TABLE audit_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    appeal_id   UUID        NOT NULL REFERENCES appeals(id),
    sequence    INT         NOT NULL,
    event_type  TEXT        NOT NULL,
    payload     JSONB       NOT NULL DEFAULT '{}',
    prev_hash   TEXT,
    event_hash  TEXT        NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (appeal_id, sequence)
);

CREATE INDEX audit_events_appeal_idx ON audit_events(appeal_id, sequence);
```

## Migration 000013_appeals_final_audit.down.sql

```sql
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS appeal_decisions;
DROP TABLE IF EXISTS appeals;
```

## Go Structs — internal/appeals

```go
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
    ID           uuid.UUID `json:"id"`
    AppealID     uuid.UUID `json:"appeal_id"`
    DecidedBy    uuid.UUID `json:"decided_by"`
    Role         string    `json:"role"`
    Decision     string    `json:"decision"`
    Justification string   `json:"justification"`
    DecidedAt    time.Time `json:"decided_at"`
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
    Events   []AuditEvent `json:"events"`
    Valid    bool         `json:"valid"`
}
```

## Grade Log Struct — internal/detection (addition)

```go
type GradeLogResult struct {
    SessionID           uuid.UUID  `json:"session_id"`
    FinalGrade          float64    `json:"final_grade"`
    SuggestedGradeRange string     `json:"suggested_grade_range"`
    OverrideReason      *string    `json:"override_reason,omitempty"`
    DecidedBy           uuid.UUID  `json:"decided_by"`
    DecidedAt           time.Time  `json:"decided_at"`
}
```

## Relationships

- `appeals.session_id` → `exam_sessions(id)` — one session can have at most one appeal (UNIQUE)
- `appeals.filed_by` → `users(id)` — student who filed
- `appeal_decisions.appeal_id` → `appeals(id)` — multiple decisions can exist (escalation then admin)
- `audit_events.appeal_id` → `appeals(id)` — ordered evidence chain per appeal
- M13 reads from (but does not own): `exam_sessions`, `submissions`, `analysis_results`, `grade_decisions`, `trace_events`

## AuditEvent Hash Computation

```
canonical = occurred_at_iso8601 + "|" + event_type + "|" + strconv.Itoa(sequence) + "|" + payload_json + "|" + prev_hash_or_empty
event_hash = hex(sha256([]byte(canonical)))
```

Event types and their payload fields:

| event_type | payload fields |
|---|---|
| `submission_received` | `session_id`, `student_id`, `actividad_id`, `submitted_at`, `original_filename` |
| `analysis_completed` | `session_id`, `heuristic_score`, `suggested_grade_range`, `llm_status` |
| `appeal_filed` | `appeal_id`, `type`, `filed_by`, `reason` |
| `decision_recorded` | `appeal_id`, `decided_by`, `role`, `decision`, `justification` |
