CREATE TABLE IF NOT EXISTS format_reports (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID        REFERENCES submissions(id),
  session_id    UUID        NOT NULL REFERENCES exam_sessions(id),
  parse_status  TEXT        NOT NULL CHECK (parse_status IN ('success', 'failed')),
  fail_reason   TEXT,
  file_type     TEXT,
  page_count    INT,
  word_count    INT,
  char_count    INT,
  doc_metadata  JSONB,
  compliant     BOOLEAN     NOT NULL DEFAULT true,
  rule_results  JSONB,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (submission_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_format_reports_no_submission
  ON format_reports (session_id) WHERE submission_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_format_reports_session
  ON format_reports (session_id);

ALTER TABLE actividades
  ADD COLUMN IF NOT EXISTS format_rules JSONB;
