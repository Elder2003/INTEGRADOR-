# API Contracts: MW6 — Tracing

**Feature**: MW6 — Tracing
**Date**: 2026-05-26
**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)
**Auth**: Bearer token via `Authorization` header (REST) or `?token=<jwt>` query param (WebSocket)

---

## Student — Session Lifecycle

### POST /actividades/:aid/sessions
Create a new monitoring session for the authenticated student.

**Role**: `estudiante`

**Response `201 Created`**:
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "student_id": "uuid",
  "started_at": "2026-06-01T10:00:00Z",
  "ended_at": null
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not enrolled
- `409` `{ "error": "session_already_active" }` — student already has an active session for this activity
- `422` `{ "error": "trace_not_enabled" }` — activity does not have trace_enabled

---

### WebSocket ws://[API_HOST]/sessions/:sid/ws?token=<jwt>
Stream trace events from the student to the backend in real time.

**Role**: `estudiante`
**Protocol**: WebSocket (RFC 6455)
**Auth**: `token` query parameter (only permitted WebSocket exception per constitution)

**Message format** (student → server, JSON):
```json
{
  "type": "keystroke | copy | paste | focus_in | focus_out",
  "timestamp": "2026-06-01T10:01:23.456Z",
  "content": "optional string, truncated to 500 chars"
}
```

**Server → client ping**: Server sends `{"type":"ping"}` every 30 seconds. Client must respond with `{"type":"pong"}` to keep the connection alive.

**Connection close**: Server closes with code `1000` (normal) when session is ended server-side.

**Error codes**:
- `4001` — invalid or expired token
- `4003` — session not found or not owned by token holder
- `4004` — session already ended

---

### POST /sessions/:sid/end
Explicitly end an active session.

**Role**: `estudiante`

**Response `204 No Content`**

**Error responses**:
- `403` `{ "error": "forbidden" }` — not the session owner
- `404` `{ "error": "not_found" }` — session not found
- `409` `{ "error": "session_already_ended" }` — already ended

---

## Docente & Co-docente — Trace Review

### GET /submissions/:subid/trace
Retrieve the complete historical trace log for a submission's linked session.

**Role**: `docente`, `co_docente`

**Response `200 OK`**:
```json
{
  "session_id": "uuid",
  "events": [
    {
      "id": "uuid",
      "session_id": "uuid",
      "type": "paste",
      "timestamp": "2026-06-01T10:01:23.456Z",
      "content": "pasted text here"
    }
  ],
  "total": 42
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — submission not in caller's scope
- `404` `{ "error": "not_found" }` — no session linked to this submission

---

### GET /cursos/:cid/actividades/:aid/sessions/active
List all currently active sessions for an activity.

**Role**: `docente`, `co_docente`

**Response `200 OK`**:
```json
{
  "actividad_id": "uuid",
  "sessions": [
    {
      "session_id": "uuid",
      "student": { "id": "uuid", "name": "Juan Pérez", "email": "juan@unifranz.edu.bo" },
      "started_at": "2026-06-01T10:00:00Z",
      "event_count": 127
    }
  ]
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not authorized for this actividad

---

### GET /sessions/:sid/events?since=<iso-timestamp>
Retrieve events for a session since a given timestamp (used for live polling).

**Role**: `docente`, `co_docente`
**Query param**: `since` (optional ISO 8601 timestamp; if omitted, returns all events)

**Response `200 OK`**: Same shape as `GET /submissions/:subid/trace` — `{ session_id, events, total }` but `events` filtered to those with `timestamp > since`.

**Error responses**:
- `403` `{ "error": "forbidden" }` — session not in caller's scope
- `404` `{ "error": "not_found" }` — session not found

---

## `src/api/tracing.ts` Function Signatures

```typescript
// Student
createSession(aid: string): Promise<SessionResponse>
endSession(sid: string): Promise<void>

// Docente / Co-docente
getTraceLog(submissionId: string): Promise<TraceLogResponse>
listActiveSessions(cid: string, aid: string): Promise<ActiveSessionsResponse>
getLiveEvents(sid: string, since?: string): Promise<TraceLogResponse>
```

`createSession` returns the full `SessionResponse` including the `id` used for the WebSocket URL.
`endSession` expects 204 and returns void; throws `ApiError` on 403/404/409.
`getTraceLog` throws `ApiError` with `not_found` when no session is linked to the submission.
`getLiveEvents` accepts an optional ISO timestamp; omit for initial load, pass the last event's timestamp for subsequent polls.
