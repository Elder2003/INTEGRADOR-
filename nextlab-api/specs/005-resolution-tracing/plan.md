# Implementation Plan: M5 — Resolution & Tracing

**Branch**: `005-resolution-tracing` | **Date**: 2026-05-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-resolution-tracing/spec.md`

---

## Summary

Implement real-time exam session tracing for nextlab-api: students open a WebSocket
connection when they start an `examen` or `defensa_oral` activity; browser events
(tab blur/focus, clipboard, heartbeat) are stored as an immutable SHA-256 hash-chained
event log; sessions close automatically at the activity deadline or when a file is
submitted; a background Redis Streams worker picks up the job and writes an
acknowledgement record. Docentes can retrieve the full event trace and verify its
integrity.

**Technical approach**: New `internal/tracing/` module · `gofiber/websocket/v2` adapter ·
WebSocket JWT auth via query param validated before upgrade · in-memory `sync.Map` hub
for multi-tab support · per-connection rate limiter (30 events/min) · SHA-256 hash chain
seeded on session ID · Redis Streams consumer group `analysis-workers` with XACK +
XCLAIM crash recovery · DB-mirrored `worker_jobs` for two-layer durability · M4
`submission/service.go` extended to close session + enqueue job on file upload ·
`cmd/worker/main.go` expanded with consumer loop + deadline sweep ticker

---

## Technical Context

**Language/Version**: Go 1.22 (module declares `go 1.25.0`)

**Primary Dependencies**:
- `gofiber/fiber/v2` — HTTP framework (locked by constitution)
- `gofiber/websocket/v2` — Fiber WebSocket adapter wrapping gorilla/websocket *(new in M5)*
- `jackc/pgx/v5/pgxpool` — PostgreSQL pool (from M1)
- `redis/go-redis/v9` — Redis Streams consumer group (from M1; extended in M5)
- `google/uuid` — UUID generation (already in go.sum)
- `crypto/sha256` — hash chaining (stdlib, no new dependency)
- `sync` — `sync.Map` for WebSocket hub (stdlib)

**Storage**:
- PostgreSQL 16: 3 new tables (`exam_sessions`, `trace_events`, `worker_jobs`)
- Redis 7 Streams: stream key `nextlab:sessions:jobs`, consumer group `analysis-workers`

**New environment variables**: None — Redis is already configured (`REDIS_ADDR`).

**Target Platform**: Linux/AMD64 (Docker Compose, same as M1–M4)

**Performance Goals**:
- Session open confirmation ≤ 1 second (SC-001)
- File upload response ≤ 2 seconds regardless of worker activity (SC-003)
- Worker acknowledgement within 3 minutes of enqueueing (SC-004)

**Constraints**:
- `trace_events`: no `UPDATE` or `DELETE` ever — append-only (Constitution Principle I)
- Async worker MUST NOT be called inline in any HTTP handler (Constitution Principle II)
- Session only created for activities where `trace_enabled = true` (generated column in
  `actividades`: `tipo IN ('examen', 'defensa_oral')`) (Constitution Principle VI)
- Student MUST NOT have access to their own trace events (Constitution Principle IV)
- `received_at` is always server clock; client timestamp is metadata only
- Clipboard event content is stripped before storage (FR-008)
- JWT for WebSocket passed as `?token=<jwt>` query param; validated before upgrade

**Scale/Scope**: ~800–1500 students, ~50–100 docentes; peak: hundreds of concurrent exam
sessions; trace events per session: typically 30–200 events over 60–120 minutes

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Immutable Evidence | ✅ **PRIMARY** | `trace_events` is append-only. Repository layer exposes no UpdateTraceEvent or DeleteTraceEvent. Hash chain makes tampering detectable. `worker_jobs` status transitions only — no row deletion. |
| II. Asynchronous Analysis Isolation | ✅ **PRIMARY** | Worker job enqueued to Redis Streams; HTTP response for file upload returns immediately. Worker runs in a separate `cmd/worker` container. No analysis logic executes inline in any HTTP handler. |
| III. Human Authority Over AI | ✅ N/A | No AI calls in M5. Worker writes an acknowledgement record only — no decisions, no scores. |
| IV. Strict Role-Based Data Isolation | ✅ **GATE** | Docente trace access scoped through `cursos.docente_id`. Co-docente requires active `codocente_grants`. Student has NO trace endpoints — Principle IV §Estudiante prohibits this explicitly. Session creation checks `inscripciones` enrollment. |
| V. Dual RAG Independence | ✅ N/A | No RAG in M5. |
| VI. Non-Invasive Auditing | ✅ **GATE** | Session creation gated on `actividades.trace_enabled = true`. No camera, microphone, or filesystem access. Student is informed via the session-open response before capture begins. |
| VII. Submission Always Accepted | ✅ Maintained | `UploadFile` extension: session close + job enqueue happen AFTER submission is persisted. If enqueue fails, submission still returns 201. Queue failure logged; worker sweep re-enqueues within 60s. |

**Result: PASS** — Principles I and II are primary enforcement drivers. Principle IV
prohibits any student-facing trace endpoint. Principle VI limits trace to examen/defensa_oral.

---

## Project Structure

### Documentation (this feature)

```text
specs/005-resolution-tracing/
├── plan.md                   ← this file
├── research.md               ← Phase 0 output
├── data-model.md             ← Phase 1 output
├── quickstart.md             ← Phase 1 output
├── contracts/
│   └── tracing.md            ← Phase 1 output
├── checklists/
│   └── requirements.md
└── tasks.md                  ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
nextlab-api/
├── cmd/
│   ├── api/
│   │   └── main.go                       # EXTEND: add tracing.RegisterRoutes(app, pgPool, rdb, cfg, authMW, roleGuard)
│   └── worker/
│       └── main.go                       # MAJOR: add pgxpool, consumer group loop,
│                                         #        XCLAIM crash recovery, deadline sweep ticker
├── internal/
│   ├── tracing/                          # Expand from package stub
│   │   ├── module.go                     # RegisterRoutes: HTTP + WebSocket routes wired
│   │   ├── model.go                      # ExamSession, TraceEvent, WorkerJob + response types
│   │   ├── repository.go                 # All DB queries (session CRUD, AppendEvent, GetTrace, worker_jobs)
│   │   ├── service.go                    # OpenSession, AppendEvent, CloseSession, EnqueueJob, VerifyChain
│   │   ├── handler_session.go            # POST/GET /actividades/:aid/session
│   │   ├── hub.go                        # WebSocket connection hub (sync.Map sessionID→[]*Conn)
│   │   ├── handler_ws.go                 # WS JWT middleware + per-connection goroutine
│   │   └── handler_docente.go            # GET /submissions/:sid/trace + /trace/verify
│   └── submission/
│       └── service.go                    # EXTEND UploadFile + ReplaceFile: detect trace_enabled,
│                                         #   call tracingSvc.CloseSessionWithSubmission
├── migrations/
│   ├── 000005_tracing.up.sql             # exam_sessions, trace_events, worker_jobs
│   └── 000005_tracing.down.sql
├── go.mod                                # Add gofiber/websocket/v2
└── go.sum                                # Updated by go mod tidy
```

**Structure Decision**: All tracing logic lives in `internal/tracing/`. The submission
module gets only a targeted extension in `service.go`. The worker (`cmd/worker/main.go`)
is expanded in-place alongside the existing Redis setup.

---

## Phase 0: Research Summary

See [research.md](research.md) for the full decision log. Key decisions:

| Topic | Decision |
|---|---|
| WebSocket library | `gofiber/websocket/v2` — Fiber-native adapter, consistent with locked stack |
| JWT over WebSocket | Query param `?token=<jwt>`; validated by Fiber middleware before upgrade |
| Job queue | Redis Streams, consumer group `analysis-workers`, XADD + XREADGROUP + XACK |
| DB durability | `worker_jobs` row inserted before XADD; `stream_id` stored back after XADD |
| Session deadline close | Lazy check on event receipt + worker ticker every 60s |
| Hash chaining | SHA-256 of `<session_id>\|<seq>\|<type>\|<ts_ns>\|<meta_json>\|<prev_hash>` |
| Rate limiting | 30 events/min per connection, in-memory per goroutine |
| Multi-tab hub | `sync.Map` keyed by session ID, value `[]*websocket.Conn` |
| M4 integration | Extend `submission/service.go` UploadFile/ReplaceFile after successful DB write |

---

## Phase 1: Design Summary

### Data Model

See [data-model.md](data-model.md). Migration `000005_tracing`:

| Table | Description |
|---|---|
| `exam_sessions` | One per student per traced activity; status, heartbeat, last_event_hash, event_count |
| `trace_events` | Append-only, SHA-256 chained; UNIQUE(session_id, sequence_num) |
| `worker_jobs` | Async job tracking; mirrors Redis stream entries for two-layer durability |

### API Contract

See [contracts/tracing.md](contracts/tracing.md). Full endpoint list:

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/actividades/:aid/session` | estudiante | Open or resume exam session |
| `GET` | `/actividades/:aid/session` | estudiante | Get current session |
| `GET` | `/sessions/:sid/ws?token=<jwt>` | estudiante | WebSocket real-time event stream |
| `GET` | `/submissions/:sid/trace` | docente, co_docente | Retrieve full trace event list |
| `GET` | `/submissions/:sid/trace/verify` | docente, co_docente | Verify hash chain integrity |

### Quickstart

See [quickstart.md](quickstart.md). 13-step verification covering: session open + idempotency,
type rejection, WebSocket connect + events, rate limit, submission close, worker ack, docente
trace retrieval, hash chain verify + tamper test, cross-course 403, crash recovery.

---

## Key Implementation Notes

### go.mod: add gofiber/websocket/v2

```bash
go get github.com/gofiber/websocket/v2
go mod tidy
```

---

### Session creation (idempotent INSERT)

```sql
INSERT INTO exam_sessions (student_id, actividad_id, status)
VALUES ($1, $2, 'active')
ON CONFLICT (student_id, actividad_id) DO NOTHING;

SELECT id, status, started_at, event_count
FROM exam_sessions
WHERE student_id = $1 AND actividad_id = $2;
```

If returned `status != 'active'`: return 410 Gone with the status.
Preconditions checked before INSERT: (1) student enrolled via `inscripciones`, (2) `actividades.trace_enabled = true`.

---

### WebSocket JWT middleware (before upgrade)

```go
app.Use("/sessions/:sid/ws", func(c *fiber.Ctx) error {
    token := c.Query("token")
    if token == "" {
        return fiber.ErrUnauthorized
    }
    claims, err := auth.ParseJWT(token, cfg.JWTSecret)
    if err != nil {
        return fiber.ErrUnauthorized
    }
    // Do NOT log token
    c.Locals("user_id", claims.UserID)
    c.Locals("user_role", claims.Role)
    return c.Next()
})
app.Get("/sessions/:sid/ws", websocket.New(wsHandler))
```

---

### AppendEvent: atomic hash chain + sequence increment

Single transaction per event:
```sql
BEGIN;
SELECT status, event_count, last_event_hash FROM exam_sessions WHERE id = $1 FOR UPDATE;
-- Service computes new hash in Go
INSERT INTO trace_events (session_id, sequence_num, event_type, received_at, metadata, event_hash)
VALUES ($1, $event_count+1, $type, now(), $meta, $new_hash);
UPDATE exam_sessions
SET event_count = event_count + 1, last_event_hash = $new_hash,
    last_heartbeat_at = CASE WHEN $type = 'heartbeat' THEN now() ELSE last_heartbeat_at END
WHERE id = $1 AND status = 'active';
-- 0 rows updated = session closed → return ErrSessionClosed
COMMIT;
```

Hash formula:
```
prev = last_event_hash (or SHA256(session_id) if event_count=0)
canonical_meta = sortedKeysJSON(metadata) or "{}" if nil
raw = "<session_id>|<seq>|<type>|<received_at.UnixNano()>|<canonical_meta>|<prev>"
event_hash = hex(SHA256(raw))
```

---

### CloseSessionWithSubmission (called from submission/service.go after COMMIT)

```go
func (s *Service) CloseSessionWithSubmission(ctx context.Context,
    studentID, actividadID, submissionID uuid.UUID) {

    session := s.repo.GetActiveSession(ctx, studentID, actividadID)
    if session == nil { return } // no session — silent no-op

    s.repo.CloseSession(ctx, session.ID, submissionID, "completed")
    s.AppendEvent(ctx, session.ID, "session_close", nil)

    // Two-layer durability: DB row first, then XADD
    jobID := s.repo.InsertWorkerJob(ctx, session.ID, submissionID)
    streamID, err := s.rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: "nextlab:sessions:jobs",
        Values: map[string]any{"job_id": jobID, "session_id": session.ID, "submission_id": submissionID},
    })
    if err != nil {
        slog.ErrorContext(ctx, "xadd failed; job will be re-enqueued by sweep", "job_id", jobID, "err", err)
        return
    }
    s.repo.SetJobStreamID(ctx, jobID, streamID)
}
```

Extension in `submission/service.go` (after COMMIT in UploadFile):
```go
if act.TraceEnabled {
    s.tracingSvc.CloseSessionWithSubmission(ctx, studentID, actividadID, sub.ID)
}
```

---

### Worker consumer loop (cmd/worker/main.go)

```go
func runConsumerLoop(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool) {
    rdb.XGroupCreateMkStream(ctx, "nextlab:sessions:jobs", "analysis-workers", "0")
    claimOrphanedMessages(ctx, rdb, db) // XCLAIM idle > 30s on startup
    go runDeadlineSweep(ctx, db, rdb)   // ticker every 60s

    workerName := "worker-" + mustHostname()
    for {
        results, _ := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
            Group:    "analysis-workers",
            Consumer: workerName,
            Streams:  []string{"nextlab:sessions:jobs", ">"},
            Count:    1,
            Block:    5 * time.Second,
        })
        for _, stream := range results {
            for _, msg := range stream.Messages {
                processJob(ctx, rdb, db, msg)
            }
        }
    }
}

func processJob(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, msg redis.XMessage) {
    jobID := msg.Values["job_id"].(string)
    markJobStatus(ctx, db, jobID, "processing")
    // M5 scope: acknowledgement only. M8 adds heuristics. M9 adds format validation.
    markJobDone(ctx, db, jobID)
    rdb.XAck(ctx, "nextlab:sessions:jobs", "analysis-workers", msg.ID)
    slog.InfoContext(ctx, "job acknowledged", "job_id", jobID)
}
```

---

### Deadline sweep (worker ticker, every 60s)

Deadline-closed query:
```sql
SELECT es.id, es.submission_id
FROM exam_sessions es
JOIN actividades a ON a.id = es.actividad_id
WHERE es.status = 'active' AND a.deadline < NOW()
```

Timed-out query (heartbeat gap > 5 min, still before deadline):
```sql
SELECT es.id FROM exam_sessions es
JOIN actividades a ON a.id = es.actividad_id
WHERE es.status = 'active'
  AND a.deadline > NOW()
  AND es.last_heartbeat_at < NOW() - INTERVAL '5 minutes'
```

---

### Hash chain verification

```go
func (s *Service) VerifyChain(ctx context.Context, sessionID uuid.UUID) (bool, int) {
    events := s.repo.GetAllEvents(ctx, sessionID) // ORDER BY sequence_num ASC
    prevHash := fmt.Sprintf("%x", sha256.Sum256([]byte(sessionID.String())))
    for _, e := range events {
        raw := fmt.Sprintf("%s|%d|%s|%d|%s|%s",
            sessionID, e.SequenceNum, e.EventType,
            e.ReceivedAt.UnixNano(), canonicalJSON(e.Metadata), prevHash)
        computed := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
        if computed != e.EventHash {
            return false, e.SequenceNum
        }
        prevHash = e.EventHash
    }
    return true, len(events)
}
```

---

### Worker: add PostgreSQL connection

```go
pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
if err != nil { slog.Error("postgres failed", "err", err); os.Exit(1) }
defer pgPool.Close()
```

`cfg.DatabaseURL` already exists from M1's `config.Load()`.

---

## Key Risks & Mitigations

| Risk | Mitigation |
|---|---|
| `gofiber/websocket/v2` incompatible with Fiber v2.52.0 | Confirmed compatible; it is the official Fiber WebSocket adapter |
| Redis XReadGroup blocks indefinitely on empty stream | `Block: 5 * time.Second` gives the goroutine periodic context-cancel checks |
| Orphaned worker_jobs (XADD succeeded, DB update failed) | Worker startup sweep queries `worker_jobs WHERE status='pending'` and re-enqueues |
| Multi-tab rate limit collision | Each WS connection has its own goroutine-local counter — no shared mutex |
| Concurrent AppendEvent breaks hash chain | `FOR UPDATE` on `exam_sessions` row serializes concurrent appends |
