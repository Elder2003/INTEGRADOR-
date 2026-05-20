# Quickstart: M5 — Resolution & Tracing

End-to-end verification steps after M5 implementation. Assumes M1–M4 are fully deployed
and Docker Compose is running with a healthy stack.

---

## Prerequisites

- Docker Compose stack running: `docker compose up -d`
- Migration 000005 applied (verified by `\dt` showing `exam_sessions`, `trace_events`, `worker_jobs`)
- `gofiber/websocket/v2` added to `go.mod` and container rebuilt: `docker compose up --build`
- Valid JWT tokens for: a student user enrolled in a course, a docente who owns that course
- An `examen` or `defensa_oral` activity with a deadline at least 30 minutes in the future
- A WebSocket client (e.g., `websocat`, browser DevTools, or a small test script)

---

## Step 1 — Verify Migration Applied

```bash
docker compose exec db psql -U postgres -d nextlab -c "\dt"
# Expected: exam_sessions, trace_events, worker_jobs all listed
```

---

## Step 2 — Student Opens Exam Session

```bash
POST /actividades/<exam-aid>/session
Authorization: Bearer <student-jwt>
```

**Expected**: `201 Created`
```json
{
  "id": "<session-id>",
  "actividad_id": "<exam-aid>",
  "status": "active",
  "started_at": "...",
  "ws_url": "/sessions/<session-id>/ws"
}
```

Verify in DB:
```sql
SELECT status, event_count, last_event_hash FROM exam_sessions WHERE id = '<session-id>';
-- Expected: status='active', event_count=0, last_event_hash=NULL
```

**Idempotency check**: call the same endpoint again.
**Expected**: `200 OK` with the same session ID (no duplicate row created).

---

## Step 3 — Non-traced Activity Type Rejected

```bash
POST /actividades/<tarea-aid>/session
Authorization: Bearer <student-jwt>
```

**Expected**: `422 Unprocessable Entity`
```json
{"error": "trace_not_applicable"}
```

---

## Step 4 — WebSocket Connection

```bash
# Using websocat (or equivalent):
websocat "ws://localhost:3000/sessions/<session-id>/ws?token=<student-jwt>"
```

**Expected first server message**:
```json
{"type": "connected", "session_id": "<session-id>", "event_count": 0}
```

Verify `session_open` event appended automatically:
```sql
SELECT sequence_num, event_type, event_hash FROM trace_events WHERE session_id = '<session-id>';
-- Expected: 1 row, sequence_num=1, event_type='session_open', event_hash IS NOT NULL
```

---

## Step 5 — Send Browser Events

Send the following messages over the WebSocket:

```json
{"type": "tab_blur"}
```
**Expected server response**:
```json
{"type": "ack", "sequence_num": 2, "received_at": "..."}
```

```json
{"type": "heartbeat"}
```
**Expected**: `{"type": "ack", "sequence_num": 3, ...}`

```json
{"type": "clipboard_copy"}
```
**Expected**: `{"type": "ack", "sequence_num": 4, ...}`

Verify in DB (3 rows total + session_open):
```sql
SELECT sequence_num, event_type FROM trace_events WHERE session_id = '<session-id>' ORDER BY sequence_num;
-- Expected: (1,session_open), (2,tab_blur), (3,heartbeat), (4,clipboard_copy)
```

Verify `last_heartbeat_at` was updated:
```sql
SELECT last_heartbeat_at, event_count FROM exam_sessions WHERE id = '<session-id>';
-- Expected: last_heartbeat_at IS NOT NULL, event_count=4
```

---

## Step 6 — Rate Limit

Send 31 events rapidly (e.g., 31 `tab_focus` messages):
```json
{"type": "tab_focus"}
```

After the 30th event in the window: **Expected**:
```json
{"type": "error", "code": "rate_limit_exceeded", "message": "Too many events. Slow down."}
```

Verify: the 31st event is NOT stored in `trace_events` (row count does not exceed 30 + initial events).

---

## Step 7 — Submit File and Session Completes

While the WebSocket is still connected, submit a file via M4:
```bash
POST /actividades/<exam-aid>/submission
Authorization: Bearer <student-jwt>
Content-Type: multipart/form-data
[file field]
```

**Expected**: `201 Created` (response within 2 seconds regardless of worker activity)

Verify session transitions to `completed`:
```sql
SELECT status, submission_id, closed_at FROM exam_sessions WHERE id = '<session-id>';
-- Expected: status='completed', submission_id IS NOT NULL, closed_at IS NOT NULL
```

Verify server sends `session_closed` over the WebSocket:
```json
{"type": "session_closed", "reason": "submission", "closed_at": "..."}
```

---

## Step 8 — Worker Job Enqueued

Verify `worker_jobs` row created:
```sql
SELECT status, stream_id, enqueued_at FROM worker_jobs WHERE session_id = '<session-id>';
-- Expected: status='queued', stream_id IS NOT NULL, enqueued_at IS NOT NULL
```

---

## Step 9 — Worker Acknowledgement (within 3 minutes)

Monitor the worker container:
```bash
docker compose logs -f worker
```

Within 3 minutes, expect log lines:
```
{"level":"INFO","msg":"job claimed","job_id":"...","session_id":"..."}
{"level":"INFO","msg":"job acknowledged","job_id":"...","duration_ms":...}
```

Verify in DB:
```sql
SELECT status, done_at, attempt_count FROM worker_jobs WHERE session_id = '<session-id>';
-- Expected: status='done', done_at IS NOT NULL, attempt_count=1
```

---

## Step 10 — Docente Retrieves Trace

```bash
GET /submissions/<submission-id>/trace
Authorization: Bearer <docente-jwt>
```

**Expected**: `200 OK` with all events in sequence order.
Verify:
- `event_count` matches total rows in `trace_events`
- Events are ordered by `sequence_num` (1, 2, 3, ...)
- No clipboard content is present in any `metadata` field
- `session_open` is sequence 1; `session_close` (if sent) is last

---

## Step 11 — Hash Chain Verification

```bash
GET /submissions/<submission-id>/trace/verify
Authorization: Bearer <docente-jwt>
```

**Expected**: `200 OK`
```json
{"valid": true, "event_count": <N>}
```

Tamper test (manual DB modification — for validation only):
```sql
UPDATE trace_events SET event_hash = 'tampered' WHERE session_id = '<sid>' AND sequence_num = 3;
```
Call verify again. **Expected**:
```json
{"valid": false, "first_invalid_sequence": 3, "event_count": <N>}
```

Restore after test:
```sql
-- Re-run the hash computation or roll back the change.
```

---

## Step 12 — Cross-Course Access Rejected

```bash
GET /submissions/<submission-id>/trace
Authorization: Bearer <other-docente-jwt>
```

**Expected**: `403 Forbidden`
```json
{"error": "forbidden"}
```

---

## Step 13 — Worker Crash Recovery

Stop the worker mid-processing:
```bash
docker compose stop worker
```

Verify job is left in `processing` or `queued` status:
```sql
SELECT status FROM worker_jobs WHERE id = '<job-id>';
```

Restart the worker:
```bash
docker compose start worker
```

Within ~60 seconds the worker runs XCLAIM recovery and picks up the orphaned message.
Verify job completes without creating a duplicate:
```sql
SELECT status, attempt_count FROM worker_jobs WHERE id = '<job-id>';
-- Expected: status='done', attempt_count=2
```
Verify only ONE `worker_jobs` row exists for this session (no duplicate created).

---

## Checklist Summary

| Scenario | Expected Result |
|---|---|
| POST session for examen | 201 with active session |
| POST session idempotent | 200 with same session |
| POST session for tarea | 422 trace_not_applicable |
| WebSocket auth with valid JWT | Upgrade succeeds, session_open appended |
| WebSocket auth with invalid JWT | HTTP 401, no upgrade |
| tab_blur event | ACK with sequence_num, stored in trace_events |
| heartbeat event | ACK, last_heartbeat_at updated |
| clipboard_copy event | ACK, no content in metadata |
| Rate limit (31st event/min) | error frame, event NOT stored |
| File upload completes session | 201 within 2s, session=completed |
| Worker acknowledgement | done within 3 min |
| Docente GET trace | All events in sequence order |
| Verify integrity | {valid: true} |
| Tampered hash | {valid: false, first_invalid_sequence: N} |
| Cross-course docente | 403 forbidden |
| Worker crash + restart | Job reprocessed, attempt_count incremented, no duplicate |
