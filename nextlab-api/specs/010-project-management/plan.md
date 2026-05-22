# Implementation Plan: M10 — Project Management

**Branch**: `010-project-management` | **Date**: 2026-05-22 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/010-project-management/spec.md`

## Summary

Three independent activity types extending the existing academic management system:
1. **`investigacion`** — multi-stage research report; docente defines named stages with individual deadlines; students submit files per-stage independently; docente views progress matrix and grades each stage submission independently.
2. **`git_analysis`** — standalone homework-like activity where the deliverable is a Git repository URL; async worker clones the repo, runs heuristic analysis (deadline_rush, multiple_authors, consistency_score), then dispatches to the LLM advisory pipeline (same M8 pattern); docente reviews report and assigns grade.
3. **`defensa_oral`** — lightweight extension of the existing `defensa_oral` tipo; docente schedules a per-student date/time; student sees their appointment; docente records grade directly with no tracing pipeline.

## Technical Context

**Language/Version**: Go 1.25.0

**Primary Dependencies**:
- `github.com/gofiber/fiber/v2` — HTTP handlers (existing)
- `github.com/jackc/pgx/v5` — PostgreSQL driver (existing)
- `github.com/google/uuid` — UUID handling (existing)
- `github.com/minio/minio-go/v7` — file storage for stage submissions (existing)
- `github.com/Elder2003/nextlab-api/internal/ai` — LLM dispatch for git analysis (existing)
- `github.com/go-git/go-git/v5` — **NEW** pure-Go Git clone for analysis worker

**Storage**: PostgreSQL (existing); MinIO for stage submission files (existing)

**Testing**: `go test ./...` with `pgxmock` or test DB; existing pattern from M3–M9

**Target Platform**: Linux server (Docker Compose)

**Project Type**: Web service (REST API + async worker)

**Performance Goals**: Git analysis report available within 3 minutes of submission for 95% of cases (SC-001); stage-progress overview for 30 students in a single response (SC-002)

**Constraints**: No new infrastructure — reuse existing MinIO, Redis, PostgreSQL, worker; shallow clone depth 200 to bound memory

**Scale/Scope**: 30 students per activity; semester-scale usage

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I — Immutable Evidence | PASS | Stage submissions stored with `is_late` set at write time; never modified retroactively |
| II — Async Isolation | PASS | Git analysis via separate `git_analysis_jobs` table; worker processes independently; failure stored, submission unaffected |
| III — Human Authority | PASS | Docente grades are final for all three features; no system override |
| IV — Role Isolation | PASS | Git analysis report 403 for students (FR-013); defense appointments scoped per student |
| V — Dual RAG Independence | N/A | Git analysis uses no RAG (commit metadata is self-contained) |
| VI — Non-Invasive Auditing | PASS | New tipos have `trace_enabled = false` (generated column evaluates to false) |
| VII — Submission Always Accepted | PASS | Late stage submissions accepted with `is_late=true`; clone failures stored, submission unaffected |

**Post-design re-check**: All gates pass. New `investigacion` and `git_analysis` tipos do not trigger `trace_enabled`. Git analysis failure path (FR-012) stores `clone_error` without touching submission record.

## Project Structure

### Documentation (this feature)

```text
specs/010-project-management/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── project-management.md   # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code

```text
nextlab-api/
├── migrations/
│   ├── 000010_project_management.up.sql    # enum extension + 5 new tables
│   └── 000010_project_management.down.sql
├── internal/
│   └── projects/
│       ├── model.go                 # entities: InvestigacionStage, StageSubmission, GitSubmission, GitAnalysis, DefenseAppointment
│       ├── repository.go            # DB queries for all 3 features
│       ├── service_investigacion.go # stage CRUD, submit, progress, grade
│       ├── service_git_analysis.go  # git submit, enqueue job, grade
│       ├── service_defensa.go       # schedule, grade
│       ├── handler.go               # Fiber handlers for all routes
│       └── module.go                # wire-up: NewModule(db, minio, ai)
├── cmd/worker/
│   └── main.go                      # add git_analysis_jobs poll loop
└── go.mod                           # add github.com/go-git/go-git/v5
```

**Structure Decision**: All M10 logic in `internal/projects/` (existing empty stub). Worker extended in `cmd/worker/main.go` to handle git analysis jobs alongside existing session-based jobs.

## Complexity Tracking

No constitution violations. No complexity justification required.
