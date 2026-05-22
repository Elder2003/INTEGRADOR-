# Quickstart: M8 — Detection Engine

Step-by-step verification of all M8 acceptance criteria against a live Docker Compose stack.

**Prerequisites**: Docker Compose running (`docker compose up -d`), M5 session+submission flow operational, M6 AI router reachable, M7 RAG ingested at least one curricular document, valid docente + student JWTs in hand.

---

## Phase 1: Migration & Build

**Step 1 — Apply migration**
```bash
docker compose run --rm api sh -c "echo 'migrations run on startup'"
# Or verify manually:
docker compose exec postgres psql -U postgres nextlab -c "\d analysis_results"
docker compose exec postgres psql -U postgres nextlab -c "\d grade_decisions"
```
Expected: both tables exist with correct columns.

**Step 2 — Build succeeds**
```bash
go build ./...
```
Expected: zero errors.

---

## Phase 2: Heuristic Detection (US1)

**Step 3 — Submit an exam with suspicious events**

As a student, open a session for a traced actividad. During the session, trigger:
- 5+ `tab_blur` events (switch tabs repeatedly)
- 5+ `clipboard_copy` events

Submit the exam file. Session closes. Worker picks up the job.

Wait up to 3 minutes, then:
```sql
SELECT analysis_type, score, flagged_categories, event_count
FROM analysis_results
WHERE session_id = '<session_id>' AND analysis_type = 'heuristic';
```
Expected:
- 1 row returned
- `score` > 0 (tab_blur × 5 + clipboard × 3 = 40 with defaults)
- `flagged_categories` includes `'tab_activity'` and `'clipboard_activity'`
- `event_count` > 0

**Step 4 — Submit with zero suspicious events**

New session, no tab switches, no clipboard. Submit. Wait for worker.
```sql
SELECT score, flagged_categories FROM analysis_results
WHERE session_id = '<clean_session_id>' AND analysis_type = 'heuristic';
```
Expected: `score = 0`, `flagged_categories = '{}'`.

**Step 5 — LLM failure does not destroy heuristic row**

Temporarily disable the AI router (set `AI_PRIMARY_URL` to an invalid host, restart API). Submit a new exam. Worker runs heuristic, attempts LLM (fails), stores LLM row with `llm_status='failed'`.
```sql
SELECT analysis_type, llm_status FROM analysis_results WHERE session_id = '<session_id>';
```
Expected: 2 rows — heuristic row untouched, LLM row with `llm_status='failed'`.

---

## Phase 3: LLM Analysis with RAG Context (US2)

**Step 6 — LLM row stored with expected fields**

With AI router functional, submit an exam. Wait for worker.
```sql
SELECT advisory_text, suggested_grade_range, backend_used, llm_status
FROM analysis_results
WHERE session_id = '<session_id>' AND analysis_type = 'llm';
```
Expected: all four columns non-null, `llm_status = 'success'`.

**Step 7 — Rubric injected in full**

Ingest a rubric document for the actividad's subject via `POST /rag/curricular/ingest` with `is_rubric=true` and `actividad_id` set. Submit an exam for that actividad. Inspect worker logs for the prompt — confirm rubric text appears untruncated.

Expected (log output or debug mode): prompt contains the full rubric text between `## Rubric` and the next `##` heading.

**Step 8 — Empty RAG context does not block LLM**

Run against an actividad with no ingested RAG documents. Worker should complete with `llm_status='success'` and a prompt that notes `"(no curricular context available)"`.

---

## Phase 4: Detection Report Endpoint (US3)

**Step 9 — Docente reads full report**

```bash
curl -H "Authorization: Bearer <docente_jwt>" \
  http://localhost:8080/sessions/<session_id>/detection-report
```
Expected: HTTP 200, response contains `heuristic` sub-report (score, flagged_categories, event_count) and `llm` sub-report (advisory_text, suggested_grade_range, backend_used), `llm_status: "success"`.

**Step 10 — Heuristic-only report when LLM failed**

Use the session from Step 5 (LLM disabled).
```bash
curl -H "Authorization: Bearer <docente_jwt>" \
  http://localhost:8080/sessions/<llm_failed_session_id>/detection-report
```
Expected: HTTP 200, `llm: null`, `llm_status: "unavailable"` — no error response.

**Step 11 — Grade within suggested range (no reason required)**

```bash
curl -X POST -H "Authorization: Bearer <docente_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"final_grade": 65}' \
  http://localhost:8080/sessions/<session_id>/grade
```
Assume `suggested_grade_range` is `"60–70"`. Expected: HTTP 201, grade recorded, no `override_reason` field required.

**Step 12 — Grade outside range without reason rejected**

```bash
curl -X POST -H "Authorization: Bearer <docente_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"final_grade": 40}' \
  http://localhost:8080/sessions/<session_id>/grade
```
Expected: HTTP 422, `"error": "override_reason_required"`.

**Step 13 — Grade outside range with reason accepted**

```bash
curl -X POST -H "Authorization: Bearer <docente_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"final_grade": 40, "override_reason": "extenuating circumstances documented"}' \
  http://localhost:8080/sessions/<session_id>/grade
```
Expected: HTTP 201, grade and reason stored.
```sql
SELECT final_grade, override_reason FROM grade_decisions WHERE session_id = '<session_id>';
```
Expected: `final_grade = 40.00`, `override_reason` = the provided string.

---

## Phase 5: Student Access Denied (US4)

**Step 14 — Student 403 on detection-report**

```bash
curl -H "Authorization: Bearer <student_jwt>" \
  http://localhost:8080/sessions/<own_session_id>/detection-report
```
Expected: HTTP 403, no detection data in response body.

**Step 15 — Student 403 on grade endpoint**

```bash
curl -X POST -H "Authorization: Bearer <student_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"final_grade": 80}' \
  http://localhost:8080/sessions/<own_session_id>/grade
```
Expected: HTTP 403.

---

## Phase 6: Idempotency

**Step 16 — Redelivered job skips completed heuristic**

Simulate redelivery by re-calling the worker handler directly with a `session_id` that already has a `analysis_type='heuristic'` row. Expected: handler detects existing row, skips heuristic step, proceeds to LLM (or skips both if LLM row also exists). No duplicate rows in `analysis_results`.

**Step 17 — Duplicate grade assignment rejected**

```bash
# Second POST to same session (already graded in Step 13)
curl -X POST -H "Authorization: Bearer <docente_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"final_grade": 70}' \
  http://localhost:8080/sessions/<session_id>/grade
```
Expected: HTTP 409 Conflict.

---

## SLA Check

**Step 18 — End-to-end under 3 minutes**

Record `enqueued_at` from `worker_jobs` and `created_at` of the LLM `analysis_results` row for the same session.
```sql
SELECT wj.enqueued_at, ar.created_at,
       EXTRACT(EPOCH FROM (ar.created_at - wj.enqueued_at)) AS seconds
FROM worker_jobs wj
JOIN analysis_results ar ON ar.session_id = wj.session_id AND ar.analysis_type = 'llm'
WHERE wj.session_id = '<session_id>';
```
Expected: `seconds` < 180.
