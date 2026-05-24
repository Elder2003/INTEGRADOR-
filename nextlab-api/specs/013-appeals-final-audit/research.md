# Research: Appeals & Final Audit (M13)

## Decision 1: Appeal Entity Key — session_id vs submission_id

- **Decision**: Appeals are keyed to `session_id` (references `exam_sessions`), not `submission_id`.
- **Rationale**: All evidence lives under `session_id`: `trace_events`, `analysis_results`, and `grade_decisions` are all indexed by `session_id`. Using `session_id` gives the appeals package a single join key to assemble the complete evidence chain. The UNIQUE constraint `(session_id)` enforces one open appeal per session.
- **Alternatives considered**: Keying to `submission_id` — would require joining through `exam_sessions` to reach analysis results; more joins, same uniqueness. `submission_id` is also nullable on exam_sessions, making it a weaker key.

## Decision 2: Grade Divergence — Use Existing grade_decisions Table

- **Decision**: The spec's "GradeDivergenceLog" maps directly to the existing `grade_decisions` table (created in M8). No new migration needed for this entity. M13 adds only a `GET /sessions/:sessionId/grade-log` endpoint in the detection package to expose it.
- **Rationale**: `grade_decisions` already records `final_grade`, `suggested_grade_range`, `override_reason`, `decided_by`, `decided_at`. The `ErrOverrideRequired` guard in the detection service already enforces justification when the grade is outside the suggested range (equivalent to the spec's divergence requirement). Adding a new table would duplicate data.
- **Alternatives considered**: New `grade_divergence_log` table — redundant, violates "don't add tables for data already stored".

## Decision 3: AuditEvent Hash Chain Computation

- **Decision**: `event_hash = SHA-256(occurred_at|event_type|sequence|payload_json|prev_hash)` where all fields are UTF-8 strings, pipe-delimited; `prev_hash` is the empty string for the first event; result is lowercase hex.
- **Rationale**: Hashing all five fields means any change to event type, sequence, timestamp, payload, or chain linkage is detectable. SHA-256 is available in Go stdlib (`crypto/sha256`), produces a 64-char hex string, and is trivially reproducible in any language. The pipe delimiter avoids ambiguity between fields.
- **Alternatives considered**: JSON canonicalization (RFC 8785) — more complex to implement correctly across languages; overkill for this schema.

## Decision 4: Evidence Chain Event Types

- **Decision**: The audit evidence chain contains exactly four event types per appeal, inserted at appeal-time by replaying existing data:
  1. `submission_received` — from `exam_sessions.started_at` + `submissions.submitted_at`
  2. `analysis_completed` — from `analysis_results` (heuristic score + LLM suggested range)
  3. `appeal_filed` — new, inserted when student POSTs the appeal
  4. `decision_recorded` — inserted when docente or admin records a decision
- **Rationale**: Events 1 and 2 are back-filled from existing immutable records at appeal creation time; this is safe because those records cannot be mutated (Principle I). Events 3 and 4 are written as they occur.
- **Alternatives considered**: Real-time streaming of all system events into audit_events — would require touching every existing handler; excessive for M13 scope.

## Decision 5: Self-Review Auto-Escalation

- **Decision**: When a docente attempts to review an appeal and they are the owner of the activity (i.e., the appeal's session's actividad belongs to their course), AND the appeal type is `grade_dispute`, the review endpoint auto-escalates to admin status instead of allowing a decision. Docentes can still reject `analysis_dispute` appeals on their own activities.
- **Rationale**: For grade disputes the docente is effectively both the respondent (they assigned the grade) and the reviewer — conflict of interest. Analysis disputes are AI-generated and have no human respondent, so self-review is acceptable there.
- **Alternatives considered**: Block all docente reviews on own sessions — overly restrictive; docentes should be able to dismiss obviously frivolous analysis disputes.

## Decision 6: Integration Test Strategy

- **Decision**: `tests/integration/` uses `TestMain` to connect to `DATABASE_URL_TEST`, truncate all affected tables once before the suite, and run all tests sequentially. Each test function is responsible for its own seed data. No per-test transaction rollback.
- **Rationale**: Per-test rollback fails when the application code opens its own transactions (US2 atomic grade+log, US3 decisions). Truncate-once is simpler and reliable. Sequential execution avoids cross-test data contamination without the complexity of parallel test isolation.
- **Alternatives considered**: Parallel tests with per-test schema isolation (`SET search_path`) — complex setup; not justified at M13 test volume.

## Decision 7: New internal/appeals Package

- **Decision**: New `internal/appeals` package owns all appeal-specific domain logic. It reads from `exam_sessions`, `analysis_results`, `grade_decisions`, `submissions`, and `users` via direct SQL — no imports from `detection`, `tracing`, or `submission` packages.
- **Rationale**: Avoids circular imports (detection already reads from tracing; adding appeals→detection would risk cycles). Direct SQL is consistent with the pattern in all other packages (academic, admin, etc.).
- **Alternatives considered**: Extending the detection package — would bloat it and blur the package's responsibility boundary.
