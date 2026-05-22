# Research: M8 — Detection Engine

## Decision 1: analysis_results table origin

**Decision**: M8 creates `analysis_results` and `grade_decisions` from scratch via `000008_detection_engine.up.sql`. The M5 migration does NOT include `analysis_results` — the spec assumption was forward-looking. The table is new.

**Rationale**: Inspection of `migrations/000005_tracing.up.sql` confirms only `exam_sessions`, `trace_events`, and `worker_jobs` were created in M5.

**Alternatives considered**: Patching M5 migration — rejected; migrations are immutable once applied.

---

## Decision 2: Two-row analysis_results schema

**Decision**: One table `analysis_results` with a `UNIQUE (session_id, analysis_type)` constraint. Each session produces at most two rows: one `analysis_type='heuristic'`, one `analysis_type='llm'`. Heuristic-specific columns (`score`, `flagged_categories`, `event_count`) are null on the LLM row and vice versa (`advisory_text`, `suggested_grade_range`, `backend_used`, `llm_status`).

**Rationale**: Matches clarification Q1. The `analysis_type` discriminator pattern was already anticipated in the spec and constitution. The `UNIQUE` constraint enforces at-most-one row per type per session.

**Alternatives considered**: Separate tables `heuristic_results` + `llm_results` — rejected for adding JOIN complexity to the report endpoint without meaningful benefit at this scale.

---

## Decision 3: grade_decisions table

**Decision**: Separate `grade_decisions` table with FK to both `exam_sessions` and `analysis_results` (the LLM row). Append-only (no UPDATE ever). `override_reason` is nullable in the DB but validated at the application layer when `final_grade` falls outside `suggested_grade_range`.

**Rationale**: Keeps `analysis_results` immutable (Principle I). Separate table gives a clean audit trail for grade decisions independent of the AI output rows.

**Alternatives considered**: Adding grade columns to `analysis_results` LLM row — rejected; would require an UPDATE, violating Principle I.

---

## Decision 4: Worker dependency injection

**Decision**: `cmd/worker/main.go` is extended to construct: `*pgxpool.Pool` (already present), `*minio.Client` (new), `*ai.Service` (new), `*rag.RetrieveService` (new). These are passed to a new `detection.NewWorkerHandler(...)` that `processJob` delegates to.

**Rationale**: The worker binary already connects to DB and Redis. MinIO, AI service, and RAG service are standard dependencies used by other modules in the same project. No additional env vars beyond those already in docker-compose.yml are needed (MinIO and AI are already wired).

**Alternatives considered**: HTTP calls back to the API server — rejected per clarification Q2 (no HTTP hop).

---

## Decision 5: Submission file retrieval

**Decision**: The `submissions` table stores the MinIO object key in a `storage_key` TEXT column. The worker fetches the submission row by `submission_id`, reads `storage_key`, then calls `minio.GetObject(ctx, minioBucket, storageKey, ...)` to stream the file bytes. File extension is derived from `original_filename`. Text extraction reuses `rag.ExtractText(data, ext)`.

**Rationale**: `storage_key` confirmed by inspecting `internal/submission/repository.go` line 57 and the `submissions` INSERT at line 105. The `rag` package's `ExtractText` already handles PDF/DOCX/TXT — no duplication.

**Alternatives considered**: Duplicate extraction logic in detection package — rejected; DRY principle, `rag.ExtractText` is already tested.

---

## Decision 6: actividad_id derivation

**Decision**: When `processJob` receives `session_id`, it queries `exam_sessions` to get `actividad_id`. From `actividad_id`, it queries `actividades` to get `codigo_materia`, `carrera`, `semestre` for the curricular RAG scope. The docente RAG scope uses `docente_id + curso_id` from `cursos JOIN actividades`.

**Rationale**: The Redis Stream message only carries `job_id` and `session_id`. The `worker_jobs` table has `submission_id` but not `actividad_id`. This join is a single DB read — negligible cost.

**Alternatives considered**: Adding `actividad_id` to the stream message — rejected; the job table and session table are the authoritative source; duplicating it in the stream creates drift risk.

---

## Decision 7: Heuristic scoring formula

**Decision**: Additive scoring capped at 100:

```
score = min(100,
  tab_blur_count   × HEURISTIC_TAB_WEIGHT   +
  clipboard_count  × HEURISTIC_CLIP_WEIGHT  +
  heartbeat_gaps   × HEURISTIC_GAP_WEIGHT
)
```

Where:
- `tab_blur_count` = count of `tab_blur` events in the session
- `clipboard_count` = count of `clipboard_copy` + `clipboard_paste` events
- `heartbeat_gaps` = count of gaps > 30s between consecutive heartbeat events

Flagged categories:
- `tab_activity` if `tab_blur_count > 0`
- `clipboard_activity` if `clipboard_count > 0`
- `heartbeat_gap` if `heartbeat_gaps > 0`

All weights read from env vars (clarification Q4): `HEURISTIC_TAB_WEIGHT` (default 5), `HEURISTIC_CLIP_WEIGHT` (default 3), `HEURISTIC_GAP_WEIGHT` (default 2), `HEURISTIC_HIGH_THRESHOLD` (default 60).

**Rationale**: Simple additive model is interpretable — docentes can reason about why a score is high. Cap at 100 prevents extreme events from producing nonsensical scores. Env-var configurability per clarification Q4.

**Alternatives considered**: Weighted average — rejected; additive is more intuitive for academic reviewers.

---

## Decision 8: LLM prompt structure

**Decision**: The detection prompt is constructed as a system + user message pair:

```
[SYSTEM]
You are an academic integrity assistant. Analyse the student submission below
and provide an advisory assessment. Do NOT assign a final grade — your output
is advisory only and will be reviewed by the course instructor.

[USER]
## Submission Text
{submission_text | "(file content unavailable)"}

## Behavioral Analysis
Suspicion score: {score}/100
Flagged indicators: {flagged_categories | "none"}
Total trace events analysed: {event_count}

## Curricular Context
{curricular_chunks joined by "\n---\n" | "(no curricular context available)"}

## Rubric (full, untruncated)
{rubric_text | "(no rubric available)"}

## Task
1. Assess whether the submission is consistent with the student's expected knowledge level.
2. Suggest a grade range (e.g., "60–70" or "Reprobado") with a brief justification.
3. Note any specific concerns raised by the behavioral indicators.
Format your response as JSON: {"advisory_text":"...","suggested_grade_range":"..."}
```

**Rationale**: Structured JSON response makes `advisory_text` and `suggested_grade_range` parseable without brittle regex. System message enforces advisory framing (Principle III). Rubric is always included in full (Principle V).

**Alternatives considered**: Free-text response with post-hoc parsing — rejected; JSON is more reliable for `suggested_grade_range` extraction.

---

## Decision 9: Grade range override detection

**Decision**: `suggested_grade_range` is a string like `"60–70"` or `"Reprobado"`. Override detection logic:
- If it contains `"–"` or `"-"`, parse as numeric range `[lo, hi]`.
- `final_grade` is considered "outside range" if it is < lo − 5 or > hi + 5 (5-point tolerance for rounding).
- If `suggested_grade_range` is non-numeric (e.g., `"Reprobado"`), override detection is skipped — `override_reason` is not required.
- If parsing fails, override detection is skipped.

**Rationale**: LLM output may use various range formats. A 5-point tolerance avoids rejecting valid grades on the boundary. Non-numeric ranges (pass/fail words) can't be compared numerically.

---

## Decision 10: Job trigger event

**Decision**: `session.closed` trigger confirmed (clarification Q5). Specifically, `CloseSessionWithSubmission` in `tracing.Service` already calls `EnqueueJobForSession` — this is the existing trigger path. M8 does NOT need to change the trigger; it only needs to extend `processJob` in the worker to call `detection.WorkerHandler.Run(ctx, sessionID, submissionID)`.

**Rationale**: The stream message format (`job_id`, `session_id`) already flows through `CloseSessionWithSubmission` → `EnqueueJobForSession` → Redis Stream → `processJob`. M8 plugs into `processJob` only.

---

## Decision 11: Detection HTTP endpoints

**Decision**: Two new routes added to `internal/detection/module.go` and mounted in `cmd/api/main.go`:

- `GET /sessions/:sessionID/detection-report` — requires docente role, scoped to docente's own courses
- `POST /sessions/:sessionID/grade` — requires docente role, same scope

Student role blocked at middleware before any DB query (Principle IV, FR-007).

**Rationale**: Routes are session-scoped (`:sessionID`) which naturally aligns with the existing `exam_sessions` table and tracing module routes. Docente scope validated by checking that `exam_sessions.actividad_id` → `actividades.curso_id` → `cursos.docente_id` matches the authenticated user.
