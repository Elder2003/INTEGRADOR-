-- M5 Resolution & Tracing migration

-- Exam sessions: one per student per traced activity (examen or defensa_oral).
CREATE TABLE IF NOT EXISTS exam_sessions (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id        UUID        NOT NULL REFERENCES users(id),
  actividad_id      UUID        NOT NULL REFERENCES actividades(id),
  submission_id     UUID        REFERENCES submissions(id),
  status            TEXT        NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','completed','deadline_closed','abandoned','timed_out')),
  started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  closed_at         TIMESTAMPTZ,
  last_heartbeat_at TIMESTAMPTZ,
  last_event_hash   TEXT,
  event_count       INT         NOT NULL DEFAULT 0,
  UNIQUE (student_id, actividad_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_sessions_actividad_status
  ON exam_sessions(actividad_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_sessions_active_heartbeat
  ON exam_sessions(last_heartbeat_at)
  WHERE status = 'active';

-- Trace events: append-only event log. No UPDATE or DELETE ever (Constitution Principle I).
CREATE TABLE IF NOT EXISTS trace_events (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id   UUID        NOT NULL REFERENCES exam_sessions(id),
  sequence_num INT         NOT NULL,
  event_type   TEXT        NOT NULL
               CHECK (event_type IN (
                 'tab_blur','tab_focus','window_blur','window_focus',
                 'clipboard_copy','clipboard_paste',
                 'heartbeat','session_open','session_close'
               )),
  received_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  metadata     JSONB,
  event_hash   TEXT        NOT NULL,
  UNIQUE (session_id, sequence_num)
);

CREATE INDEX IF NOT EXISTS idx_trace_events_session_seq
  ON trace_events(session_id, sequence_num ASC);

-- Worker jobs: async processing jobs mirroring Redis stream entries for DB-level durability.
CREATE TABLE IF NOT EXISTS worker_jobs (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id    UUID        NOT NULL REFERENCES exam_sessions(id),
  submission_id UUID        NOT NULL REFERENCES submissions(id),
  stream_id     TEXT,
  status        TEXT        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','queued','processing','done','failed')),
  enqueued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  claimed_at    TIMESTAMPTZ,
  done_at       TIMESTAMPTZ,
  attempt_count INT         NOT NULL DEFAULT 0,
  error         TEXT
);

CREATE INDEX IF NOT EXISTS idx_worker_jobs_pending
  ON worker_jobs(status, enqueued_at)
  WHERE status IN ('pending', 'queued', 'processing');
