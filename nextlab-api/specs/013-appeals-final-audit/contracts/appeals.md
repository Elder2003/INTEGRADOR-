# API Contracts: Appeals & Final Audit (M13)

## Authentication

All endpoints require `Authorization: Bearer <jwt>`. Role is extracted from JWT claims (`user_role`).

---

## POST /appeals

File a new appeal. Student role only.

**Request**
```json
{
  "session_id": "uuid",
  "type": "analysis_dispute | grade_dispute",
  "reason": "string (required, non-empty)"
}
```

**Responses**

| Status | Body | Condition |
|--------|------|-----------|
| 201 | `Appeal` object | Created |
| 400 | `{error, message}` | Missing/invalid fields |
| 403 | `{error: "forbidden"}` | Caller is not the session's student |
| 404 | `{error: "not_found"}` | Session not found |
| 409 | `{error: "conflict"}` | An open appeal already exists for this session |

---

## GET /appeals/:id

Get a single appeal by ID.

**Responses**

| Status | Body | Condition |
|--------|------|-----------|
| 200 | `Appeal` object | Found |
| 403 | `{error: "forbidden"}` | Not the student, activity's docente, or admin |
| 404 | `{error: "not_found"}` | Not found |

---

## GET /appeals

List appeals. Results are scoped by caller role.

**Query params**: `status` (optional), `session_id` (optional, student only)

**Role behaviour**:
- **student** → only their own appeals
- **docente** → appeals for their activities
- **admin** → all appeals; `?status=escalated` filters

**Response**
```json
{ "appeals": [Appeal], "total": int }
```

---

## GET /actividades/:aid/appeals

List pending appeals for a specific actividad. Docente/admin only.

**Response**
```json
{ "actividad_id": "uuid", "appeals": [Appeal], "total": int }
```

---

## PUT /appeals/:id/review

Docente reviews a pending appeal (upheld or rejected).

**Request**
```json
{
  "decision": "upheld | rejected",
  "justification": "string (required)"
}
```

**Responses**

| Status | Body | Condition |
|--------|------|-----------|
| 200 | Updated `Appeal` + `AppealDecision` | Resolved |
| 200 | Updated `Appeal` (status=escalated) | Auto-escalated: `grade_dispute` where docente owns the graded session |
| 400 | `{error}` | Missing justification |
| 403 | `{error: "forbidden"}` | Not the activity's docente |
| 409 | `{error: "already_resolved"}` | Appeal not in pending state |

---

## PUT /appeals/:id/escalate

Docente escalates a pending appeal to admin.

**Request**
```json
{ "notes": "string (optional)" }
```

**Responses**

| Status | Body | Condition |
|--------|------|-----------|
| 200 | Updated `Appeal` (status=escalated) | OK |
| 403 | `{error: "forbidden"}` | Not the activity's docente |
| 409 | `{error: "already_resolved"}` | Not in pending state |

---

## PUT /appeals/:id/admin-review

Admin issues a final decision on an escalated appeal.

**Request**
```json
{
  "decision": "upheld | rejected | overridden",
  "justification": "string (required)"
}
```

**Responses**

| Status | Body | Condition |
|--------|------|-----------|
| 200 | Updated `Appeal` + `AppealDecision` | admin_resolved |
| 400 | `{error}` | Missing justification |
| 403 | `{error: "forbidden"}` | Not admin |
| 409 | `{error: "already_resolved"}` | Not in escalated state |

---

## GET /appeals/:id/evidence

Return the hash-chained audit evidence for an appeal. Admin or owning docente only.

**Response**
```json
{
  "appeal_id": "uuid",
  "valid": true,
  "events": [
    {
      "id": "uuid",
      "appeal_id": "uuid",
      "sequence": 1,
      "event_type": "submission_received",
      "payload": { "session_id": "...", "submitted_at": "...", "original_filename": "..." },
      "prev_hash": null,
      "event_hash": "abc123...",
      "occurred_at": "2026-05-23T10:00:00Z"
    }
  ]
}
```

`valid` is `true` if all `event_hash` values match the canonical computation. A hash mismatch sets `valid: false` and indicates the sequence position of the first broken link.

| Status | Condition |
|--------|-----------|
| 200 | Found |
| 403 | Not admin or owning docente |
| 404 | Appeal not found |

---

## GET /sessions/:sessionId/grade-log

Get the grade decision for a session. Docente (owning) or admin only. Lives in the detection package.

**Response**
```json
{
  "session_id": "uuid",
  "final_grade": 85.5,
  "suggested_grade_range": "80-90",
  "override_reason": "Student demonstrated original work in oral defense",
  "decided_by": "uuid",
  "decided_at": "2026-05-23T11:00:00Z"
}
```

| Status | Condition |
|--------|-----------|
| 200 | Grade decision exists |
| 404 | No grade decision yet |
| 403 | Not owning docente or admin |
