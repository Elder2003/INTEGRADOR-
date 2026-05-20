-- M4 File Storage migration
-- Adds late_submission_allowed to actividades, immutable config change log, and submissions table.

-- 1. Extend actividades with the docente-configurable late-submission flag.
ALTER TABLE actividades
  ADD COLUMN IF NOT EXISTS late_submission_allowed BOOLEAN NOT NULL DEFAULT true;

-- 2. Immutable audit log for activity config flag changes (append-only, never UPDATE/DELETE).
CREATE TABLE IF NOT EXISTS actividad_config_changes (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  actividad_id   UUID        NOT NULL REFERENCES actividades(id),
  changed_by     UUID        NOT NULL REFERENCES users(id),
  field_name     TEXT        NOT NULL,
  previous_value TEXT        NOT NULL,
  new_value      TEXT        NOT NULL,
  changed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_actividad_config_changes_actividad
  ON actividad_config_changes(actividad_id, changed_at DESC);

-- 3. Submissions table — one active submission per student per activity.
CREATE TABLE IF NOT EXISTS submissions (
  id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id             UUID        NOT NULL REFERENCES users(id),
  actividad_id           UUID        NOT NULL REFERENCES actividades(id),
  storage_key            TEXT        NOT NULL,
  original_filename      TEXT        NOT NULL,
  file_size_bytes        BIGINT      NOT NULL CHECK (file_size_bytes > 0),
  mime_type              TEXT        NOT NULL,
  submitted_at           TIMESTAMPTZ NOT NULL,
  is_late                BOOLEAN     NOT NULL,
  edit_window_expires_at TIMESTAMPTZ,
  last_replaced_at       TIMESTAMPTZ,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (student_id, actividad_id)
);

CREATE INDEX IF NOT EXISTS idx_submissions_actividad ON submissions(actividad_id);
CREATE INDEX IF NOT EXISTS idx_submissions_student   ON submissions(student_id);
