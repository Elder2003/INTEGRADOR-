# API Contracts: M8 — Detection Engine

All endpoints require a valid JWT (`Authorization: Bearer <token>`).
Student role is rejected at middleware before any DB query for both endpoints (FR-007, Principle IV).

---

## GET /sessions/:sessionID/detection-report

Returns the detection report for a session. Accessible to docentes scoped to the session's course.

### Authorization
- **Required role**: `docente` (or `admin`)
- **Scope check**: `exam_sessions.actividad_id → actividades.curso_id → cursos.docente_id` must match the authenticated user's ID. Returns `403` if the session belongs to a different docente's course.
- **Student role**: Always `403` regardless of session ownership.

### Path Parameters
| Param | Type | Description |
|---|---|---|
| sessionID | UUID | The exam session ID |

### Responses

**200 OK — both heuristic and LLM results available**
```json
{
  "session_id": "uuid",
  "student_id": "uuid",
  "actividad_id": "uuid",
  "heuristic": {
    "score": 42,
    "flagged_categories": ["tab_activity", "clipboard_activity"],
    "event_count": 87,
    "created_at": "2026-05-20T14:32:00Z"
  },
  "llm": {
    "advisory_text": "The submission shows adequate understanding of the topic...",
    "suggested_grade_range": "60–70",
    "backend_used": "local/llama3",
    "llm_status": "success",
    "created_at": "2026-05-20T14:33:15Z"
  },
  "llm_status": "success",
  "grade_decision": null
}
```

**200 OK — heuristic only (LLM failed or not yet complete)**
```json
{
  "session_id": "uuid",
  "student_id": "uuid",
  "actividad_id": "uuid",
  "heuristic": {
    "score": 15,
    "flagged_categories": ["tab_activity"],
    "event_count": 42,
    "created_at": "2026-05-20T14:32:00Z"
  },
  "llm": null,
  "llm_status": "unavailable",
  "grade_decision": null
}
```

**200 OK — grade already assigned (grade_decision present)**
```json
{
  "session_id": "uuid",
  "student_id": "uuid",
  "actividad_id": "uuid",
  "heuristic": { ... },
  "llm": { ... },
  "llm_status": "success",
  "grade_decision": {
    "final_grade": 55,
    "suggested_grade_range": "60–70",
    "override_reason": "Student demonstrated sufficient understanding during oral clarification",
    "decided_by": "uuid",
    "decided_at": "2026-05-20T15:00:00Z"
  }
}
```

**400 Bad Request** — sessionID is not a valid UUID
```json
{ "error": "invalid_request", "message": "sessionID must be a valid UUID" }
```

**403 Forbidden** — student role or wrong docente
```json
{ "error": "forbidden", "message": "access denied" }
```

**404 Not Found** — session does not exist
```json
{ "error": "not_found", "message": "session not found" }
```

**404 Not Found** — session exists but no heuristic result yet (detection not complete)
```json
{ "error": "not_found", "message": "detection report not yet available" }
```

---

## POST /sessions/:sessionID/grade

Assigns a final grade to a session. The grade decision is stored atomically in `grade_decisions`. If the grade falls outside the LLM-suggested range, `override_reason` is required.

### Authorization
- **Required role**: `docente` (or `admin`)
- **Scope check**: Same as GET detection-report above.
- **Student role**: Always `403`.

### Path Parameters
| Param | Type | Description |
|---|---|---|
| sessionID | UUID | The exam session ID |

### Request Body
```json
{
  "final_grade": 55,
  "override_reason": "Student demonstrated understanding in follow-up"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| final_grade | number | Yes | Numeric grade assigned by docente |
| override_reason | string | Conditional | Required when grade outside LLM-suggested range (see Decision 9) |

### Responses

**201 Created** — grade recorded
```json
{
  "session_id": "uuid",
  "final_grade": 55,
  "suggested_grade_range": "60–70",
  "override_reason": "Student demonstrated understanding in follow-up",
  "decided_by": "uuid",
  "decided_at": "2026-05-20T15:00:00Z"
}
```

**400 Bad Request** — missing `final_grade` or invalid type
```json
{ "error": "invalid_request", "message": "final_grade is required and must be a number" }
```

**409 Conflict** — grade already assigned for this session
```json
{ "error": "conflict", "message": "a grade decision already exists for this session" }
```

**422 Unprocessable Entity** — grade outside suggested range, no override reason
```json
{
  "error": "override_reason_required",
  "message": "final_grade 55 is outside the suggested range 60–70; override_reason is required"
}
```

**403 Forbidden** — student role or wrong docente
```json
{ "error": "forbidden", "message": "access denied" }
```

**404 Not Found** — session does not exist or detection report not ready
```json
{ "error": "not_found", "message": "session or detection report not found" }
```
