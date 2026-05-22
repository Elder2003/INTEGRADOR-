# Feature Specification: M8 — Detection Engine

**Feature Branch**: `008-detection-engine`

**Created**: 2026-05-21

**Status**: Draft

**Input**: User description: "Lets do the milestone 8"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Worker Runs Heuristic Detection on a Submission (Priority: P1)

After a student submits an exam, the async worker picks up the queued job and runs behavioral heuristic analysis on the session's trace events. The heuristics examine patterns such as tab-switch frequency, clipboard activity count, timing anomalies between keystrokes and paste events, and session heartbeat gaps. The result — a heuristic score and a set of flagged event categories — is stored in the `analysis_results` table regardless of whether the subsequent LLM step succeeds.

**Why this priority**: Heuristic detection is the fail-safe layer of the detection pipeline. It runs entirely within the system without external dependencies, so it must always produce a stored result. If the LLM call fails, the heuristic result is the evidence the docente sees. It is the unconditional foundation of Principle II's partial-result requirement.

**Independent Test**: Can be fully tested by submitting an exam with a prepared trace (containing several tab-blur events and clipboard copies), waiting up to 3 minutes, and verifying a heuristic analysis record exists in the database with a non-null score and flagged categories — independent of any LLM call.

**Acceptance Scenarios**:

1. **Given** a student submits an exam file and the session closes, **When** the async worker processes the job, **Then** within 3 minutes a heuristic analysis record is stored with: a numeric suspicion score (0–100), the list of flagged event types, and the total event count analysed.
2. **Given** a session with zero suspicious events (no tab switches, no clipboard activity, no timing gaps), **When** heuristics run, **Then** the stored score is 0 and the flagged list is empty — no false positive.
3. **Given** a session with 15+ tab-blur events and 10+ clipboard-copy events, **When** heuristics run, **Then** the stored score is above the configured high-suspicion threshold and at least `tab_activity` and `clipboard_activity` appear in the flagged list.
4. **Given** the LLM analysis step fails after heuristics complete, **When** the job finishes, **Then** the heuristic result record already in the database is NOT deleted or rolled back — it remains as a standalone, usable result.

---

### User Story 2 — Worker Runs LLM Analysis with RAG Context (Priority: P2)

Following the heuristic step, the async worker builds a prompt that includes the student's submission content, the heuristic summary, and relevant context retrieved from both the curricular RAG (scoped to the subject) and the docente RAG (scoped to the activity). This enriched prompt is sent to the AI router (M6). The AI router returns an advisory analysis — a natural-language assessment with a suggested grade range and a justification. The result is stored in the `analysis_results` table alongside the heuristic record.

**Why this priority**: LLM analysis is the substantive, human-readable component of the detection report. It converts raw heuristic signals into an explanation the docente can act on. Without it, the docente sees only a score with no interpretation.

**Independent Test**: Can be fully tested by submitting an exam file with known content, waiting for the worker to complete, and verifying that an LLM analysis record exists with non-empty `justification`, a `suggested_grade_range`, and the `backend` label confirming which AI backend processed the request.

**Acceptance Scenarios**:

1. **Given** a submission has completed heuristic analysis, **When** the LLM step runs, **Then** the prompt sent to the router includes: the submission text, the heuristic score and flagged categories, and at least one chunk from the curricular RAG scoped to the subject.
2. **Given** an activity has an ingested rubric document, **When** the LLM prompt is built, **Then** the rubric is included in full — never truncated or replaced by a similarity fragment (Principle V).
3. **Given** the LLM returns a completion, **When** the result is stored, **Then** the record includes: `advisory_text` (the justification), `suggested_grade_range` (e.g., "60–70"), `backend_used` (from M6 router log), and `analysis_type = 'llm'`.
4. **Given** the LLM analysis record is stored, **When** a docente reads the report, **Then** the advisory text is clearly labelled as AI-generated and non-binding — no field in the response implies an automatic grade or penalty.

---

### User Story 3 — Docente Views the Detection Report (Priority: P3)

A docente opens a submission to review it. The system displays the detection report: the heuristic score with flagged event categories, the LLM advisory text with suggested grade range, and the backend attribution. The docente then assigns the final grade. If the assigned grade differs from the suggested range, the docente must provide a brief reason before saving.

**Why this priority**: The report is the primary deliverable of the entire detection pipeline. Without a readable report endpoint the pipeline produces results no one can see. The override-reason requirement (RF-18) is a constitution gate (Principle III) and must be enforced at this layer.

**Independent Test**: Can be fully tested by reading the detection report endpoint as a docente after a submission has been fully processed, confirming both the heuristic and LLM sub-reports are present, then assigning a grade within the suggested range (no reason required) and a grade outside the range (reason required and enforced).

**Acceptance Scenarios**:

1. **Given** a submission has been fully processed (heuristic + LLM stored), **When** a docente requests the detection report, **Then** the response contains both the heuristic sub-report (score, flagged categories, event count) and the LLM sub-report (advisory text, suggested grade range, backend label).
2. **Given** LLM analysis failed and only the heuristic result exists, **When** a docente requests the detection report, **Then** the response contains the heuristic sub-report with a clear `llm_status: "unavailable"` field — the heuristic result is shown, not an error page.
3. **Given** a docente assigns a grade within the LLM-suggested range, **When** they save, **Then** the grade is recorded without requiring a reason.
4. **Given** a docente assigns a grade outside the LLM-suggested range, **When** they try to save without a reason, **Then** the save is rejected with a validation error explaining that a reason is required when overriding the suggested range.
5. **Given** a docente assigns a grade outside the suggested range and provides a reason, **When** they save, **Then** the grade and reason are recorded in the same database transaction (RF-18) — the reason cannot be stored without the grade and vice versa.

---

### User Story 4 — Student Cannot Access the Detection Report (Priority: P4)

A student attempts to access any endpoint that exposes analysis results, heuristic scores, alert status, or flagged event categories for their own session. The system denies access regardless of authentication state.

**Why this priority**: This is a hard constitution constraint (Principle IV). A student who can read their own detection report can craft submissions to evade detection, invalidating the entire audit system.

**Independent Test**: Can be fully tested by authenticating as a student, then calling the detection report endpoint for a session that belongs to that student — the system must return 403 regardless of the student's enrollment.

**Acceptance Scenarios**:

1. **Given** a student is authenticated and has a completed submission, **When** they call the detection report endpoint for their own session, **Then** the response is `403 Forbidden` — no detection data is exposed.
2. **Given** a student calls the alert or analysis endpoints with any parameter, **When** the request is processed, **Then** the role middleware denies the request before any database query runs — no timing side-channel reveals whether data exists.

---

### Edge Cases

- What happens when the job is dequeued but the session record is missing (deleted externally)? → The worker logs a warning and drops the job; no partial record is written.
- What happens when the submission file cannot be read from MinIO? → The worker stores a heuristic-only result with `submission_text: null` and skips the LLM step; the detection report shows `content_unavailable: true`.
- What happens when the Redis Streams job is redelivered after a worker crash mid-analysis? → The worker checks whether an `analysis_results` record already exists for that session; if a heuristic record exists, it skips the heuristic step and proceeds directly to the LLM step (idempotent re-entry).
- What happens when the RAG retrieval returns an empty context? → The LLM step runs with only the submission text and heuristic summary — no error; the prompt notes that no curricular context was available.
- What happens when the heuristic step itself panics? → The worker recovers from the panic, logs the stack trace, marks the job as failed in the stream, and does NOT store a partial record.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The async worker MUST run heuristic analysis on trace events as the first step of every detection job — before any LLM call is attempted. Scoring weights (`HEURISTIC_TAB_WEIGHT`, `HEURISTIC_CLIP_WEIGHT`, `HEURISTIC_GAP_WEIGHT`) and the high-suspicion threshold (`HEURISTIC_HIGH_THRESHOLD`) MUST be read from environment variables with sensible defaults, not hardcoded — institutions differ in acceptable false-positive rates and must be able to recalibrate without a code change.
- **FR-002**: The heuristic result MUST be persisted to the `analysis_results` table as a **new row** (`analysis_type='heuristic'`) before the LLM call is initiated, so that a stored record exists regardless of LLM outcome (Principle II). Each session produces two rows maximum — one per `analysis_type`; LLM-specific columns are null on the heuristic row and vice versa.
- **FR-003**: The LLM prompt MUST include: the submission text (or a `content_unavailable` note), the heuristic score and flagged categories, curricular RAG context scoped to the subject, and docente RAG context scoped to the activity. RAG context is retrieved via direct Go calls to `rag.RetrieveService.QueryCurricular` and `rag.RetrieveService.QueryDocente` — no HTTP hop.
- **FR-004**: Any rubric document linked to the activity MUST be injected into the LLM prompt in full — never fragmented, never replaced by a similarity-ranked chunk (Principle V).
- **FR-005**: The LLM result MUST be stored with `advisory_text`, `suggested_grade_range`, `backend_used`, and `analysis_type = 'llm'` — these fields are append-only once written (Principle I).
- **FR-006**: The entire detection job (heuristic + LLM) MUST complete within 3 minutes of the job being enqueued (Principle II, RNF-08).
- **FR-007**: No LLM result, heuristic score, alert status, or flagged event data MUST be accessible to the student role under any circumstances (Principle IV).
- **FR-008**: The detection report endpoint MUST be accessible only to docentes scoped to the activity's course (Principle IV).
- **FR-009**: When a docente assigns a grade that falls outside the LLM-suggested range, the save operation MUST require a non-empty `override_reason` in the same request (RF-18, Principle III). Both `final_grade` and `override_reason` MUST be written to the `grade_decisions` table in a single transaction — neither can be stored without the other.
- **FR-010**: If the LLM call fails, the worker MUST NOT retry inline — it stores the heuristic result as complete and marks `llm_status: 'failed'` on the job record (Principle II).
- **FR-011**: All AI output in the detection report MUST be labelled as advisory — no field may imply an automatic grade assignment or penalty (Principle III).
- **FR-012**: The worker MUST be idempotent: if redelivered for a session that already has a heuristic record, it skips the heuristic step and proceeds to the LLM step.
- **FR-013**: The detection job MUST be triggered by the `session.closed` event on the existing M5 Redis Stream — no new worker process is introduced. The `session.closed` event guarantees the exam session is fully finalized and the submission file is committed to MinIO before the detection pipeline begins reading it.

### Key Entities

- **DetectionJob**: A Redis Streams message on the `session.closed` event, carrying `session_id`, `submission_id`, and `activity_id`; emitted when the exam session finalizes, guaranteeing the submission file is committed to MinIO before the detection pipeline reads it.
- **HeuristicResult**: A row in `analysis_results` with `analysis_type = 'heuristic'` containing `score` (INT, 0–100), `flagged_categories` (TEXT[], nullable for LLM row), and `event_count` (INT). Written as an INSERT — never updated after creation.
- **LLMResult**: A row in `analysis_results` with `analysis_type = 'llm'` containing `advisory_text` (TEXT), `suggested_grade_range` (TEXT), `backend_used` (TEXT), and `llm_status` (TEXT: `'success'` | `'failed'`). Written as a separate INSERT after the heuristic row — heuristic-specific columns are null on this row.
- **DetectionReport**: The docente-facing aggregation of HeuristicResult + LLMResult for a given submission; read-only; never exposed to the student role.
- **GradeDecision**: A row in a dedicated `grade_decisions` table — `(id UUID PK, session_id UUID FK→sessions, analysis_result_id UUID FK→analysis_results, final_grade NUMERIC, suggested_grade_range TEXT, override_reason TEXT NULLABLE, decided_by UUID FK→users, decided_at TIMESTAMPTZ)`; `override_reason` is required when `final_grade` falls outside `suggested_grade_range`; written in a single transaction with no subsequent updates (append-only, Principle I). The `analysis_results` table is never mutated by the grade step.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of completed exam submissions have a heuristic analysis record stored within 3 minutes of submission close — no submission is left without a detection record.
- **SC-002**: The async worker never blocks the student's file upload response — the upload HTTP response is returned in under 2 seconds in all cases.
- **SC-003**: When the LLM backend is unavailable, the detection report is still accessible to the docente showing the heuristic result and `llm_status: "unavailable"` — zero silent failures.
- **SC-004**: No detection report, heuristic score, or flag data is returned to a student-authenticated request — 0% data leakage across the role boundary.
- **SC-005**: Grade submissions where the final grade is outside the suggested range and no reason is provided are rejected 100% of the time — the override-reason constraint has no bypass path.
- **SC-006**: LLM prompts for activities with a rubric always contain the full rubric text — 0% of prompts contain a truncated or fragmented rubric.

---

## Assumptions

- The M5 async worker and Redis Streams consumer group are operational; M8 adds a new handler to the existing worker, not a new worker process.
- The M6 AI router is operational and accessible within the same Docker network.
- The M7 dual RAG retrieval endpoints are operational; M8 calls `rag.RetrieveService` directly as a Go function call (`QueryCurricular` / `QueryDocente`) — the `*rag.RetrieveService` instance is injected into the worker handler at startup. No HTTP hop to the API server.
- The `analysis_results` table exists from M5 with at minimum `session_id`, `analysis_type`, and `created_at` columns; M8 adds the heuristic and LLM-specific columns via migration.
- Submission content (files) is already stored in MinIO from M4; M8 reads files by `submission_id` using the existing MinIO client.
- The heuristic algorithm for M8 is rule-based (no ML model). Weights and thresholds are configurable via environment variables — `HEURISTIC_TAB_WEIGHT` (default 5), `HEURISTIC_CLIP_WEIGHT` (default 3), `HEURISTIC_GAP_WEIGHT` (default 2), `HEURISTIC_HIGH_THRESHOLD` (default 60) — so detection sensitivity can be tuned per deployment without a code change or redeploy. Hardcoding these values would require a release cycle every time an institution recalibrates detection sensitivity based on observed false-positive rates.
- The suggested grade range from the LLM is a free-text string (e.g., "60–70" or "Reprobado") — parsing it for comparison with the docente's assigned grade is done by simple range check logic configured per institution.
- Mobile/browser clients are out of scope; the detection report is consumed by the docente-facing web frontend (React, separate repo).
- Each session produces at most two rows in `analysis_results`: one `analysis_type='heuristic'` (always present on successful detection) and one `analysis_type='llm'` (present only when the LLM step succeeds). These are separate INSERTs — the heuristic row is never mutated by the LLM step.

## Clarifications

### Session 2026-05-20

- Q: Should heuristic and LLM results be stored as two separate rows in the same table or as a single row per session updated in two phases? → A: Two rows per session in `analysis_results`, each with its own `analysis_type` discriminator ('heuristic' / 'llm'); each is a separate INSERT; columns specific to the other type are null.
- Q: Does the detection worker invoke RAG retrieval via direct Go function calls or via HTTP to the API server? → A: Direct Go function call — `*rag.RetrieveService` injected at worker startup; no HTTP hop.
- Q: Where are the docente's final grade and override reason stored? → A: Dedicated `grade_decisions` table with FK to both sessions and analysis_results; single-transaction INSERT; analysis_results remains append-only.
- Q: Are heuristic weights and thresholds configurable or hardcoded? → A: Env-var configurable (`HEURISTIC_TAB_WEIGHT`, `HEURISTIC_CLIP_WEIGHT`, `HEURISTIC_GAP_WEIGHT`, `HEURISTIC_HIGH_THRESHOLD`) — institutions need to recalibrate detection sensitivity based on observed false-positive rates without requiring a code change or redeploy.
- Q: Which Redis Stream event triggers the detection job? → A: `session.closed` — guarantees the exam session is finalized and the submission file is committed to MinIO before the detection pipeline begins.
