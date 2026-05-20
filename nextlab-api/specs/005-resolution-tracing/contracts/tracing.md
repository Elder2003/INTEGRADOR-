# API Contract: M5 — Resolution & Tracing

**Base prefix**: All routes require `Authorization: Bearer <jwt>` header unless noted.

---

## Session Management

### POST `/actividades/:aid/session`

**Role**: `estudiante`

Opens an exam session for the given activity. If an active session already exists for this
student and activity, the existing session is returned (no duplicate created).

**Path parameters**:
- `aid` — UUID of the actividad (must have `trace_enabled = true`)

**Request body**: none

**Responses**:

| Status | Body | Condition |
|---|---|---|
| 201 | `ExamSessionResponse` | New session created |
| 200 | `ExamSessionResponse` | Existing active session returned |
| 403 | `{"error": "not_enrolled"}` | Student is not enrolled in the curso |
| 422 | `{"error": "trace_not_applicable"}` | Activity type is not `examen` or `defensa_oral` |
| 410 | `{"error": "session_closed", "status": "<status>"}` | A prior closed session exists (deadline passed or abandoned) |

**ExamSessionResponse**:
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "status": "active",
  "started_at": "2026-05-20T10:00:00Z",
  "ws_url": "/sessions/{id}/ws"
}
```

---

### GET `/actividades/:aid/session`

**Role**: `estudiante`

Returns the current session for this student and activity.

**Responses**:

| Status | Body | Condition |
|---|---|---|
| 200 | `ExamSessionResponse` | Session found |
| 404 | `{"error": "session_not_found"}` | No session exists |

---

## WebSocket Event Stream

### GET `/sessions/:sid/ws?token=<jwt>`

**Role**: `estudiante`

Upgrades to a WebSocket connection for real-time event streaming. The JWT is passed as a
query parameter because browser WebSocket APIs cannot set custom headers during the upgrade
handshake (see research.md Decision 2).

**Authentication**: The `token` query parameter is validated by Fiber middleware **before**
the WebSocket upgrade. An invalid or missing token returns HTTP 401 — the connection is
never opened.

**Path parameters**:
- `sid` — UUID of the exam session (must belong to the authenticated student and be `active`)

**On successful upgrade**: the server sends a `connected` message:
```json
{"type": "connected", "session_id": "uuid", "event_count": 0}
```

---

#### Client → Server Messages

The client sends JSON objects. Unknown fields are ignored.

```json
{
  "type": "<event_type>",
  "client_ts": 1716192000000
}
```

- `type` (required): one of `tab_blur`, `tab_focus`, `window_blur`, `window_focus`,
  `clipboard_copy`, `clipboard_paste`, `heartbeat`, `session_close`
- `client_ts` (optional): client-side Unix millisecond timestamp; stored as metadata only,
  not used for ordering or validation.
- `session_open` is sent by the **server** automatically on connection; the client MUST NOT
  send it.
- Clipboard events: the client MUST NOT include clipboard content. If any `content` field is
  present in a clipboard event, the server strips it before storage.

---

#### Server → Client Messages

**Acknowledgement** (after each accepted event):
```json
{
  "type": "ack",
  "sequence_num": 5,
  "received_at": "2026-05-20T10:05:30.123Z"
}
```

**Rate limit exceeded** (event not stored; connection remains open):
```json
{
  "type": "error",
  "code": "rate_limit_exceeded",
  "message": "Too many events. Slow down."
}
```

**Session closed by server** (deadline reached or timeout detected):
```json
{
  "type": "session_closed",
  "reason": "deadline",
  "closed_at": "2026-05-20T12:00:00Z"
}
```
`reason` values: `"deadline"`, `"timeout"`, `"submission"`.
After this message the server closes the WebSocket with close code 1000.

**Late event rejected** (session already closed when event arrives):
```json
{
  "type": "error",
  "code": "session_closed",
  "message": "Session is no longer active."
}
```

---

#### Rate Limiting

- 30 events per 60-second window per connection (in-memory per goroutine).
- Excess events receive an error frame; they are NOT stored; the connection stays open.

---

#### Heartbeat Protocol

- Client sends `{"type": "heartbeat"}` every 30 seconds.
- Server stores the event and updates `exam_sessions.last_heartbeat_at`.
- If no heartbeat is received for > 5 minutes during an active session (before the deadline),
  the server closes the session with status `timed_out`.

---

## Trace Retrieval (Docente)

### GET `/submissions/:sid/trace`

**Role**: `docente`, `co_docente` (with active grant for the activity)

Returns all trace events for the session linked to the given submission, in ascending
sequence order.

**Path parameters**:
- `sid` — UUID of the submission

**Responses**:

| Status | Body | Condition |
|---|---|---|
| 200 | `TraceResponse` | Events returned |
| 404 | `{"error": "not_found"}` | Submission not found or no linked session |
| 403 | `{"error": "forbidden"}` | Docente does not own the course / co-docente lacks grant |

**TraceResponse**:
```json
{
  "session_id": "uuid",
  "student_id": "uuid",
  "actividad_id": "uuid",
  "status": "completed",
  "started_at": "2026-05-20T10:00:00Z",
  "closed_at": "2026-05-20T12:00:00Z",
  "event_count": 42,
  "events": [
    {
      "sequence_num": 1,
      "event_type": "session_open",
      "received_at": "2026-05-20T10:00:00.123Z",
      "metadata": null,
      "event_hash": "a3f2..."
    },
    {
      "sequence_num": 2,
      "event_type": "tab_blur",
      "received_at": "2026-05-20T10:05:10.456Z",
      "metadata": null,
      "event_hash": "b91c..."
    }
  ]
}
```

Events are ordered by `sequence_num ASC`. `metadata` may be null or a JSON object. Clipboard
content is never present.

---

### GET `/submissions/:sid/trace/verify`

**Role**: `docente`, `co_docente` (with active grant for the activity)

Recomputes the full hash chain from scratch and reports whether it matches the stored hashes.

**Responses**:

| Status | Body | Condition |
|---|---|---|
| 200 | `{"valid": true, "event_count": 42}` | Chain verified |
| 200 | `{"valid": false, "first_invalid_sequence": 17, "event_count": 42}` | Tampering detected at seq 17 |
| 404 | `{"error": "not_found"}` | |
| 403 | `{"error": "forbidden"}` | |

---

## Error Response Format

All error responses use the same JSON envelope:
```json
{"error": "<error_code>"}
```

| Code | Meaning |
|---|---|
| `not_enrolled` | Student is not enrolled in the course |
| `trace_not_applicable` | Activity type does not require tracing |
| `session_closed` | Session is not active; events rejected |
| `rate_limit_exceeded` | Too many events in the rolling window |
| `forbidden` | Role or grant check failed |
| `not_found` | Requested resource does not exist |

---

## Endpoint Summary

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/actividades/:aid/session` | estudiante | Open or resume exam session |
| `GET` | `/actividades/:aid/session` | estudiante | Get current session status |
| `GET` | `/sessions/:sid/ws?token=<jwt>` | estudiante | WebSocket real-time event stream |
| `GET` | `/submissions/:sid/trace` | docente, co_docente | Retrieve full trace event list |
| `GET` | `/submissions/:sid/trace/verify` | docente, co_docente | Verify hash chain integrity |
