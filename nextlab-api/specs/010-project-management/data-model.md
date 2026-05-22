# Data Model: M10 — Project Management

**Feature**: `specs/010-project-management/spec.md`
**Date**: 2026-05-22

---

## Existing Entities (M3) — unchanged

| Entity | Key columns | Notes |
|--------|-------------|-------|
| `actividades` | `id, curso_id, tipo actividad_tipo, deadline, trace_enabled GENERATED` | `tipo` enum extended in M10 |
| `inscripciones` | `user_id, curso_id, is_active` | enrollment |
| `submissions` | `(student_id, actividad_id) UNIQUE, file_key, submitted_at` | unchanged; used by practica/tarea/informe |

---

## Migration 000010 — Enum Extension

```sql
ALTER TYPE actividad_tipo ADD VALUE 'investigacion';
ALTER TYPE actividad_tipo ADD VALUE 'git_analysis';
```

`trace_enabled` generated column is unaffected: both new valores evaluate to `false`.

---

## New Entities

### `investigacion_stages`

Named checkpoints within an `investigacion` actividad. One actividad has 1..N stages.

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | `DEFAULT gen_random_uuid()` |
| `actividad_id` | UUID | `NOT NULL REFERENCES actividades(id) ON DELETE CASCADE` |
| `name` | TEXT | `NOT NULL` |
| `display_order` | INTEGER | `NOT NULL DEFAULT 0` — cosmetic sort order |
| `deadline` | TIMESTAMPTZ | `NOT NULL` — per-stage deadline |
| `description` | TEXT | `NOT NULL DEFAULT ''` |
| `created_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |

Index: `(actividad_id, display_order)`

Constraint: `UNIQUE (actividad_id, name)` — no duplicate stage names within one activity.

**Validation rules**:
- Deletion blocked when any `stage_submissions.stage_id = id` exists (FR-006).
- Publishing an `investigacion` actividad with 0 stages is rejected at the application layer (FR-007).

---

### `stage_submissions`

A student's file upload to a specific stage. One row per (student, stage) pair.

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | `DEFAULT gen_random_uuid()` |
| `stage_id` | UUID | `NOT NULL REFERENCES investigacion_stages(id)` |
| `student_id` | UUID | `NOT NULL REFERENCES users(id)` |
| `file_key` | TEXT | `NOT NULL` — MinIO object key |
| `original_filename` | TEXT | `NOT NULL` |
| `is_late` | BOOLEAN | `NOT NULL` — set at submit time (`submitted_at > stage.deadline`) |
| `grade` | NUMERIC(5,2) | nullable — 0–100, assigned by docente |
| `grade_note` | TEXT | `NOT NULL DEFAULT ''` |
| `graded_at` | TIMESTAMPTZ | nullable |
| `graded_by` | UUID | nullable `REFERENCES users(id)` |
| `submitted_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |

Constraint: `UNIQUE (stage_id, student_id)` — one submission per student per stage (upsert on re-submit).

Index: `(stage_id)`, `(student_id, stage_id)`

**State transitions**:
- `pending` → not yet submitted (virtual — no row exists).
- `submitted` → row exists, `is_late = false`.
- `submitted_late` → row exists, `is_late = true`.

---

### `git_submissions`

A student's Git URL submission for a `git_analysis` actividad. One row per (student, actividad).

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | `DEFAULT gen_random_uuid()` |
| `actividad_id` | UUID | `NOT NULL REFERENCES actividades(id)` |
| `student_id` | UUID | `NOT NULL REFERENCES users(id)` |
| `repo_url` | TEXT | `NOT NULL` |
| `submitted_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |
| `updated_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |

Constraint: `UNIQUE (actividad_id, student_id)` — UPSERT on re-submit; triggers new analysis job.

---

### `git_analyses`

Async analysis record attached to a `git_submission`. One row per attempt; latest is authoritative.

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | |
| `git_submission_id` | UUID | `NOT NULL REFERENCES git_submissions(id)` |
| `status` | TEXT | `NOT NULL DEFAULT 'pending'` — `pending / success / failed` |
| `commit_count` | INTEGER | nullable |
| `authors` | JSONB | nullable — `["Alice","Bob"]` |
| `first_commit_at` | TIMESTAMPTZ | nullable |
| `last_commit_at` | TIMESTAMPTZ | nullable |
| `deadline_rush_pct` | NUMERIC(5,2) | nullable — % of commits in last 24h before deadline |
| `consistency_score` | INTEGER | nullable — 0–100 |
| `flags` | JSONB | nullable — `["deadline_rush","multiple_authors"]` |
| `llm_advisory` | TEXT | nullable |
| `llm_status` | TEXT | `NOT NULL DEFAULT 'pending'` — `pending / success / failed` |
| `fail_reason` | TEXT | nullable — e.g. `clone_error` |
| `grade` | NUMERIC(5,2) | nullable |
| `override_reason` | TEXT | `NOT NULL DEFAULT ''` |
| `graded_by` | UUID | nullable `REFERENCES users(id)` |
| `graded_at` | TIMESTAMPTZ | nullable |
| `created_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |
| `updated_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |

Index: `(git_submission_id, created_at DESC)` — latest analysis first.

---

### `git_analysis_jobs`

Worker job queue for async Git analysis. Separate from `worker_jobs` (which requires an exam session FK).

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | |
| `git_analysis_id` | UUID | `NOT NULL REFERENCES git_analyses(id)` |
| `status` | TEXT | `NOT NULL DEFAULT 'queued'` — `queued / processing / done / failed` |
| `retries` | INTEGER | `NOT NULL DEFAULT 0` |
| `queued_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |
| `started_at` | TIMESTAMPTZ | nullable |
| `completed_at` | TIMESTAMPTZ | nullable |

Index: `(status, queued_at)` — worker poll query.

---

### `defense_appointments`

Scheduled date/time and grade for a student's `defensa_oral`. One row per (student, actividad).

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID PK | |
| `actividad_id` | UUID | `NOT NULL REFERENCES actividades(id)` |
| `student_id` | UUID | `NOT NULL REFERENCES users(id)` |
| `scheduled_at` | TIMESTAMPTZ | `NOT NULL` |
| `grade` | NUMERIC(5,2) | nullable |
| `graded_by` | UUID | nullable `REFERENCES users(id)` |
| `graded_at` | TIMESTAMPTZ | nullable |
| `created_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |
| `updated_at` | TIMESTAMPTZ | `NOT NULL DEFAULT now()` |

Constraint: `UNIQUE (actividad_id, student_id)` — UPSERT to replace appointment (latest active, per spec edge case).

Index: `(actividad_id)`, `(student_id, actividad_id)`

---

## Entity Relationships

```
actividades (tipo=investigacion)
  └─── investigacion_stages (1..N)
         └─── stage_submissions (0..1 per student)

actividades (tipo=git_analysis)
  └─── git_submissions (0..1 per student)
         └─── git_analyses (0..N, latest is current)
                └─── git_analysis_jobs (0..1 per analysis)

actividades (tipo=defensa_oral)
  └─── defense_appointments (0..1 per student)
```
