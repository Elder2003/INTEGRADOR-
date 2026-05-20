# Tasks: M5 — Resolution & Tracing

**Input**: Design documents from `specs/005-resolution-tracing/`

**Prerequisites**: plan.md ✅ | spec.md ✅ | research.md ✅ | data-model.md ✅ | contracts/tracing.md ✅ | quickstart.md ✅

**Organization**: Tasks are grouped by user story. Each phase is independently deliverable and testable. No test tasks generated (not requested in spec).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the WebSocket library dependency before any feature work begins.

- [X] T001 Add `github.com/gofiber/websocket/v2` to `nextlab-api/go.mod` via `go get github.com/gofiber/websocket/v2` and run `go mod tidy`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database migration, Go model structs, and module skeletons — everything all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T002 Write `nextlab-api/migrations/000005_tracing.up.sql` per data-model.md: `CREATE TABLE exam_sessions` (id, student_id FK users, actividad_id FK actividades, submission_id FK submissions nullable, status TEXT NOT NULL DEFAULT 'active' CHECK(status IN('active','completed','deadline_closed','abandoned','timed_out')), started_at TIMESTAMPTZ NOT NULL DEFAULT now(), closed_at TIMESTAMPTZ nullable, last_heartbeat_at TIMESTAMPTZ nullable, last_event_hash TEXT nullable, event_count INT NOT NULL DEFAULT 0, UNIQUE(student_id, actividad_id)) with indexes on (actividad_id, status) and (last_heartbeat_at) WHERE status='active'; `CREATE TABLE trace_events` (id, session_id FK exam_sessions, sequence_num INT NOT NULL, event_type TEXT NOT NULL CHECK(one of 9 valid types), received_at TIMESTAMPTZ NOT NULL DEFAULT now(), metadata JSONB nullable, event_hash TEXT NOT NULL, UNIQUE(session_id, sequence_num)) with index on (session_id, sequence_num ASC); `CREATE TABLE worker_jobs` (id, session_id FK exam_sessions, submission_id FK submissions, stream_id TEXT nullable, status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN('pending','queued','processing','done','failed')), enqueued_at TIMESTAMPTZ NOT NULL DEFAULT now(), claimed_at TIMESTAMPTZ nullable, done_at TIMESTAMPTZ nullable, attempt_count INT NOT NULL DEFAULT 0, error TEXT nullable) with partial index on (status, enqueued_at) WHERE status IN('pending','queued','processing')
- [X] T003 [P] Write `nextlab-api/migrations/000005_tracing.down.sql`: `DROP TABLE IF EXISTS worker_jobs`; `DROP TABLE IF EXISTS trace_events`; `DROP TABLE IF EXISTS exam_sessions`
- [X] T004 [P] Create `nextlab-api/internal/tracing/model.go` with Go structs: `ExamSession` (all DB columns as Go fields using pgx-compatible types); `ExamSessionResponse` (id, actividad_id, status, started_at, ws_url string); `TraceEvent` (all DB columns); `TraceEventItem` (sequence_num, event_type, received_at, metadata, event_hash — used in API response); `TraceResponse` (session_id, student_id, actividad_id, status, started_at, closed_at, event_count, events []TraceEventItem); `VerifyResponse` (valid bool, event_count int, first_invalid_sequence *int — nullable); `WorkerJob` (all DB columns); `ClientEvent` (type string, client_ts *int64 for incoming WS messages); `TraceAck` (type="ack", sequence_num int, received_at time.Time — outgoing WS message)
- [X] T005 [P] Create `nextlab-api/internal/tracing/module.go` skeleton: `RegisterRoutes(app *fiber.App, db *pgxpool.Pool, rdb *redis.Client, cfg *config.Config, authMW fiber.Handler, roleGuard func(...string) fiber.Handler)` — include Hub, Repository, Service struct initializations; route wiring completed per user story phase
- [X] T006 [P] Create `nextlab-api/internal/tracing/repository.go` with `Repository` struct (db *pgxpool.Pool, rdb *redis.Client); implement: `GetActividadTraceInfo(ctx, actividadID, studentID)` returns (traceEnabled bool, enrolled bool, err error) via JOIN actividades + inscripciones; `InsertSessionIdempotent(ctx, studentID, actividadID)` uses INSERT ON CONFLICT DO NOTHING; `GetSessionByStudentActivity(ctx, studentID, actividadID) *ExamSession`; `CloseSession(ctx, sessionID uuid.UUID, submissionID *uuid.UUID, status string)`; stub all other method signatures that will be filled in per US phase
- [X] T007 [P] Create `nextlab-api/internal/tracing/service.go` with `Service` struct (repo *Repository, rdb *redis.Client); define sentinel errors: `ErrSessionClosed`, `ErrTraceNotApplicable`, `ErrNotEnrolled`, `ErrForbidden`, `ErrSessionNotFound`; stub OpenSession, AppendEvent, CloseSessionWithSubmission, VerifyChain signatures; implement `canonicalJSON(metadata map[string]any) string` helper (sorts keys, no whitespace, returns `"{}"` for nil); implement `computeEventHash(sessionID uuid.UUID, seq int, eventType string, receivedAtNano int64, canonicalMeta string, prevHash string) string` using `crypto/sha256`

**Checkpoint**: Run `docker compose up --build` — server starts, migration `000005` applies cleanly (`\dt` shows `exam_sessions`, `trace_events`, `worker_jobs`), no compilation errors.

---

## Phase 3: User Story 1 — Student Opens an Exam Session (Priority: P1) 🎯 MVP

**Goal**: A student can open an `examen` or `defensa_oral` activity and receive an active session. The system rejects trace activation for other activity types. Opening the same activity twice returns the existing session without creating a duplicate.

**Independent Test**: POST /actividades/:aid/session for examen → 201 with status=active and ws_url; POST same endpoint again → 200 with same session ID; POST for a tarea activity → 422 trace_not_applicable; unenrolled student → 403.

### Implementation

- [X] T008 [P] [US1] Implement session creation queries in `nextlab-api/internal/tracing/repository.go`: `InsertSessionIdempotent(ctx, studentID, actividadID)` (INSERT INTO exam_sessions ON CONFLICT (student_id, actividad_id) DO NOTHING); `GetSessionByStudentActivity(ctx, studentID, actividadID) *ExamSession` (SELECT all columns; returns nil if not found)
- [X] T009 [US1] Implement `OpenSession(ctx context.Context, studentID, actividadID uuid.UUID) (*ExamSession, bool, error)` in `nextlab-api/internal/tracing/service.go`: (1) call repo.GetActividadTraceInfo — if !enrolled return ErrNotEnrolled; if !traceEnabled return ErrTraceNotApplicable; (2) call InsertSessionIdempotent; (3) fetch session with GetSessionByStudentActivity; (4) if session.Status != "active" return (nil, false, ErrSessionClosed); return (session, isNew bool, nil) where isNew = session.StartedAt within last second
- [X] T010 [US1] Implement `POST /actividades/:aid/session` and `GET /actividades/:aid/session` handlers in `nextlab-api/internal/tracing/handler_session.go`: POST: parse aid UUID, read user_id from c.Locals("user_id"), call svc.OpenSession, map ErrNotEnrolled→403, ErrTraceNotApplicable→422, ErrSessionClosed→410 with status field, isNew→201/200, return ExamSessionResponse with ws_url="/sessions/"+session.ID+"/ws"; GET: return 200 ExamSessionResponse or 404
- [X] T011 [US1] Wire `POST /actividades/:aid/session` and `GET /actividades/:aid/session` in `nextlab-api/internal/tracing/module.go` with `authMW` + `roleGuard("estudiante")`
- [X] T012 [US1] Extend `nextlab-api/cmd/api/main.go`: import `internal/tracing`; add `tracing.RegisterRoutes(app, pgPool, rdb, cfg, auth.AuthMiddleware(cfg), auth.RoleGuard)` call after existing route registrations

**Checkpoint**: US1 fully testable. POST session for examen → 201; repeat → 200 (same ID); POST for tarea → 422; unenrolled student → 403; DB row exists with status='active'.

---

## Phase 4: User Story 2 — Browser Events Are Captured (Priority: P2)

**Goal**: Students connected via WebSocket can send browser events that are stored as an immutable SHA-256 hash-chained event log. Rate limiting prevents flooding. Session is marked abandoned when the connection closes without a formal close event.

**Independent Test**: Connect WebSocket with valid JWT → session_open appended; send tab_blur → ACK with sequence_num=2 and correct hash chain; send clipboard_copy → stored without content in metadata; send 31 events/min → 31st gets error frame, is NOT stored; disconnect without session_close → session status becomes abandoned.

### Implementation

- [X] T013 [P] [US2] Create `nextlab-api/internal/tracing/hub.go`: `Hub` struct wrapping `sync.Map` (key: string sessionID, value: []*websocket.Conn); `NewHub() *Hub`; `Add(sessionID string, c *websocket.Conn)`; `Remove(sessionID string, c *websocket.Conn)` (filters out the specific conn from the slice); `GetConns(sessionID string) []*websocket.Conn`; `BroadcastClose(sessionID string, msg []byte)` (writes close frame to all connections for a session then removes them)
- [X] T014 [P] [US2] Implement `AppendEvent` query in `nextlab-api/internal/tracing/repository.go`: single TX — `SELECT status, event_count, last_event_hash FROM exam_sessions WHERE id=$1 FOR UPDATE`; `INSERT INTO trace_events (session_id, sequence_num, event_type, received_at, metadata, event_hash) VALUES ($1, $event_count+1, $type, now(), $meta, $hash) RETURNING received_at`; `UPDATE exam_sessions SET event_count=event_count+1, last_event_hash=$hash, last_heartbeat_at=CASE WHEN $type='heartbeat' THEN now() ELSE last_heartbeat_at END WHERE id=$1 AND status='active'`; if UPDATE rowsAffected=0 return ErrSessionClosed; return (sequenceNum int, receivedAt time.Time, err error)
- [X] T015 [US2] Implement `AppendEvent(ctx, sessionID uuid.UUID, eventType string, metadata map[string]any) (*TraceAck, error)` in `nextlab-api/internal/tracing/service.go`: (1) fetch current session for prev hash (use the FOR UPDATE path in the TX, so the hash is computed inside the TX); strip clipboard content from metadata before passing; compute prevHash = SHA256(sessionID.String()) if event_count=0, else last_event_hash from session row; compute event_hash using computeEventHash; call repo.AppendEvent; return TraceAck{Type:"ack", SequenceNum: seq, ReceivedAt: ts}; return ErrSessionClosed on that error
- [X] T016 [US2] Implement WebSocket handler in `nextlab-api/internal/tracing/handler_ws.go`: (a) `wsJWTMiddleware(cfg *config.Config) fiber.Handler`: read `c.Query("token")`, call auth.ParseJWT, set c.Locals("user_id") and c.Locals("user_role"), return fiber.ErrUnauthorized on failure — do NOT log the token; (b) `makeWSHandler(hub *Hub, svc *Service) func(*websocket.Conn)`: on connect: register in hub + defer Remove; validate the session belongs to the authenticated user (query by session ID + user_id); append session_open event; send `{"type":"connected","session_id":"...","event_count":N}`; enter read loop with per-connection rate limiter (rateCount int, rateReset time.Time, window 60s, limit 30); on each message: decode ClientEvent, rate check, call svc.AppendEvent, send TraceAck or error frame; on ErrSessionClosed: send session_closed frame + break; on loop exit: if session still active → call repo.CloseSession(ctx, sessionID, nil, "abandoned") + append session_close event
- [X] T017 [US2] Wire WebSocket route in `nextlab-api/internal/tracing/module.go`: `app.Use("/sessions/:sid/ws", wsJWTMiddleware(cfg))`; `app.Get("/sessions/:sid/ws", websocket.New(makeWSHandler(hub, svc)))`; instantiate `hub := tracing.NewHub()` in RegisterRoutes and pass to makeWSHandler

**Checkpoint**: US2 fully testable. WS connect + session_open appended (sequence_num=1); tab_blur ACK with sequence_num=2; clipboard_copy metadata has no content; rate limit enforced; disconnect without close → session=abandoned.

---

## Phase 5: User Story 3 — Session Completion Triggers Async Processing (Priority: P3)

**Goal**: When a student submits their exam file, the session closes and a worker job is enqueued. The worker picks up the job and stores an acknowledgement within 3 minutes. Worker handles restart via XCLAIM recovery. A sweep closes sessions that expired while disconnected.

**Independent Test**: Submit exam file → 201 within 2s; session transitions to completed; worker_jobs row created; within 3 min job status=done; worker restart recovers in-flight job without duplication.

### Implementation

- [X] T018 [P] [US3] Implement worker_jobs and sweep queries in `nextlab-api/internal/tracing/repository.go`: `InsertWorkerJob(ctx, sessionID, submissionID uuid.UUID) uuid.UUID` (INSERT INTO worker_jobs RETURNING id); `SetJobStreamID(ctx, jobID uuid.UUID, streamID string)`; `MarkJobStatus(ctx, jobID uuid.UUID, status string, claimedAt *time.Time, doneAt *time.Time, errMsg *string)`; `GetOrphanedJobs(ctx) []WorkerJob` (status='pending' AND enqueued_at < NOW()-INTERVAL '60 seconds'); `GetDeadlinedSessions(ctx) []ExamSession` (JOIN actividades: status='active' AND a.deadline < NOW()); `GetTimedOutSessions(ctx) []ExamSession` (JOIN actividades: status='active' AND a.deadline > NOW() AND last_heartbeat_at < NOW()-INTERVAL '5 minutes'); `GetActiveSessionByID(ctx, sessionID uuid.UUID) *ExamSession`
- [X] T019 [P] [US3] Implement `CloseSessionWithSubmission(ctx, studentID, actividadID, submissionID uuid.UUID)` in `nextlab-api/internal/tracing/service.go`: (1) repo.GetSessionByStudentActivity — if nil, return (no session, silent no-op); (2) repo.CloseSession(ctx, session.ID, &submissionID, "completed"); (3) svc.AppendEvent(ctx, session.ID, "session_close", nil); (4) jobID = repo.InsertWorkerJob(ctx, session.ID, submissionID); (5) XADD to "nextlab:sessions:jobs" with fields {job_id, session_id, submission_id}; (6) on XADD error: slog.ErrorContext + return (job row stays 'pending', worker sweep re-enqueues); (7) repo.SetJobStreamID(ctx, jobID, streamID); also implement `EnqueueJobForSession(ctx, sessionID, submissionID uuid.UUID)` helper reusable by the worker sweep
- [X] T020 [US3] Extend `nextlab-api/internal/submission/service.go` and `nextlab-api/internal/submission/module.go`: add `tracingSvc TracingService` interface field to `submission.Service` (define `TracingService` interface with `CloseSessionWithSubmission(ctx, studentID, actividadID, submissionID uuid.UUID)` so submission does not directly import tracing); in `UploadFile` — after COMMIT, if `act.TraceEnabled`, call `s.tracingSvc.CloseSessionWithSubmission(ctx, studentID, actividadID, sub.ID)`; in `ReplaceFile` — same check after successful UpdateSubmission TX; update `submission.RegisterRoutes` signature to accept `tracingSvc TracingService`; update `nextlab-api/cmd/api/main.go` to pass `tracingSvc` (the *tracing.Service created for tracing.RegisterRoutes) to submission.RegisterRoutes
- [X] T021 [US3] Expand `nextlab-api/cmd/worker/main.go`: add `pgxpool.New(ctx, cfg.DatabaseURL)` connection with os.Exit(1) on failure; import `internal/tracing`; instantiate `tracingRepo := tracing.NewRepository(pgPool, rdb)` and `tracingSvc := tracing.NewService(tracingRepo, rdb)`; call `runConsumerLoop(ctx, rdb, tracingRepo, tracingSvc)`; implement `runConsumerLoop`: call `rdb.XGroupCreateMkStream` (idempotent); call `claimOrphanedMessages(ctx, rdb, tracingRepo)` on startup; start `go runDeadlineSweep(ctx, tracingRepo, tracingSvc, rdb)`; enter `rdb.XReadGroup` loop (Group:"analysis-workers", Consumer:"worker-"+hostname, Block:5s, Count:1); call `processJob` for each message
- [X] T022 [US3] Implement `processJob`, `claimOrphanedMessages`, `runDeadlineSweep` in `nextlab-api/cmd/worker/main.go`: `processJob(ctx, rdb, repo, msg)`: parse job_id + session_id + submission_id; `repo.MarkJobStatus(processing)`; `repo.MarkJobStatus(done)`; `rdb.XAck`; `slog.InfoContext("job acknowledged")`; `claimOrphanedMessages`: `rdb.XPending` to find idle>30s messages; `rdb.XClaim` each; for claimed: re-run processJob; `runDeadlineSweep`: ticker 60s — (a) `repo.GetDeadlinedSessions()`: for each, close as 'deadline_closed' or 'completed' if submission linked + AppendEvent + EnqueueJobForSession if submission; (b) `repo.GetTimedOutSessions()`: for each, close as 'timed_out' + AppendEvent; also sweep orphaned worker_jobs: `repo.GetOrphanedJobs()` → re-XADD each

**Checkpoint**: US3 fully testable. Exam file submission → 201 within 2s; session=completed; worker_jobs created; worker ack within 3 min; stop worker mid-job → restart recovers; deadline sweep closes expired sessions.

---

## Phase 6: User Story 4 — Docente Reviews Trace Evidence (Priority: P4)

**Goal**: A docente can retrieve the full immutable trace for a student submission and verify the SHA-256 hash chain is intact. Access is restricted to the owning docente or co-docente with an active grant.

**Independent Test**: After a session with N events, docente GET /submissions/:sid/trace → N events in sequence order; GET /submissions/:sid/trace/verify → {valid: true}; manually tamper one hash → {valid: false, first_invalid_sequence: N}; wrong-course docente → 403.

### Implementation

- [X] T023 [P] [US4] Implement trace retrieval queries in `nextlab-api/internal/tracing/repository.go`: `GetTraceBySubmission(ctx, submissionID, callerID uuid.UUID, role string) (*ExamSession, []TraceEvent, error)` — for role="docente": JOIN submissions → exam_sessions → actividades → cursos WHERE c.docente_id=$callerID; for role="co_docente": JOIN codocente_grants WHERE g.codocente_id=$callerID AND g.is_active=true AND g.actividad_id=es.actividad_id; fetch all trace_events ORDER BY sequence_num ASC; return ErrForbidden if no rows matched; `GetAllEvents(ctx, sessionID uuid.UUID) []TraceEvent` (ORDER BY sequence_num ASC — used by VerifyChain)
- [X] T024 [P] [US4] Implement `VerifyChain(ctx context.Context, sessionID uuid.UUID) (valid bool, eventCount int, firstInvalidSeq int)` in `nextlab-api/internal/tracing/service.go`: fetch events via repo.GetAllEvents; set prevHash = hex(SHA256(sessionID.String())); for each event: recompute event_hash using computeEventHash(sessionID, seq, type, receivedAt.UnixNano(), canonicalJSON(metadata), prevHash); if computed != stored → return (false, len(events), e.SequenceNum); return (true, len(events), 0) on success
- [X] T025 [US4] Implement `GET /submissions/:sid/trace` and `GET /submissions/:sid/trace/verify` handlers in `nextlab-api/internal/tracing/handler_docente.go`: GET trace: parse sid UUID; read callerID from c.Locals("user_id") and role from c.Locals("user_role"); call repo.GetTraceBySubmission; map ErrForbidden→403, not-found→404; return TraceResponse; GET verify: call repo.GetTraceBySubmission for role-check then svc.VerifyChain; return VerifyResponse with first_invalid_sequence omitted (null) when valid=true
- [X] T026 [US4] Wire `GET /submissions/:sid/trace` and `GET /submissions/:sid/trace/verify` in `nextlab-api/internal/tracing/module.go` with `authMW` + `roleGuard("docente", "co_docente")`

**Checkpoint**: US4 fully testable. Docente gets all events in sequence order; verify returns valid=true; tampered hash → first_invalid_sequence set; co-docente with grant → 200; co-docente without grant → 403; wrong-course docente → 403.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end validation, error handling audit, and log completeness.

- [ ] T027 Run full `quickstart.md` validation steps 1–13: session idempotency, tarea rejection, WS connect + events, rate limit, submission close (2s SLA), worker ack (3min SLA), docente trace retrieval, hash verify, tamper test, cross-course 403, crash recovery (attempt_count incremented, no duplicate job)
- [X] T028 Audit error responses and logging: verify WS invalid/missing JWT returns HTTP 401 before upgrade (not a WS-level error); verify events sent to closed session return `{"type":"error","code":"session_closed"}` frame without closing connection; verify all docente 403s return `{"error":"forbidden"}`; verify worker log lines include `job_id` and `session_id` fields; verify token value never appears in any log line (grep worker + api logs for the test token)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (go.mod updated) — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Phase 2 — no dependency on US2/US3/US4
- **US2 (Phase 4)**: Depends on US1 (WS handler validates session belongs to student; needs session to exist)
- **US3 (Phase 5)**: Depends on US1 (needs active session to close) and US2 (calls AppendEvent for session_close)
- **US4 (Phase 6)**: Depends on US3 (needs completed sessions with jobs); independent of US2 WS code
- **Polish (Phase 7)**: Depends on all stories complete

### User Story Dependencies

- **US1 (P1)**: Only Foundational — MVP deliverable
- **US2 (P2)**: Depends on US1 (session must exist before WS can attach)
- **US3 (P3)**: Depends on US1 + US2 (AppendEvent for session_close; submission extension)
- **US4 (P4)**: Depends on US3 (completed sessions needed for trace retrieval)

### Within Each Phase

- `[P]`-marked tasks within a phase touch different files and can run in parallel
- Repository tasks precede service tasks; service tasks precede handler tasks
- Module wiring is always last within a phase

### Parallel Opportunities

- T002 + T003 (up/down migration — different files)
- T003, T004, T005, T006, T007 (all Foundational [P] tasks — different files)
- T008 (repo) parallel with T009 start blocked (service needs repo complete)
- T013 (hub.go) + T014 (repo AppendEvent) — parallel in US2
- T018 (repo worker queries) + T019 (service) — parallel in US3 (different concerns)
- T023 (repo trace queries) + T024 (service VerifyChain) — parallel in US4

---

## Parallel Example: User Story 2

```bash
# Can run in parallel (different files):
Task T013: "Create hub.go in internal/tracing/hub.go"
Task T014: "Implement AppendEvent query in internal/tracing/repository.go"

# Then sequentially (service depends on both):
Task T015: "Implement AppendEvent in service.go (uses computeEventHash + repo)"
Task T016: "Implement WebSocket handler in handler_ws.go (uses service + hub)"
Task T017: "Wire WebSocket route in module.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational — **CRITICAL** (T002–T007)
3. Complete Phase 3: US1 (T008–T012)
4. **STOP and VALIDATE**: POST session → 201; repeat → 200; tarea → 422
5. Demo session creation before proceeding to real-time events

### Incremental Delivery

1. Phase 1–2 → Go module + DB tables ready
2. Phase 3 → Students can open sessions → **MVP**
3. Phase 4 → Real-time event capture via WebSocket
4. Phase 5 → File submission closes session; worker acknowledges
5. Phase 6 → Docentes can review and verify traces
6. Phase 7 → End-to-end validation → M5 milestone deliverable

---

## Notes

- `[P]` = can run in parallel with other `[P]` tasks in the same phase (different files)
- `[USn]` = maps task to specific user story for traceability
- The `TracingService` interface in `submission/service.go` (T020) prevents a direct import cycle: `submission` → interface; `main.go` wires the concrete `*tracing.Service`
- Worker's `processJob` in M5 writes acknowledgement only; M8 adds heuristics, M9 adds format validation — the processJob function is the extension point
- `trace_events` has NO update or delete operations anywhere in repository.go — any attempt is a constitution violation
- Token value from WS query param must never be logged; verify this in T028
- `canonicalJSON` must produce identical output for logically equivalent metadata objects — map key sorting is mandatory for hash chain reproducibility
