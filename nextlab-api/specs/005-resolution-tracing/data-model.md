# Data Model: M5 — Resolution & Tracing

**Migration**: `000005_tracing`

---

## New Tables

### `exam_sessions`

One record per student per traced activity. Created when a student opens an `examen` or
`defensa_oral` activity. The session lifecycle is bounded by the activity deadline.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | |
| `student_id` | UUID | NOT NULL, FK → users(id) | |
| `actividad_id` | UUID | NOT NULL, FK → actividades(id) | |
| `submission_id` | UUID | FK → submissions(id), nullable | Linked when session closes with a submission |
| `status` | TEXT | NOT NULL, DEFAULT 'active', CHECK | One of: `active`, `completed`, `deadline_closed`, `abandoned`, `timed_out` |
| `started_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() | Server time session was opened |
| `closed_at` | TIMESTAMPTZ | nullable | Server time session was closed |
| `last_heartbeat_at` | TIMESTAMPTZ | nullable | Updated on each `heartbeat` event receipt |
| `last_event_hash` | TEXT | nullable | Hash of the most recent trace event; seed for the next event's chain |
| `event_count` | INT | NOT NULL, DEFAULT 0 | Monotonically incremented; next event's `sequence_num` = event_count + 1 |
| | | UNIQUE(student_id, actividad_id) | No duplicate sessions per student per activity |

**Indexes**:
```sql
CREATE INDEX idx_exam_sessions_actividad_status
  ON exam_sessions(actividad_id, status);

CREATE INDEX idx_exam_sessions_active_heartbeat
  ON exam_sessions(last_heartbeat_at)
  WHERE status = 'active';
```

**Status transitions**:
```
active → completed       (submission linked before or at deadline)
active → deadline_closed (deadline reached, no linked submission)
active → abandoned       (WebSocket closes without session_close event, before deadline)
active → timed_out       (no heartbeat for > 5 minutes before deadline)
```
All transitions are one-way. No status can move back to `active`.

---

### `trace_events`

Immutable append-only event log. No `UPDATE` or `DELETE` operations are ever issued against
this table (Constitution Principle I).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | |
| `session_id` | UUID | NOT NULL, FK → exam_sessions(id) | |
| `sequence_num` | INT | NOT NULL | 1-based, monotonically increasing within the session |
| `event_type` | TEXT | NOT NULL, CHECK | One of: `tab_blur`, `tab_focus`, `window_blur`, `window_focus`, `clipboard_copy`, `clipboard_paste`, `heartbeat`, `session_open`, `session_close` |
| `received_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() | Server clock time of receipt |
| `metadata` | JSONB | nullable | Optional context (e.g., window title). Clipboard content is NEVER stored. |
| `event_hash` | TEXT | NOT NULL | SHA-256 chained hash (see hash formula) |
| | | UNIQUE(session_id, sequence_num) | Guarantees monotonic, gap-free ordering |

**Index**:
```sql
CREATE INDEX idx_trace_events_session_seq
  ON trace_events(session_id, sequence_num ASC);
```

**Hash formula** (SHA-256 over canonical string):
```
hash = SHA256("<session_id>|<sequence_num>|<event_type>|<received_at_unix_nano>|<canonical_metadata_json>|<prev_hash>")
```
- `canonical_metadata_json`: JSON with keys sorted lexicographically, no whitespace; `{}` if metadata is null.
- `prev_hash` for sequence_num=1: `SHA256("<session_id>")` — chain is rooted in the unforgeable session ID.
- `prev_hash` for sequence_num>1: `event_hash` from the row where `sequence_num = this - 1`.

---

### `worker_jobs`

Tracks async processing jobs for completed exam sessions. Mirrors Redis Streams entries
for DB-level durability (see research.md Decision 4).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | |
| `session_id` | UUID | NOT NULL, FK → exam_sessions(id) | |
| `submission_id` | UUID | NOT NULL, FK → submissions(id) | |
| `stream_id` | TEXT | nullable | Redis stream message ID; set after successful XADD |
| `status` | TEXT | NOT NULL, DEFAULT 'pending', CHECK | One of: `pending`, `queued`, `processing`, `done`, `failed` |
| `enqueued_at` | TIMESTAMPTZ | NOT NULL, DEFAULT now() | |
| `claimed_at` | TIMESTAMPTZ | nullable | Set when worker picks up the job |
| `done_at` | TIMESTAMPTZ | nullable | Set when worker writes acknowledgement |
| `attempt_count` | INT | NOT NULL, DEFAULT 0 | Incremented on each claim |
| `error` | TEXT | nullable | Last error message on failure |

**Index**:
```sql
CREATE INDEX idx_worker_jobs_pending
  ON worker_jobs(status, enqueued_at)
  WHERE status IN ('pending', 'queued', 'processing');
```

**Durability pattern** (from research.md Decision 4):
```
1. INSERT INTO worker_jobs (session_id, submission_id, status='pending') RETURNING id
2. XADD nextlab:sessions:jobs * job_id <id> session_id <sid> submission_id <subid>
3. UPDATE worker_jobs SET stream_id=<stream-id>, status='queued' WHERE id=<id>
```
If step 2 fails: job row remains `pending`; worker startup sweep re-enqueues via XADD.
If step 3 fails: stream message exists but DB status is still `pending`; worker re-enqueues
on startup (idempotent: XCLAIM detects duplicate before processing).

---

## Migration Files

### `000005_tracing.up.sql`

```sql
-- M5 Resolution & Tracing migration

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

-- Append-only event log; no UPDATE/DELETE ever (Constitution Principle I)
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

-- Async worker job tracking (mirrors Redis Streams entries for durability)
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
```

### `000005_tracing.down.sql`

```sql
DROP TABLE IF EXISTS worker_jobs;
DROP TABLE IF EXISTS trace_events;
DROP TABLE IF EXISTS exam_sessions;
```

---

## Entity Relationships

```
actividades ──────────── exam_sessions ──────────── trace_events
    │                          │
    │                          │
users (student_id) ────────────┘
                               │
submissions ───────────────────┘
                               │
                          worker_jobs
```

- One `actividad` (tipo=examen or defensa_oral) → zero or many `exam_sessions` (one per student)
- One `exam_session` → zero or many `trace_events` (ordered by sequence_num)
- One `exam_session` → zero or one `worker_job` (created when session closes with submission)
- One `submission` → zero or one `worker_job` via the exam session

---

## Validation Rules

| Rule | Source | Enforcement |
|---|---|---|
| Session only for `examen`/`defensa_oral` | FR-001, FR-020 | Service layer checks `actividades.trace_enabled = true` |
| One active session per student per activity | FR-003 | `UNIQUE(student_id, actividad_id)` on `exam_sessions` |
| Events append-only | FR-017, Principle I | No UPDATE/DELETE in repository; CHECK on event_type |
| Clipboard content never stored | FR-008 | Service strips content from `clipboard_*` metadata before INSERT |
| sequence_num monotonically increasing | FR-005 | `event_count` column incremented atomically in same transaction as INSERT |
| Session closed before new events accepted | FR-011c | Service checks `status = 'active'` before AppendEvent |
| Docente sees only own courses' sessions | FR-018, Principle IV | All queries JOIN through `cursos.docente_id` or `codocente_grants` |
| Student cannot read their own trace | FR-019, Principle IV | No student-facing trace endpoint |
