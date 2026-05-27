# Data Model: MW6 — Tracing

**Feature**: MW6 — Tracing
**Date**: 2026-05-26

---

## New Types (add to `src/types/academic.ts`)

### `SessionResponse`
Returned by `POST /actividades/:aid/sessions` when the student starts a monitoring session.

```ts
export interface SessionResponse {
  id: string              // UUID — used as the WebSocket session ID
  actividad_id: string
  student_id: string
  started_at: string      // ISO 8601
  ended_at: string | null // null while session is active
}
```

### `TraceEventPayload`
Sent *to* the server via WebSocket by the student. Not stored as a frontend type in academic.ts — defined inline in the hook.

```ts
// Internal to useTracingSession hook (not exported from academic.ts)
interface TraceEventPayload {
  type: 'keystroke' | 'copy' | 'paste' | 'focus_in' | 'focus_out'
  timestamp: string       // ISO 8601
  content?: string        // For copy/paste: the copied/pasted text (truncated to 500 chars)
}
```

### `TraceLogEntry`
A single recorded event returned by the historical or polling endpoints (docente-facing).

```ts
export interface TraceLogEntry {
  id: string
  session_id: string
  type: 'keystroke' | 'copy' | 'paste' | 'focus_in' | 'focus_out'
  timestamp: string       // ISO 8601
  content: string | null  // null for keystroke/focus events
}
```

### `TraceLogResponse`
Returned by `GET /submissions/:subid/trace` (historical) and `GET /sessions/:sid/events` (polling).

```ts
export interface TraceLogResponse {
  session_id: string
  events: TraceLogEntry[]
  total: number
}
```

### `ActiveSessionEntry`
One entry in the live sessions list for an activity.

```ts
export interface ActiveSessionEntry {
  session_id: string
  student: StudentInfo    // re-uses existing StudentInfo from MW5
  started_at: string      // ISO 8601
  event_count: number
}
```

### `ActiveSessionsResponse`
Returned by `GET /cursos/:cid/actividades/:aid/sessions/active`.

```ts
export interface ActiveSessionsResponse {
  actividad_id: string
  sessions: ActiveSessionEntry[]
}
```

---

## Modified Type (update in `src/types/academic.ts`)

### `SubmissionListItem` — add `session_id` field

```ts
// Before (MW5):
export interface SubmissionListItem {
  id: string
  student: StudentInfo
  original_filename: string
  file_size_bytes: number
  mime_type: string
  submitted_at: string
  is_late: boolean
  is_locked: boolean
  last_replaced_at: string | null
  edit_window_expires_at: string | null
}

// After (MW6 — add one field):
export interface SubmissionListItem {
  // ...all existing fields...
  session_id: string | null  // null = no active session was linked at upload time
}
```

---

## Client-Side Session State Machine

The `useTracingSession` hook manages the following states (not a backend type — local React state):

```
idle → starting → connecting → active → ended
                            ↘ reconnecting → active
                                          ↘ error
```

| State | Description |
|---|---|
| `idle` | No session created; student hasn't confirmed start |
| `starting` | POST /sessions in flight |
| `connecting` | WebSocket handshake in progress |
| `active` | WebSocket open; events streaming |
| `reconnecting` | Connection dropped; retry in progress (≤3 attempts) |
| `ended` | Session explicitly ended or max reconnects reached cleanly |
| `error` | Unrecoverable error (3 reconnect attempts exhausted) |
