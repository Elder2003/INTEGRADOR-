# Data Model: M8 — Detection Engine

## New Tables (migration 000008)

---

### analysis_results

Append-only. Two rows per session maximum — one per `analysis_type`. Never UPDATE or DELETE (Principle I).

| Column | Type | Constraints | Notes |
|---|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() | |
| session_id | UUID | NOT NULL FK→exam_sessions(id) | |
| analysis_type | TEXT | NOT NULL CHECK IN ('heuristic','llm') | Discriminator |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() | |
| score | INT | NULL | Heuristic only: 0–100 |
| flagged_categories | TEXT[] | NULL | Heuristic only: e.g. ['tab_activity','clipboard_activity'] |
| event_count | INT | NULL | Heuristic only: total trace events analysed |
| advisory_text | TEXT | NULL | LLM only: natural-language assessment |
| suggested_grade_range | TEXT | NULL | LLM only: e.g. "60–70" or "Reprobado" |
| backend_used | TEXT | NULL | LLM only: AI router backend label |
| llm_status | TEXT | NULL CHECK IN ('success','failed') | LLM only |

**Constraints:**
```sql
UNIQUE (session_id, analysis_type)
```

**Indexes:**
```sql
CREATE INDEX idx_analysis_results_session ON analysis_results(session_id);
```

**State transitions:**
- Heuristic row: INSERT once when heuristic step completes → immutable
- LLM row: INSERT once when LLM step completes (or with `llm_status='failed'`) → immutable
- No UPDATE path exists

---

### grade_decisions

One row per session. Append-only (Principle I). Written in single transaction with grade assignment.

| Column | Type | Constraints | Notes |
|---|---|---|---|
| id | UUID | PK DEFAULT gen_random_uuid() | |
| session_id | UUID | NOT NULL FK→exam_sessions(id) | |
| analysis_result_id | UUID | NOT NULL FK→analysis_results(id) | FK to the LLM row |
| final_grade | NUMERIC(5,2) | NOT NULL | Docente-assigned grade |
| suggested_grade_range | TEXT | NULL | Snapshot of LLM suggestion at time of decision |
| override_reason | TEXT | NULL | Required by app layer when grade outside range |
| decided_by | UUID | NOT NULL FK→users(id) | Docente's user ID |
| decided_at | TIMESTAMPTZ | NOT NULL DEFAULT now() | |

**Constraints:**
```sql
UNIQUE (session_id)  -- one grade decision per session
```

**Validation rule (application layer, not DB):**
- If `suggested_grade_range` is a numeric range (e.g. "60–70") AND `final_grade` falls outside [lo−5, hi+5]: `override_reason` MUST be non-empty.
- If `suggested_grade_range` is non-numeric or null: `override_reason` is optional.

---

## Existing Tables Referenced (no structural changes)

### exam_sessions (M5)
Used by M8 to: resolve `actividad_id` from `session_id`; scope docente access by joining to `actividades → cursos`.

Key columns used:
- `id` — join key
- `actividad_id` — RAG curricular scope
- `submission_id` — MinIO file lookup
- `student_id` — included in detection context

### trace_events (M5)
Read by the heuristic engine. Queried by `session_id`, grouped by `event_type`.

Key columns used:
- `event_type` — `tab_blur`, `clipboard_copy`, `clipboard_paste`, `heartbeat`
- `received_at` — heartbeat gap calculation
- `sequence_num` — ordering for gap analysis

### worker_jobs (M5)
M8 updates status from `processing` → `done` / `failed` after detection completes. Existing columns used:
- `session_id`, `submission_id` — input to detection
- `status` — updated by M8 worker handler
- `error` — populated on failure

### submissions (M4)
M8 reads `storage_key` (MinIO object key) and `original_filename` (for ext-based extraction). No schema change.

### actividades (M3)
M8 joins to get `codigo_materia`, `carrera`, `semestre` for curricular RAG scope. No schema change.

### cursos (M3)
M8 joins to get `docente_id` for docente RAG scope and access control. No schema change.

---

## Entity Relationships (M8 additions)

```
exam_sessions (1) ──< analysis_results (0..2)
exam_sessions (1) ──< grade_decisions (0..1)
analysis_results (1) ──< grade_decisions (via analysis_result_id FK to LLM row)
users (1) ──< grade_decisions (via decided_by)
```

---

## Config Extensions (internal/config/config.go)

Four new env-var-backed fields added to `Config` struct:

| Field | Env Var | Default | Notes |
|---|---|---|---|
| HeuristicTabWeight | HEURISTIC_TAB_WEIGHT | 5 | Score per tab_blur event |
| HeuristicClipWeight | HEURISTIC_CLIP_WEIGHT | 3 | Score per clipboard event |
| HeuristicGapWeight | HEURISTIC_GAP_WEIGHT | 2 | Score per heartbeat gap |
| HeuristicHighThreshold | HEURISTIC_HIGH_THRESHOLD | 60 | Score above which `high_suspicion=true` |
