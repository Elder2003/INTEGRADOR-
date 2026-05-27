# Data Model: Alerts & Analytics (M11)

## New Tables

### `alerts`

Append-only event log per docente. One row per system event. Never updated or deleted (Principle I).

```sql
CREATE TABLE alerts (
    id          BIGSERIAL PRIMARY KEY,               -- used as SSE event id (Last-Event-ID)
    docente_id  UUID        NOT NULL REFERENCES users(id),
    actividad_id UUID       REFERENCES actividades(id) ON DELETE SET NULL,
    student_id  UUID        REFERENCES users(id),    -- nullable (course-level events have no student)
    type        TEXT        NOT NULL,                -- enum enforced in application layer
    payload     JSONB       NOT NULL DEFAULT '{}',   -- event-specific metadata
    advisory    BOOLEAN     NOT NULL DEFAULT false,  -- true for AI-generated alerts
    is_read     BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX alerts_docente_id_idx ON alerts (docente_id, id);
CREATE INDEX alerts_actividad_id_idx ON alerts (actividad_id);
```

**Alert types** (enforced as Go constants):

| Type | Trigger | advisory |
|---|---|---|
| `submission_received` | Student submits a file (any tipo) | false |
| `late_submission` | Student submits after stage deadline | false |
| `format_violation` | Format validator marks submission non-compliant | true |
| `integrity_flag` | Detection engine raises a flag | true |
| `git_analysis_done` | Git worker completes analysis | true |

**Payload shape by type**:

```json
// submission_received
{ "submission_id": "uuid", "filename": "string", "submitted_at": "RFC3339" }

// late_submission
{ "stage_id": "uuid", "minutes_late": 42 }

// format_violation
{ "submission_id": "uuid", "violations": ["page_count", "font_size"], "severity": "major" }

// integrity_flag
{ "submission_id": "uuid", "flags": ["copy_paste", "last_minute"], "score": 0.87 }

// git_analysis_done
{ "git_analysis_id": "uuid", "flag_count": 2, "flags": ["deadline_rush"] }
```

---

## Redis Keys (ephemeral — not in migrations)

| Key pattern | Value | TTL | Purpose |
|---|---|---|---|
| `sse_token:<uuid>` | `<docente_id>` (string) | 60 s | Short-lived one-time SSE auth token |
| `alerts:<docente_id>` | pub/sub channel | N/A | Live fan-out channel per docente |

---

## No New Tables for Analytics

Analytics are computed on-demand from existing tables:

| Activity tipo | Grade source table | Submission count source |
|---|---|---|
| `tarea`, `informe`, `examen` | `submissions.grade` | `submissions` |
| `investigacion` | `stage_submissions.grade` | `stage_submissions` |
| `git_analysis` | `git_analyses.grade` | `git_submissions` |
| `defensa_oral` | `defense_appointments.grade` | `defense_appointments` |

Flag count source: `alerts` table filtered by `type IN ('integrity_flag', 'format_violation')` for the actividad.

---

## Go Structs

```go
// internal/alerts/model.go

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

type AlertWriter interface {
    WriteAlert(ctx context.Context, a Alert) error
}

var (
    ErrAlertNotFound = errors.New("alert not found")
    ErrForbidden     = errors.New("forbidden")
)

// internal/analytics/model.go

type GradeDistribution struct {
    Range0_49  int     `json:"range_0_49"`
    Range50_69 int     `json:"range_50_69"`
    Range70_89 int     `json:"range_70_89"`
    Range90_100 int    `json:"range_90_100"`
}

type ActivityAnalytics struct {
    ActividadID       uuid.UUID         `json:"actividad_id"`
    SubmissionCount   int               `json:"submission_count"`
    PendingCount      int               `json:"pending_count"`
    LateCount         int               `json:"late_count"`
    FlaggedCount      int               `json:"flagged_count"`
    GradeDistribution GradeDistribution `json:"grade_distribution"`
    AvgGrade          *float64          `json:"avg_grade"`
}

type CourseAnalytics struct {
    CursoID          uuid.UUID           `json:"curso_id"`
    Activities       []ActivityAnalytics `json:"activities"`
    TotalSubmissions int                 `json:"total_submissions"`
    AvgGrade         *float64            `json:"avg_grade"`
    FlagRate         float64             `json:"flag_rate"`
}
```
