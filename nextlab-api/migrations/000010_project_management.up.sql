-- M10: Project Management
-- Three new activity types: investigacion, git_analysis (enum extension);
-- defensa_oral already exists and is extended by defense_appointments.

ALTER TYPE actividad_tipo ADD VALUE IF NOT EXISTS 'investigacion';
ALTER TYPE actividad_tipo ADD VALUE IF NOT EXISTS 'git_analysis';

-- Stages within an investigacion activity
CREATE TABLE investigacion_stages (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id UUID        NOT NULL REFERENCES actividades(id) ON DELETE CASCADE,
  name         TEXT        NOT NULL,
  display_order INTEGER    NOT NULL DEFAULT 0,
  deadline     TIMESTAMPTZ NOT NULL,
  description  TEXT        NOT NULL DEFAULT '',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (actividad_id, name)
);

CREATE INDEX idx_investigacion_stages_actividad ON investigacion_stages(actividad_id, display_order);

-- Per-student file submissions to a specific stage
CREATE TABLE stage_submissions (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  stage_id          UUID        NOT NULL REFERENCES investigacion_stages(id),
  student_id        UUID        NOT NULL REFERENCES users(id),
  file_key          TEXT        NOT NULL,
  original_filename TEXT        NOT NULL,
  is_late           BOOLEAN     NOT NULL DEFAULT false,
  grade             NUMERIC(5,2),
  grade_note        TEXT        NOT NULL DEFAULT '',
  graded_at         TIMESTAMPTZ,
  graded_by         UUID        REFERENCES users(id),
  submitted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (stage_id, student_id)
);

CREATE INDEX idx_stage_submissions_stage   ON stage_submissions(stage_id);
CREATE INDEX idx_stage_submissions_student ON stage_submissions(student_id, stage_id);

-- Student Git URL submissions for git_analysis activities
CREATE TABLE git_submissions (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id UUID        NOT NULL REFERENCES actividades(id),
  student_id   UUID        NOT NULL REFERENCES users(id),
  repo_url     TEXT        NOT NULL,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (actividad_id, student_id)
);

CREATE INDEX idx_git_submissions_actividad ON git_submissions(actividad_id);
CREATE INDEX idx_git_submissions_student   ON git_submissions(student_id, actividad_id);

-- Async analysis records for a git_submission
CREATE TABLE git_analyses (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  git_submission_id   UUID        NOT NULL REFERENCES git_submissions(id),
  status              TEXT        NOT NULL DEFAULT 'pending',
  commit_count        INTEGER,
  authors             JSONB,
  first_commit_at     TIMESTAMPTZ,
  last_commit_at      TIMESTAMPTZ,
  deadline_rush_pct   NUMERIC(5,2),
  consistency_score   INTEGER,
  flags               JSONB,
  llm_advisory        TEXT,
  llm_status          TEXT        NOT NULL DEFAULT 'pending',
  fail_reason         TEXT,
  grade               NUMERIC(5,2),
  override_reason     TEXT        NOT NULL DEFAULT '',
  graded_by           UUID        REFERENCES users(id),
  graded_at           TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_git_analyses_submission ON git_analyses(git_submission_id, created_at DESC);

-- Worker job queue for async git analysis (separate from worker_jobs which requires exam_sessions FK)
CREATE TABLE git_analysis_jobs (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  git_analysis_id UUID        NOT NULL REFERENCES git_analyses(id),
  status          TEXT        NOT NULL DEFAULT 'queued',
  retries         INTEGER     NOT NULL DEFAULT 0,
  queued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_git_analysis_jobs_status ON git_analysis_jobs(status, queued_at);

-- Scheduled oral defense appointments per student
CREATE TABLE defense_appointments (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id UUID        NOT NULL REFERENCES actividades(id),
  student_id   UUID        NOT NULL REFERENCES users(id),
  scheduled_at TIMESTAMPTZ NOT NULL,
  grade        NUMERIC(5,2),
  graded_by    UUID        REFERENCES users(id),
  graded_at    TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (actividad_id, student_id)
);

CREATE INDEX idx_defense_appointments_actividad ON defense_appointments(actividad_id);
CREATE INDEX idx_defense_appointments_student   ON defense_appointments(student_id, actividad_id);
