# Data Model: M9 — Format Validation

## New Tables

### `format_reports`

Stores one format validation result per submitted file. Append-only (Principle I).

```sql
CREATE TABLE IF NOT EXISTS format_reports (
  id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id  UUID         NOT NULL REFERENCES submissions(id),
  session_id     UUID         NOT NULL REFERENCES exam_sessions(id),
  parse_status   TEXT         NOT NULL CHECK (parse_status IN ('success', 'failed')),
  fail_reason    TEXT,          -- 'password_protected' | 'unsupported_file_type' |
                                -- 'empty_file' | 'corrupt_file' | 'no_submission'
  file_type      TEXT,          -- 'pdf' | 'docx' | 'unsupported' | NULL on failure
  page_count     INT,
  word_count     INT,
  char_count     INT,
  doc_metadata   JSONB,         -- {author, created_at, modified_at, title}
  compliant      BOOLEAN        NOT NULL DEFAULT true,
  rule_results   JSONB,         -- [{rule, status, actual, required}]
  created_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
  UNIQUE (submission_id)
);

CREATE INDEX IF NOT EXISTS idx_format_reports_session
  ON format_reports(session_id);
```

**Notes**:
- `UNIQUE(submission_id)` enforces idempotency — one report per submission file.
- `fail_reason` is NULL when `parse_status = 'success'`.
- `compliant = false` when any rule fails OR when `parse_status = 'failed'`.
- `rule_results` JSONB array element shape: `{"rule": "min_pages", "status": "fail", "actual": 2, "required": 3}`.
- `doc_metadata` JSONB shape: `{"author": "...", "created_at": "...", "title": "..."}` — all fields optional.

---

## Modified Tables

### `actividades` (ALTER TABLE)

Add a nullable JSONB column for docente-configured format rules.

```sql
ALTER TABLE actividades
  ADD COLUMN IF NOT EXISTS format_rules JSONB;
```

**`format_rules` JSONB shape**:
```json
{
  "min_pages": 2,
  "max_pages": 10,
  "min_word_count": 500,
  "required_types": ["pdf"]
}
```
All fields are optional. NULL column means no rules configured → `compliant = true` on any submission.

---

## Go Types

### `internal/format/model.go`

```go
type FormatReport struct {
    ID           uuid.UUID       `json:"id"`
    SubmissionID uuid.UUID       `json:"submission_id"`
    SessionID    uuid.UUID       `json:"session_id"`
    ParseStatus  string          `json:"parse_status"`  // "success" | "failed"
    FailReason   string          `json:"fail_reason,omitempty"`
    FileType     string          `json:"file_type,omitempty"`
    PageCount    int             `json:"page_count"`
    WordCount    int             `json:"word_count"`
    CharCount    int             `json:"char_count"`
    DocMetadata  map[string]any  `json:"doc_metadata,omitempty"`
    Compliant    bool            `json:"compliant"`
    RuleResults  []RuleResult    `json:"rule_results"`
    CreatedAt    time.Time       `json:"created_at"`
}

type RuleResult struct {
    Rule     string `json:"rule"`      // "min_pages" | "max_pages" | "min_word_count" | "file_type"
    Status   string `json:"status"`    // "pass" | "fail"
    Actual   any    `json:"actual"`    // int or string depending on rule
    Required any    `json:"required"`
}

type FormatRulesConfig struct {
    MinPages      *int     `json:"min_pages,omitempty"`
    MaxPages      *int     `json:"max_pages,omitempty"`
    MinWordCount  *int     `json:"min_word_count,omitempty"`
    RequiredTypes []string `json:"required_types,omitempty"`
}
```

---

## Relationships

```
exam_sessions (id) ──── format_reports (session_id)
submissions   (id) ──1─ format_reports (submission_id)   [UNIQUE]
actividades   (id) ──── exam_sessions  (actividad_id)
actividades.format_rules (JSONB) → FormatRulesConfig
```

---

## Migration Files

| File | Purpose |
|------|---------|
| `migrations/000009_format_validation.up.sql` | CREATE `format_reports`; ALTER `actividades` ADD `format_rules` |
| `migrations/000009_format_validation.down.sql` | DROP `format_reports`; ALTER `actividades` DROP `format_rules` |
