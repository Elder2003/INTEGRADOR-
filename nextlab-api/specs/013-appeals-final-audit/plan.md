# Implementation Plan: Appeals & Final Audit

**Branch**: `013-appeals-final-audit` | **Date**: 2026-05-23 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/013-appeals-final-audit/spec.md`

## Summary

Add a formal appeals system on top of the existing exam-session + grade-decision infrastructure. Students file appeals against analysis or grade results; docentes review or escalate; admins make final decisions. All appeal records and decisions are append-only. A hash-chained evidence endpoint (`GET /appeals/:id/evidence`) assembles submission metadata, analysis results, and appeal decisions into a tamper-verifiable audit trail. End-to-end integration tests cover the three critical flows against a real PostgreSQL database.

## Technical Context

**Language/Version**: Go 1.25

**Primary Dependencies**: Fiber v2, pgx/v5, golang-jwt/v5, crypto/sha256 (stdlib)

**Storage**: PostgreSQL — new tables `appeals`, `appeal_decisions`, `audit_events`; existing `grade_decisions`, `analysis_results`, `exam_sessions`, `trace_events` tables are read-only from the appeals package

**Testing**: `go test ./tests/integration/...` with real PostgreSQL (no mocks); `DATABASE_URL_TEST` env var

**Target Platform**: Linux server (existing Docker Compose stack)

**Project Type**: Web service — REST API extension

**Performance Goals**: Appeal submission < 500 ms p95; evidence chain retrieval < 1 s p95 (low-traffic endpoint)

**Constraints**: No UPDATE or DELETE on appeal, appeal_decision, or audit_event rows (Principle I); admin has global scope; all decisions immutable after creation

**Scale/Scope**: Low volume (appeals per exam session are rare); up to ~50 concurrent docentes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I — Immutable Evidence | **PASS** | `appeals`, `appeal_decisions`, `audit_events` are append-only; no UPDATE/DELETE migrations |
| II — Async Analysis Isolation | **PASS** | Appeals are synchronous HTTP; no analysis work done inline |
| III — Human Authority Over AI | **PASS** | Grade divergence already enforced in `grade_decisions`; M13 exposes it; admin makes final appeal decision |
| IV — Strict Role-Based Data Isolation | **PASS** | Students see own; docentes see their activities; admins have global scope |
| V — Dual RAG Independence | **N/A** | No RAG interaction in appeals |
| VI — Non-Invasive Auditing | **PASS** | Evidence chain reads existing session data; no new capture mechanism |
| VII — Submission Always Accepted | **PASS** | Appeals accepted regardless of activity deadline state |

*Post-design re-check: all gates hold. No violations.*

## Project Structure

### Documentation (this feature)

```text
specs/013-appeals-final-audit/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── appeals.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/
└── appeals/
    ├── model.go          # Appeal, AppealDecision, AuditEvent, GradeLogResult structs
    ├── repository.go     # DB operations for appeals, decisions, audit_events
    ├── handler.go        # Fiber handlers for all appeal endpoints
    └── module.go         # NewModule, RegisterRoutes

migrations/
├── 000013_appeals_final_audit.up.sql
└── 000013_appeals_final_audit.down.sql

tests/
└── integration/
    ├── setup_test.go         # TestMain: DB connection, truncate helper
    ├── exam_session_test.go  # Flow: open session → events → close → analysis
    ├── submission_test.go    # Flow: upload → format report → grade decision
    └── appeal_test.go        # Flow: file appeal → review → escalate → admin resolve → evidence chain
```

**Structure Decision**: Single-package extension (`internal/appeals`) + integration tests under `tests/integration/`. The appeals package reads exam_sessions, analysis_results, grade_decisions, and trace_events via direct SQL (no cross-package imports into detection/tracing to avoid circular deps). Grade-log endpoint lives in the detection handler (already owns grade_decisions).
