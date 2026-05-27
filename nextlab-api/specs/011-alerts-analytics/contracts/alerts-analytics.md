# API Contracts: Alerts & Analytics (M11)

All endpoints require JWT authentication unless noted. Error shape:
```json
{ "error": "code", "message": "human-readable", "correlation_id": "string" }
```

---

## US1 — Real-Time Alert Stream

### POST /alerts/stream-token

Issues a short-lived one-time SSE auth token.

**Auth**: JWT (any docente/co-docente role)

**Request**: empty body

**Response 200**:
```json
{ "token": "550e8400-e29b-41d4-a716-446655440000" }
```

**Response 401**: invalid or expired JWT

---

### GET /alerts/stream?token=\<value\>

Opens a persistent SSE connection. The browser `EventSource` sends `Last-Event-ID` automatically on reconnect; the server replays missed alerts before switching to live events.

**Auth**: `?token=<sse_token>` (one-time, 60 s TTL — consumed on first use)

**Response**: `Content-Type: text/event-stream`

**SSE event format**:
```
id: 42
event: submission_received
data: {"alert_id":42,"actividad_id":"uuid","student_id":"uuid","payload":{...},"advisory":false,"created_at":"RFC3339"}

```

**Event types**: `submission_received`, `late_submission`, `format_violation`, `integrity_flag`, `git_analysis_done`

**Response 401**: invalid, expired, or already-used token

---

### GET /alerts

Returns paginated list of alerts for the authenticated docente.

**Auth**: JWT

**Query params**:
- `since_id` (optional, integer) — return only alerts with id > since_id
- `unread_only` (optional, boolean, default false)
- `limit` (optional, integer, default 50, max 200)

**Response 200**:
```json
{
  "alerts": [
    {
      "id": 42,
      "actividad_id": "uuid",
      "student_id": "uuid",
      "type": "submission_received",
      "payload": {},
      "advisory": false,
      "is_read": false,
      "created_at": "2026-05-23T10:00:00Z"
    }
  ],
  "total": 100
}
```

---

### PATCH /alerts/:id/read

Marks a single alert as read.

**Auth**: JWT (docente who owns the alert)

**Request**: empty body

**Response 200**:
```json
{ "ok": true }
```

**Response 403**: alert belongs to a different docente
**Response 404**: alert not found

---

## US2 — Per-Activity Analytics

### GET /actividades/:actividadId/analytics

Returns aggregate statistics for a single activity.

**Auth**: JWT (docente who owns the activity)

**Response 200**:
```json
{
  "actividad_id": "uuid",
  "submission_count": 18,
  "pending_count": 5,
  "late_count": 3,
  "flagged_count": 2,
  "grade_distribution": {
    "range_0_49": 1,
    "range_50_69": 4,
    "range_70_89": 8,
    "range_90_100": 5
  },
  "avg_grade": 74.2
}
```

**Notes**:
- `pending_count` = enrolled students who have not submitted
- `flagged_count` = count of `integrity_flag` or `format_violation` alerts for this activity
- `avg_grade` is `null` when no grades have been assigned
- Grade distribution counts only graded submissions (ungraded are excluded from buckets)

**Response 403**: activity belongs to a different docente
**Response 404**: activity not found

---

## US3 — Course-Level Analytics

### GET /cursos/:cursoId/analytics

Returns aggregate statistics across all activities in a course.

**Auth**: JWT (docente who owns the course)

**Response 200**:
```json
{
  "curso_id": "uuid",
  "activities": [
    {
      "actividad_id": "uuid",
      "submission_count": 18,
      "pending_count": 5,
      "late_count": 3,
      "flagged_count": 2,
      "grade_distribution": { "range_0_49": 1, "range_50_69": 4, "range_70_89": 8, "range_90_100": 5 },
      "avg_grade": 74.2
    }
  ],
  "total_submissions": 54,
  "avg_grade": 71.8,
  "flag_rate": 0.037
}
```

**Notes**:
- `flag_rate` = total flagged alerts / total submissions (0.0–1.0); `null` if no submissions
- `avg_grade` is `null` when no grades assigned across the course

**Response 403**: course belongs to a different docente
**Response 404**: course not found

---

## Alert Writing (internal — not HTTP)

These are in-process calls made by the submission handler and async workers. They are not HTTP endpoints but are part of the contract between M11 and prior milestones.

| Caller | Alert type | When |
|---|---|---|
| `internal/submission` HTTP handler | `submission_received` | After file accepted and stored |
| `internal/projects` stage submit handler | `submission_received` + `late_submission` | After stage submission; `late_submission` added if `is_late=true` |
| `internal/format` worker | `format_violation` | After format analysis marks non-compliant |
| `internal/detection` worker | `integrity_flag` | After detection engine raises ≥1 flag |
| `internal/projects` git worker | `git_analysis_done` | After git analysis completes (success or failed) |
