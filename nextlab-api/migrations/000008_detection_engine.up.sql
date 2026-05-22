-- M8 Detection Engine migration

-- Analysis results: append-only, two rows max per session (heuristic + llm).
-- Never UPDATE or DELETE after INSERT (Constitution Principle I).
CREATE TABLE IF NOT EXISTS analysis_results (
  id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id            UUID        NOT NULL REFERENCES exam_sessions(id),
  analysis_type         TEXT        NOT NULL CHECK (analysis_type IN ('heuristic', 'llm')),
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- heuristic-specific columns (null on llm rows)
  score                 INT,
  flagged_categories    TEXT[],
  event_count           INT,
  -- llm-specific columns (null on heuristic rows)
  advisory_text         TEXT,
  suggested_grade_range TEXT,
  backend_used          TEXT,
  llm_status            TEXT        CHECK (llm_status IN ('success', 'failed')),
  UNIQUE (session_id, analysis_type)
);

CREATE INDEX IF NOT EXISTS idx_analysis_results_session
  ON analysis_results(session_id);

-- Grade decisions: one per session, append-only (Constitution Principle I).
-- override_reason enforced at application layer when grade outside suggested range.
CREATE TABLE IF NOT EXISTS grade_decisions (
  id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id            UUID        NOT NULL REFERENCES exam_sessions(id),
  analysis_result_id    UUID        NOT NULL REFERENCES analysis_results(id),
  final_grade           NUMERIC(5,2) NOT NULL,
  suggested_grade_range TEXT,
  override_reason       TEXT,
  decided_by            UUID        NOT NULL REFERENCES users(id),
  decided_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (session_id)
);
