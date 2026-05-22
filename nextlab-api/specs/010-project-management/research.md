# Research: M10 — Project Management

**Feature**: `specs/010-project-management/spec.md`
**Date**: 2026-05-22

---

## Decision 1: Git Cloning Library

**Decision**: Use `github.com/go-git/go-git/v5` (pure Go in-process clone).

**Rationale**: Pure Go — no system `git` binary required in the worker Docker image. Shallow clone (`--depth 50`) is supported via `CloneOptions.Depth`, keeping memory bounded. The library provides `CommitIterator` with author name, message, and timestamp directly usable for heuristic extraction.

**Alternatives considered**:
- Shell `exec.Command("git", "clone")`: requires git installed in container; adds OS-level dependency; harder to sandbox; subprocess errors are harder to capture cleanly.
- `github.com/libgit2/git2go`: CGo dependency; C library must be present at build time; increases image size significantly.

---

## Decision 2: Worker Job Routing for Git Analysis

**Decision**: Add a separate `git_analysis_jobs` table with `git_submission_id` FK; teach the worker to check both `worker_jobs` (exam sessions) and `git_analysis_jobs` on each poll cycle.

**Rationale**: The existing `worker_jobs` table has `session_id UUID NOT NULL REFERENCES exam_sessions(id)`. Git analysis has no exam session. Rather than NULLing the FK (which would break the non-null constraint) or adding a nullable FK, a dedicated table keeps the schema clean and preserves the invariant that every `worker_jobs` row has a valid session.

**Alternatives considered**:
- Make `worker_jobs.session_id` nullable and add `git_submission_id` nullable: creates two nullable FKs with a check constraint; clutters the existing table used by all M4–M9 jobs.
- Single generic jobs table with polymorphic discriminator: premature generalization; adds complexity without benefit for current scope.

---

## Decision 3: Stage Submissions vs Extending `submissions`

**Decision**: New `stage_submissions` table (separate from the existing `submissions` table).

**Rationale**: The existing `submissions` table is keyed on `(student_id, actividad_id)` with a `UNIQUE` constraint — one submission per student per activity. `investigacion` requires one submission per student per stage, making the existing key incompatible. A new table avoids altering the unique constraint and avoids breaking all M3–M9 code that assumes at most one submission per (student, actividad).

**Alternatives considered**:
- Add `stage_id` nullable FK to `submissions`: breaks the existing UNIQUE constraint semantics; all existing queries would need NULL checks; risky blast radius.
- Shared table with discriminator column: same problems as above.

---

## Decision 4: `actividad_tipo` Enum Extension

**Decision**: `ALTER TYPE actividad_tipo ADD VALUE 'investigacion'; ALTER TYPE actividad_tipo ADD VALUE 'git_analysis';` in a new migration (000010).

**Rationale**: PostgreSQL allows adding values to an existing enum in a transaction-safe way (since PG 12). No existing data is affected; the generated column `trace_enabled = tipo IN ('examen', 'defensa_oral')` evaluates to `false` for both new types, so no tracing is triggered.

**Alternatives considered**:
- New string column instead of enum: loses DB-level validation; inconsistent with existing tipos.
- Drop and recreate enum: requires CASCADE dropping all dependent objects; not safe with existing data.

---

## Decision 5: `DefenseAppointment` — Separate Table or Column

**Decision**: New `defense_appointments` table: `(student_id, actividad_id) UNIQUE`, `scheduled_at TIMESTAMPTZ`, `grade NUMERIC(5,2)` nullable, `graded_at TIMESTAMPTZ` nullable.

**Rationale**: Keeps the `actividades` table clean; the appointment is a per-student record orthogonal to the activity definition. A table follows the same pattern as `exam_sessions` for `examen` tipo.

**Alternatives considered**:
- Add columns to `submissions`: oral defense explicitly has no submission (FR-018); would require a row with no file to hold the date — semantically wrong.

---

## Decision 6: `internal/projects/` Package Scope

**Decision**: All M10 business logic lives in `internal/projects/`. The package already exists as an empty stub (`package projects`). It will contain: `model.go`, `repository.go`, `service_investigacion.go`, `service_git_analysis.go`, `service_defensa.go`, `handler.go`, `module.go`.

**Rationale**: Follows the existing module pattern (`internal/academic/`, `internal/detection/`, `internal/format/`). Avoids adding M10 types to `internal/academic/` which is already M3–M6 scope.

---

## Decision 7: Git Analysis LLM Prompt

**Decision**: Reuse `ai.Service.Dispatch` with a new Git-specific prompt (no RAG — commit metadata is self-contained). JSON response format: `{"advisory_text":"...","suggested_grade_range":"..."}`.

**Rationale**: Same dispatch infrastructure as M8; no curricular or docente RAG needed for commit history analysis. The LLM receives: commit count, author list, first/last commit dates, deadline-rush percentage, consistency score, heuristic flags.

**Alternatives considered**:
- RAG with rubric: Git commit patterns are not rubric-evaluated; RAG would add latency for no benefit.

---

## Decision 8: Shallow Clone Depth

**Decision**: `CloneOptions.Depth = 200` (fetch last 200 commits on default branch only).

**Rationale**: Bounds worker memory and clone time for large repositories. Commit count, author list, and deadline-rush percentage are all accurately computed from recent history. For repositories with <200 commits the full history is fetched automatically.

**Alternatives considered**:
- Full clone: unbounded; a repo with 100k commits would exhaust worker memory.
- Depth 50: too shallow for semester-long projects that may have daily commits over 4 months (~120 commits).
