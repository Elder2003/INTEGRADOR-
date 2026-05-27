# Feature Specification: Appeals & Final Audit

**Feature Branch**: `013-appeals-final-audit`

**Created**: 2026-05-23

**Status**: Draft

**Input**: M13 — Appeals & Final Audit: Immutable chain; end-to-end tests (Principles I, III)

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Appeal on Analysis Result (Priority: P1)

A student who disagrees with an AI-generated plagiarism or format detection result can file a formal appeal through the platform. The student states their reason; the appeal is recorded in an immutable log and routed to the docente for initial review. Once submitted, the appeal record cannot be altered.

**Why this priority**: This is the core justification for the entire audit system. Without an appeal mechanism, the AI analysis output has no institutional recourse, which undermines Principle III (Human Authority Over AI). All other M13 stories depend on the appeal record existing.

**Independent Test**: Authenticated student `POST /appeals` with `submission_id` and reason → 201; `GET /appeals/:id` returns the record; student cannot PATCH/DELETE the appeal; docente sees the appeal in their queue.

**Acceptance Scenarios**:

1. **Given** a student has a graded or analysed submission, **When** `POST /appeals` with `submission_id`, `type` (analysis_dispute or grade_dispute), and `reason`, **Then** appeal is created with status=pending, returning 201 with the appeal record.
2. **Given** an appeal exists, **When** the student or docente `GET /appeals/:id`, **Then** the full appeal record is returned with all fields.
3. **Given** an appeal exists, **When** any user attempts to modify or delete it, **Then** 405 Method Not Allowed — the record is immutable after creation.
4. **Given** a student is authenticated, **When** `GET /appeals?submission_id=X`, **Then** only appeals belonging to that student are returned.
5. **Given** a docente is authenticated, **When** `GET /actividades/:aid/appeals`, **Then** all pending appeals for activities they own are returned.
6. **Given** a student submits a duplicate appeal for the same submission, **When** an open appeal already exists for that submission, **Then** 409 Conflict.

---

### User Story 2 — Grade Justification on AI Divergence (Priority: P2)

When a docente assigns a final grade that differs from the AI-suggested grade on a submission, the system requires a written justification before the grade is saved. This divergence log is immutable and forms part of the audit trail.

**Why this priority**: This directly implements Principle III (Human Authority Over AI) and RF-18. Without grade divergence logging, the system cannot demonstrate that human judgment is recorded when overriding AI recommendations.

**Independent Test**: Docente `POST /submissions/:id/grade` with `final_grade` different from `suggested_grade` → 422 if `reason` is absent; → 200 with divergence log created if `reason` present; `GET /submissions/:id/grade-log` returns the log.

**Acceptance Scenarios**:

1. **Given** a submission with an AI-suggested grade, **When** docente `POST /submissions/:id/grade` with `final_grade` equal to `suggested_grade`, **Then** grade is saved; no divergence log required; 200 returned.
2. **Given** a submission with an AI-suggested grade, **When** docente `POST /submissions/:id/grade` with `final_grade` different from `suggested_grade` and no `reason`, **Then** 422 Unprocessable Entity with error indicating reason is required.
3. **Given** a submission with an AI-suggested grade, **When** docente `POST /submissions/:id/grade` with `final_grade` different from `suggested_grade` and a `reason`, **Then** grade and divergence log are saved atomically; 200 returned.
4. **Given** a grade and divergence log exist, **When** any user attempts to update or delete the divergence log, **Then** 405 — the log is immutable.
5. **Given** a docente accesses a graded submission, **When** `GET /submissions/:id/grade-log`, **Then** the final grade, suggested grade, reason (if divergent), and timestamp are returned.

---

### User Story 3 — Docente Appeal Review & Admin Escalation (Priority: P3)

A docente reviews a pending appeal and either upholds or rejects it. If the docente cannot resolve (e.g., conflict of interest, escalation threshold), the appeal is escalated to the admin. The admin's decision is final and immutable. All decisions are logged.

**Why this priority**: Required to close the appeal lifecycle. An appeal with no resolution path is unusable institutionally. Depends on US1 (appeal creation).

**Independent Test**: Docente `PUT /appeals/:id/review` with decision=rejected and justification → appeal status=resolved; `PUT /appeals/:id/escalate` → status=escalated; admin `PUT /appeals/:id/admin-review` → status=resolved with admin decision; all decisions append-only (no re-review on resolved appeals).

**Acceptance Scenarios**:

1. **Given** a pending appeal, **When** docente `PUT /appeals/:id/review` with `decision` (upheld or rejected) and `justification`, **Then** appeal status changes to resolved; decision is recorded with timestamp; 200 returned.
2. **Given** a pending appeal, **When** docente `PUT /appeals/:id/escalate` with optional notes, **Then** appeal status changes to escalated; admin receives it in their queue; 200 returned.
3. **Given** a resolved appeal, **When** any user attempts another review decision, **Then** 409 — appeal already resolved, no re-review permitted.
4. **Given** an escalated appeal, **When** admin `PUT /appeals/:id/admin-review` with `decision` and `justification`, **Then** appeal status changes to admin_resolved; decision is recorded; 200 returned.
5. **Given** admin is authenticated, **When** `GET /appeals?status=escalated`, **Then** all escalated appeals across all activities are returned.
6. **Given** a non-owner docente or student, **When** they attempt to review/escalate an appeal, **Then** 403.

---

### User Story 4 — Immutable Evidence Audit Endpoint (Priority: P4)

An admin or docente can retrieve the full, tamper-verifiable chain of evidence for an appeal — including trace events, submission metadata, analysis results, and appeal decisions — with hash verification to prove chain integrity.

**Why this priority**: This is the deliverable that makes M13 legally and institutionally defensible (Principle I). Without the audit endpoint, the appeals system has no external verification mechanism.

**Independent Test**: `GET /appeals/:id/evidence` returns ordered event chain; each event includes a `prev_hash` and `event_hash` (SHA-256); first event has `prev_hash=null`; re-computing hashes on the client matches; inserting a fake event breaks chain integrity.

**Acceptance Scenarios**:

1. **Given** an appeal with a resolved decision, **When** admin `GET /appeals/:id/evidence`, **Then** a time-ordered list of all events is returned (submission received, analysis completed, appeal filed, decision recorded), each with `event_hash` and `prev_hash`.
2. **Given** the event chain, **When** any client computes SHA-256 of the pipe-delimited string `occurred_at|event_type|sequence|payload_json|prev_hash` (empty string for null prev_hash) for each event, **Then** the computed lowercase hex hash matches `event_hash` for every event in the chain.
3. **Given** an appeal's evidence chain, **When** admin or owning docente accesses `GET /appeals/:id/evidence`, **Then** 200 returned; non-owner docente, co-docente, or student → 403.
4. **Given** a tampered trace event in the database (simulated), **When** the audit endpoint is called, **Then** the broken hash link is visible in the response (hash mismatch on that event), enabling detection without trusting the application.

---

### User Story 5 — End-to-End Integration Tests (Priority: P5)

The system includes automated integration tests that exercise the three critical flows end-to-end: (1) exam session capture and analysis, (2) submission and format report, (3) appeal lifecycle to resolution. Tests run against a real in-process test database and cover the entire request path.

**Why this priority**: The constitution mandates integration tests by M13. These tests protect all prior milestones from regression and provide confidence for production deployment.

**Independent Test**: `go test ./tests/integration/...` passes with a real PostgreSQL test database; tests cover the exam session flow, submission + detection flow, and appeal flow; no mocks on the database layer.

**Acceptance Scenarios**:

1. **Given** a test database is running, **When** integration tests execute, **Then** all tests pass with no database mocks — real DB queries only.
2. **Given** the exam session integration test, **When** it runs, **Then** it covers: student opens session → trace events captured → session closed → analysis result stored.
3. **Given** the submission integration test, **When** it runs, **Then** it covers: student uploads file → format report generated → detection result stored → docente grades.
4. **Given** the appeal integration test, **When** it runs, **Then** it covers: student files appeal → docente reviews → admin escalates → admin resolves → evidence chain verified.
5. **Given** the integration test suite, **When** any individual test fails, **Then** the failure points to a specific flow step with a clear error message.

---

### Edge Cases

- Student files appeal after grading period closed: the appeal is accepted (Principle VII — submission always accepted applies to appeals too); docente sees a flag that the activity period has passed.
- Docente reviews their own grade divergence appeal: the docente is the respondent; they cannot self-review — appeal is automatically escalated to admin.
- Admin reviews an appeal for an activity they don't own: admin has global scope (Principle IV allows admin global scope).
- Grade divergence logging when no AI-suggested grade exists (e.g., manual grading only): divergence check is skipped; grade saves without requiring a reason.
- Appeal filed on a submission that has already been appealed and resolved: 409 Conflict — only one appeal per submission is permitted at a time.
- Integration test isolation: tests run against a dedicated `DATABASE_URL_TEST` database that is truncated (all relevant tables) before each suite run. Individual tests do not roll back transactions, allowing the code under test to use its own transactions normally.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Students MUST be able to file an appeal on any of their analysed or graded submissions, specifying type (analysis_dispute or grade_dispute) and a written reason.
- **FR-002**: Appeal records MUST be immutable after creation — no UPDATE or DELETE is permitted on appeal rows or their decision logs.
- **FR-003**: When a docente assigns a final grade that differs from the AI-suggested grade, the system MUST require a written justification before the grade is persisted.
- **FR-004**: Grade divergence logs MUST be persisted atomically with the grade in the same transaction and MUST be immutable after creation.
- **FR-005**: Docentes MUST be able to review pending appeals on their activities (uphold or reject with justification), or escalate to admin.
- **FR-006**: Admins MUST be able to view all escalated appeals and issue final, immutable decisions.
- **FR-007**: `GET /appeals/:id/evidence` MUST return the time-ordered event chain for an appeal, each event containing `event_hash` (SHA-256 of canonical payload + prev_hash) and `prev_hash`.
- **FR-008**: The evidence hash chain MUST be independently verifiable — a client computing SHA-256 on each event must reproduce every `event_hash` in the chain.
- **FR-009**: All appeal and divergence endpoints MUST enforce role-based access: students see only their own; docentes see only their activities; admins have global scope.
- **FR-010**: A student MUST NOT be able to file a second open appeal on the same submission while a prior appeal is still pending or escalated.
- **FR-011**: Integration tests MUST cover the exam session, submission+detection, and appeal flows against a real PostgreSQL test database (no mocks on the DB layer).
- **FR-012**: All appeal decision records MUST store the deciding user's ID, decision, justification, and timestamp.

### Key Entities

- **Appeal**: id, submission_id, filed_by (student_id), type (analysis_dispute | grade_dispute), status (pending | escalated | resolved | admin_resolved), reason, created_at. Immutable after insert.
- **AppealDecision**: id, appeal_id, decided_by (user_id), role (docente | admin), decision (upheld | rejected | overridden), justification, decided_at. Append-only.
- **GradeDivergenceLog**: id, submission_id, docente_id, suggested_grade, final_grade, reason, recorded_at. Immutable after insert.
- **AuditEvent**: appeal_id, sequence, event_type, payload (JSONB), prev_hash, event_hash, occurred_at. Append-only.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A student can file an appeal in a single API call; the appeal record is visible to the docente immediately.
- **SC-002**: `GET /appeals/:id/evidence` returns the full evidence chain; every `event_hash` can be independently reproduced by the client using the provided `prev_hash` and canonical payload.
- **SC-003**: Docente grade with AI divergence: system rejects a grade-save without a reason in one response cycle (no multi-step interaction required).
- **SC-004**: All appeal and audit endpoints return 403 for roles without access within a single request (no additional calls required to enforce access control).
- **SC-005**: Integration test suite runs to completion (pass or fail) in under 5 minutes on a standard developer machine.
- **SC-006**: Zero data-layer mocks in the integration test suite — all DB interactions use a real PostgreSQL connection.

## Assumptions

- All prior milestones (M1–M12) are complete and stable; this spec does not redefine their interfaces.
- The `trace_events` and `analysis_results` tables created in M5/M8 already store the raw events that feed the evidence chain; M13 adds the hash layer on top.
- AI-suggested grades come from the `analysis_results` table (created in M8); the `suggested_grade` field is assumed to exist there or will be added in this milestone.
- A single appeal per submission at a time is sufficient for M13; multi-round appeals are out of scope.
- Integration tests use a separate test PostgreSQL database configured via `DATABASE_URL_TEST` env var. All relevant tables are truncated before each suite run; tests do NOT use per-test transaction rollback. Tests run within the existing `go test` framework under `tests/integration/`.
- The `appeals` table is owned by a new `internal/appeals` package to maintain package isolation (Principle IV).
- Hash algorithm for evidence chain: SHA-256. Canonical payload is the pipe-delimited concatenation of all AuditEvent fields in alphabetical key order: `occurred_at (ISO-8601) | event_type | sequence (decimal string) | payload_json | prev_hash` (empty string when null). `event_hash = SHA-256(canonical_payload)` as a lowercase hex string.
- Admin has global scope for all appeal operations (consistent with Principle IV and M12 admin role).

## Clarifications

### Session 2026-05-23

- Q: How should integration tests be isolated — per-test transaction rollback or a separate test DB? → A: Separate test DB (`DATABASE_URL_TEST`) truncated before each suite run. Per-test rollback is incompatible with application code that opens its own transactions (US2 atomic grade+log save, US3 review decisions).
- Q: What is the exact canonical payload format for AuditEvent hash computation? → A: SHA-256 of pipe-delimited `occurred_at|event_type|sequence|payload_json|prev_hash` (all as UTF-8 strings; prev_hash is empty string when null). Result is a lowercase hex string stored as `event_hash`.
- Q: Does the appeal state machine need an `under_review` state between pending and resolved/escalated? → A: No — `under_review` is removed. Appeals transition directly from `pending` to `resolved` (docente decision) or `escalated` (docente escalation), then to `admin_resolved` (admin decision). Four states total: pending, escalated, resolved, admin_resolved.
- Constitution §M13 states: "Appeals & Final Audit | Immutable chain; end-to-end tests | I, III". This spec implements that deliverable in full.
- "Immutable chain" covers both the existing trace-event immutability (Principle I) and the new hash-verification evidence endpoint.
- "End-to-end tests" means integration tests hitting a real DB — not unit tests with mocks.
- The formal appeal process is referenced in the constitution as RF-20; this spec implements RF-20 and RF-18 (grade divergence).
