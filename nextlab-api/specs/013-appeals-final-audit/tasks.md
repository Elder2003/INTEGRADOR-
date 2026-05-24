# Tasks: M13 — Appeals & Final Audit

**Input**: Design documents from `specs/013-appeals-final-audit/`

**Prerequisites**: spec.md ✓, research.md ✓, data-model.md ✓, plan.md ✓, contracts/appeals.md ✓

---

## Phase 1: Setup

- [X] T001 Write `migrations/000013_appeals_final_audit.up.sql`: `CREATE TABLE appeals (id UUID PK DEFAULT gen_random_uuid(), session_id UUID NOT NULL REFERENCES exam_sessions(id), filed_by UUID NOT NULL REFERENCES users(id), type TEXT NOT NULL CHECK (type IN ('analysis_dispute','grade_dispute')), status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','escalated','resolved','admin_resolved')), reason TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(session_id)); CREATE INDEX appeals_filed_by_idx ON appeals(filed_by); CREATE INDEX appeals_status_idx ON appeals(status); CREATE TABLE appeal_decisions (id UUID PK DEFAULT gen_random_uuid(), appeal_id UUID NOT NULL REFERENCES appeals(id), decided_by UUID NOT NULL REFERENCES users(id), role TEXT NOT NULL CHECK (role IN ('docente','admin')), decision TEXT NOT NULL CHECK (decision IN ('upheld','rejected','overridden')), justification TEXT NOT NULL, decided_at TIMESTAMPTZ NOT NULL DEFAULT now()); CREATE INDEX appeal_decisions_appeal_idx ON appeal_decisions(appeal_id); CREATE TABLE audit_events (id UUID PK DEFAULT gen_random_uuid(), appeal_id UUID NOT NULL REFERENCES appeals(id), sequence INT NOT NULL, event_type TEXT NOT NULL, payload JSONB NOT NULL DEFAULT '{}', prev_hash TEXT, event_hash TEXT NOT NULL, occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(), UNIQUE(appeal_id, sequence)); CREATE INDEX audit_events_appeal_idx ON audit_events(appeal_id, sequence)`
- [X] T002 Write `migrations/000013_appeals_final_audit.down.sql`: `DROP TABLE IF EXISTS audit_events; DROP TABLE IF EXISTS appeal_decisions; DROP TABLE IF EXISTS appeals`

---

## Phase 2: Foundational

- [X] T003 [P] Write `internal/appeals/model.go`: `Appeal` {id, session_id, filed_by, type, status, reason, created_at}; `AppealDecision` {id, appeal_id, decided_by, role, decision, justification, decided_at}; `AuditEvent` {id, appeal_id, sequence int, event_type, payload map[string]any, prev_hash *string, event_hash string, occurred_at}; `FileAppealRequest` {session_id uuid.UUID, type string, reason string}; `ReviewRequest` {decision string, justification string}; `EscalateRequest` {notes string}; `EvidenceChain` {appeal_id uuid.UUID, valid bool, events []AuditEvent}; `AppealFilter` {filed_by *uuid.UUID, session_id *uuid.UUID, actividad_id *uuid.UUID, status string}; `ErrorResponse` {error, message}
- [X] T004 Write `internal/appeals/repository.go`: `Repository` struct {db *pgxpool.Pool}; `NewRepository(db *pgxpool.Pool) *Repository`; `InsertAppeal(ctx, req FileAppealRequest, filedBy uuid.UUID) (*Appeal, error)` — begins tx; INSERT INTO appeals; SELECT session context from exam_sessions JOIN submissions (submission_id) JOIN analysis_results (session_id, analysis_type='heuristic') LEFT JOIN analysis_results llm (session_id, analysis_type='llm'); build and insert 3 audit events (submission_received, analysis_completed, appeal_filed) chaining hashes using `computeHash`; commit; `GetAppeal(ctx, id uuid.UUID) (*Appeal, error)` — nil on no rows; `HasOpenAppeal(ctx, sessionID uuid.UUID) (bool, error)` — SELECT EXISTS WHERE session_id=$1 AND status IN ('pending','escalated'); `ListAppeals(ctx, filter AppealFilter) ([]Appeal, int, error)` — dynamic WHERE by filed_by/actividad_id (JOIN exam_sessions→actividades→cursos)/status with COUNT; `UpdateAppealStatus(ctx, id uuid.UUID, status string) error`; `InsertDecision(ctx, appealID, decidedBy uuid.UUID, role, decision, justification string) (*AppealDecision, error)` — begins tx; INSERT INTO appeal_decisions; SELECT last audit_event hash for this appeal; INSERT audit_event(decision_recorded, seq=N+1) with prev_hash; commit; `GetDecisions(ctx, appealID uuid.UUID) ([]AppealDecision, error)`; `GetAuditEvents(ctx, appealID uuid.UUID) ([]AuditEvent, error)` — ORDER BY sequence ASC; `VerifyStudentOwnsSession(ctx, sessionID, studentID uuid.UUID) (bool, error)` — SELECT EXISTS FROM exam_sessions WHERE id=$1 AND student_id=$2; `VerifyDocenteOwnsAppealSession(ctx, appealID, docenteID uuid.UUID) (bool, error)` — JOIN appeals→exam_sessions→actividades→cursos WHERE docente_id=$2; `computeHash(occurredAt time.Time, eventType string, sequence int, payloadJSON string, prevHash string) string` — returns hex(sha256(pipe-delimited: occurred_at ISO8601|event_type|sequence|payload_json|prev_hash))
- [X] T005 Write `internal/appeals/module.go`: `Module` struct {repo *Repository}; `NewModule(db *pgxpool.Pool) *Module`; stub `RegisterRoutes(app *fiber.App, jwtMW fiber.Handler, roleGuard func(...string) fiber.Handler)` returning nil for all routes

---

## Phase 3: US1 — Student Appeal Filing (P1) 🎯 MVP

**Goal**: Students file appeals; docentes see them in their queue.

**Independent Test**: POST /appeals with valid session_id → 201; GET /appeals/:id → returns record; duplicate → 409; GET /actividades/:aid/appeals → docente sees the appeal; non-owner → 403.

- [X] T006 [P] [US1] Write `internal/appeals/handler_file.go`: `FileAppealHandler(repo *Repository) fiber.Handler` — parse callerID; parse FileAppealRequest from body; validate type is analysis_dispute or grade_dispute; validate reason non-empty; VerifyStudentOwnsSession → 403; HasOpenAppeal → 409; InsertAppeal → 201 Appeal; map unique violation → 409; `GetAppealHandler(repo *Repository) fiber.Handler` — parse id; GetAppeal; check caller is filed_by, docente of session, or admin; 403 or 404 as appropriate; return 200; `ListAppealsHandler(repo *Repository) fiber.Handler` — parse role from locals; if student → filter filed_by=callerID; if docente → filter actividad scoped to docente's sessions; if admin → all; parse ?status ?session_id query params; return {appeals, total}; `ListActivityAppealsHandler(repo *Repository) fiber.Handler` — parse :aid; verify caller is docente of actividad or admin; filter appeals by actividad_id; return {actividad_id, appeals, total}
- [X] T007 [US1] Wire US1 routes in `internal/appeals/module.go` `RegisterRoutes`: `app.Post("/appeals", jwtMW, FileAppealHandler(m.repo))`; `app.Get("/appeals/:id", jwtMW, GetAppealHandler(m.repo))`; `app.Get("/appeals", jwtMW, ListAppealsHandler(m.repo))`; `app.Get("/actividades/:aid/appeals", jwtMW, ListActivityAppealsHandler(m.repo))`

**Checkpoint**: Student files appeal; docente sees it in queue; duplicate blocked; non-owner gets 403.

---

## Phase 4: US2 — Grade Log Endpoint (P2)

**Goal**: Docente retrieves the grade decision log for a session (reads existing grade_decisions table).

**Independent Test**: GET /sessions/:sessionId/grade-log → returns grade+suggested_range+override_reason; 404 if no grade yet; non-owner docente → 403.

- [X] T008 [P] [US2] Add `GradeLogResult` struct to `internal/detection/model.go`: {SessionID uuid.UUID, FinalGrade float64, SuggestedGradeRange string, OverrideReason *string, DecidedBy uuid.UUID, DecidedAt time.Time}; add `gradeLog` method to `internal/detection/handler.go`: parse sessionID; verify VerifyDocenteOwnsSession → 403; GetGradeDecision → 404 if nil; map to GradeLogResult; return 200
- [X] T009 [US2] Register `sessions.Get("/grade-log", h.gradeLog)` in `internal/detection/module.go` RegisterRoutes after existing sessions group routes

**Checkpoint**: Docente can retrieve the grade decision with override_reason for any session they own.

---

## Phase 5: US3 — Docente Review & Admin Escalation (P3)

**Goal**: Docente reviews pending appeals; auto-escalates grade_dispute on own session; admin makes final decisions.

**Independent Test**: PUT /appeals/:id/review with justification → resolved; grade_dispute on own session → auto-escalated; PUT /appeals/:id/escalate → escalated; PUT /appeals/:id/admin-review → admin_resolved; re-review on resolved → 409.

- [X] T010 [P] [US3] Write `internal/appeals/handler_review.go`: `ReviewAppealHandler(repo *Repository) fiber.Handler` — parse id; GetAppeal → 404; check appeal.status == "pending" else 409 "already_resolved"; VerifyDocenteOwnsAppealSession → 403; parse ReviewRequest; if req.Decision not in {upheld,rejected} → 400; if appeal.Type=="grade_dispute" AND docente owns the session → auto-escalate: UpdateAppealStatus(escalated), return 200 {appeal}; else InsertDecision(role="docente",...), UpdateAppealStatus("resolved"), return 200 {appeal, decision}; `EscalateAppealHandler(repo *Repository) fiber.Handler` — parse id; GetAppeal → 404; check status=="pending" else 409; VerifyDocenteOwnsAppealSession → 403; parse EscalateRequest (optional notes); UpdateAppealStatus("escalated"); return 200 updated Appeal; `AdminReviewAppealHandler(repo *Repository) fiber.Handler` — parse id; GetAppeal → 404; check status=="escalated" else 409; verify caller role=="admin" else 403; parse ReviewRequest; InsertDecision(role="admin",...); UpdateAppealStatus("admin_resolved"); return 200 {appeal, decision}
- [X] T011 [US3] Wire US3 routes in `internal/appeals/module.go` `RegisterRoutes`: `app.Put("/appeals/:id/review", jwtMW, ReviewAppealHandler(m.repo))`; `app.Put("/appeals/:id/escalate", jwtMW, EscalateAppealHandler(m.repo))`; `app.Put("/appeals/:id/admin-review", jwtMW, roleGuard("admin"), AdminReviewAppealHandler(m.repo))`

**Checkpoint**: Docente rejects analysis_dispute; grade_dispute auto-escalates; admin resolves escalated appeal; resolved appeal cannot be re-reviewed.

---

## Phase 6: US4 — Immutable Evidence Audit Endpoint (P4)

**Goal**: GET /appeals/:id/evidence returns time-ordered hash-chained events; all hashes independently verifiable.

**Independent Test**: GET /appeals/:id/evidence → 4 events; prev_hash[0]==null; re-computing SHA-256(occurred_at|event_type|sequence|payload_json|prev_hash) matches event_hash for every event; admin and docente get 200; student gets 403.

- [X] T012 [P] [US4] Write `internal/appeals/handler_evidence.go`: `verifyChain(events []AuditEvent) bool` — recomputes event_hash for each event using `computeHash` (reuse from repository via helper); returns true if all match; `GetEvidenceHandler(repo *Repository) fiber.Handler` — parse id; GetAppeal → 404; verify caller is admin or VerifyDocenteOwnsAppealSession → 403; GetAuditEvents(appealID); call verifyChain; return 200 EvidenceChain{appeal_id, valid, events}; note: to use computeHash from repository, extract it to `internal/appeals/hash.go` as a package-level function `ComputeHash(occurredAt, eventType string, sequence int, payloadJSON, prevHash string) string`; move/refactor computeHash out of repository.go at same time
- [X] T013 [US4] Wire `GET /appeals/:id/evidence` in `internal/appeals/module.go` `RegisterRoutes`: `app.Get("/appeals/:id/evidence", jwtMW, GetEvidenceHandler(m.repo))`

**Checkpoint**: Evidence chain returns 4 events with valid=true; broken hash is detectable (valid=false on first mismatch).

---

## Phase 7: US5 — End-to-End Integration Tests (P5)

**Goal**: Real-DB integration tests covering exam session, submission+grade, and appeal lifecycle flows.

**Independent Test**: `go test ./tests/integration/...` -tags integration -v passes against DATABASE_URL_TEST; no DB mocks; all 3 flows covered.

- [X] T014 [P] [US5] Write `tests/integration/setup_test.go`: package integration_test; `TestMain(m *testing.M)` — connect to os.Getenv("DATABASE_URL_TEST") via pgxpool; if DATABASE_URL_TEST empty, skip all tests; truncate tables in order: audit_events, appeal_decisions, appeals, grade_decisions, analysis_results, trace_events, exam_sessions, submissions, format_reports, format_templates, actividad_format_template, actividad_interdisciplinary_scopes, actividades, grupos, cursos, users; run m.Run(); helper `seedUser(t, db, role) uuid.UUID`; helper `seedActividad(t, db, docenteID) uuid.UUID`; helper `seedExamSession(t, db, studentID, actividadID) uuid.UUID`; helper `seedAnalysis(t, db, sessionID, score int, suggestedRange string)`
- [X] T015 [P] [US5] Write `tests/integration/exam_session_test.go`: `TestExamSessionFlow(t *testing.T)` — seed admin, docente, curso, actividad, student; insert exam_session row; insert 3 trace_events with event_hash; update exam_session status='closed'; call seedAnalysis; assert analysis_results has heuristic row with session_id; all assertions use real DB queries, no HTTP calls, no mocks
- [X] T016 [P] [US5] Write `tests/integration/submission_test.go`: `TestSubmissionGradeFlow(t *testing.T)` — seed from T015 setup; insert submission row; insert grade_decision with final_grade=80, suggested_grade_range="75-85", override_reason=null; assert grade_decisions row exists; insert second grade_decision attempt → expect pgconn.PgError with UNIQUE violation (grade_decisions UNIQUE session_id assumed or test verifies idempotency); test override_reason: insert grade_decision with final_grade=95, suggested_grade_range="75-85", override_reason="demonstrated exceptional understanding" → assert override_reason stored
- [X] T017 [US5] Write `tests/integration/appeal_test.go`: `TestAppealFileAndReject(t *testing.T)` — seed exam session + analysis; call repo.InsertAppeal directly with type=analysis_dispute; assert appeal.status="pending"; assert 3 audit_events exist (submission_received, analysis_completed, appeal_filed); call repo.InsertDecision(role=docente, decision=rejected, justification="..."); assert appeal status=resolved; assert 4th audit_event is decision_recorded; call repo.GetAuditEvents; call verifyChain helper; assert valid=true; `TestAppealEscalateAndAdminResolve(t *testing.T)` — similar seed; InsertAppeal(grade_dispute); UpdateAppealStatus(escalated); InsertDecision(role=admin, decision=upheld, justification="..."); assert admin_resolved; assert 4 events; `TestEvidenceChainTampering(t *testing.T)` — file appeal, verify chain; manually UPDATE audit_events SET payload='{}' WHERE sequence=2; call verifyChain; assert valid=false

**Checkpoint**: `go test ./tests/integration/... -tags integration` passes all 3 test functions; DATABASE_URL_TEST required; suite completes in under 5 minutes.

---

## Phase 8: Polish

- [X] T018 Register appeals module in `cmd/api/main.go`: add import `"github.com/Elder2003/nextlab-api/internal/appeals"`; after adminModule registration: `appealsModule := appeals.NewModule(pgPool)`; `appealsModule.RegisterRoutes(app, auth.AuthMiddleware(cfg), auth.RoleGuard)`
- [X] T019 Run `go build ./...` and `go vet ./...` from repo root; fix any compile errors or vet warnings
- [ ] T020 Run quickstart scenarios from `specs/013-appeals-final-audit/quickstart.md` against live stack; document any gaps

---

## Dependencies & Execution Order

- Phase 1 (setup) → blocks all others
- Phase 2 (foundational) → T003 [P] runs with T004; T005 depends on T004; blocks all user story phases
- US1 (P1) → T006 [P] runs with T004 in parallel; T007 depends on T006; **MVP stop point**
- US2 (P2) → T008 [P] is independent of US1 (different package); T009 depends on T008
- US3 (P3) → T010 [P] depends on repo (T004); T011 depends on T010; requires US1 for appeal to exist
- US4 (P4) → T012 [P] depends on T004; must extract `ComputeHash` helper first; T013 depends on T012
- US5 (P5) → T014/T015/T016 [P] can run together after DB access confirmed; T017 depends on T014 + repo complete (T004)
- Phase 8 (polish) → depends on all user stories

## Notes

- T012 requires extracting `computeHash` from repository.go into `internal/appeals/hash.go` so it can be called from handler_evidence.go without circular reference — do this during T004 and note in T012
- T015 and T016 are unit-style integration tests (direct repo calls); T017 tests the full chain using repo methods
- `grade_decisions` table already exists (M8 migration) — T016 reads from it but does not migrate it
- US2 (T008/T009) adds a new handler method to the existing detection package; no new package created
- Integration tests require `//go:build integration` tag or `DATABASE_URL_TEST` guard in TestMain
