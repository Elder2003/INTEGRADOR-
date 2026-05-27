# API Contracts: MW7 — Grading

**Feature**: MW7 — Grading
**Date**: 2026-05-26
**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)
**Auth**: Bearer token via `Authorization` header (all endpoints)

---

## Docente — Grade Management

### POST /submissions/:sid/grade
Create a new grade for a submission (no existing grade).

**Role**: `docente`

**Request body** (`application/json`):
```json
{
  "score": 85,
  "feedback": "Buen trabajo, pero le falta profundidad en la sección de conclusiones."
}
```

**Response `201 Created`**:
```json
{
  "id": "uuid",
  "submission_id": "uuid",
  "score": 85,
  "feedback": "Buen trabajo, pero le falta profundidad en la sección de conclusiones.",
  "graded_at": "2026-06-01T14:30:00Z",
  "graded_by": "uuid"
}
```

**Error responses**:
- `400` `{ "error": "score_out_of_range" }` — score not in 0–100
- `403` `{ "error": "forbidden" }` — not the activity's docente
- `404` `{ "error": "not_found" }` — submission not found
- `409` `{ "error": "grade_already_exists" }` — use PATCH to update

---

### PATCH /submissions/:sid/grade
Update an existing grade (overwrite score and/or feedback).

**Role**: `docente`

**Request body** (`application/json`):
```json
{
  "score": 90,
  "feedback": "Revisado — correcciones bien aplicadas."
}
```

**Response `200 OK`**: Same shape as POST response — full `GradeDecision` object.

**Error responses**:
- `400` `{ "error": "score_out_of_range" }` — score not in 0–100
- `403` `{ "error": "forbidden" }` — not the activity's docente
- `404` `{ "error": "not_found" }` — submission or grade not found

---

## Student — Exam Submission

### POST /actividades/:aid/submission/exam
Submit the student's exam answers as a single free-form text block.

**Role**: `estudiante`

**Request body** (`application/json`):
```json
{
  "answer": "Respuesta 1: ...\n\nRespuesta 2: ..."
}
```

**Response `201 Created`**: Full `SubmissionResponse` with `exam_answer` populated.
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "original_filename": null,
  "file_size_bytes": 0,
  "mime_type": "text/plain",
  "submitted_at": "2026-06-01T11:00:00Z",
  "is_late": false,
  "edit_window_expires_at": null,
  "is_locked": true,
  "last_replaced_at": null,
  "grade": null,
  "stage": null,
  "exam_answer": "Respuesta 1: ...\n\nRespuesta 2: ..."
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not enrolled
- `409` `{ "error": "exam_already_submitted" }` — student already submitted (locked)
- `422` `{ "error": "activity_not_exam_type" }` — activity tipo !== 'examen'
- `422` `{ "error": "trace_session_required" }` — trace_enabled but no active session

---

## Docente — Investigación Stage

### PATCH /submissions/:sid/stage
Set or update the investigación stage for an `informe` submission.

**Role**: `docente`

**Request body** (`application/json`):
```json
{
  "stage": "desarrollo"
}
```

**Valid values**: `"propuesta"`, `"desarrollo"`, `"revisión"`, `"aprobado"`

**Response `200 OK`**:
```json
{
  "stage": "desarrollo"
}
```

**Error responses**:
- `400` `{ "error": "invalid_stage" }` — value not in the allowed enum
- `403` `{ "error": "forbidden" }` — not the activity's docente
- `404` `{ "error": "not_found" }` — submission not found
- `422` `{ "error": "activity_not_informe_type" }` — activity tipo !== 'informe'

---

## Docente — Defensa Oral Scheduling

### GET /cursos/:cid/actividades/:aid/defensa-slots
Retrieve all submissions for a `defensa_oral` activity paired with their assigned slots.

**Role**: `docente`

**Response `200 OK`**:
```json
{
  "actividad_id": "uuid",
  "entries": [
    {
      "submission": {
        "id": "uuid",
        "student": { "id": "uuid", "name": "Ana López", "email": "ana@unifranz.edu.bo" },
        "original_filename": null,
        "file_size_bytes": 0,
        "mime_type": null,
        "submitted_at": "2026-06-01T08:00:00Z",
        "is_late": false,
        "is_locked": false,
        "last_replaced_at": null,
        "edit_window_expires_at": null,
        "session_id": null,
        "grade": null,
        "stage": null
      },
      "slot": {
        "submission_id": "uuid",
        "scheduled_at": "2026-06-10T09:00:00Z",
        "location": "Aula 301"
      }
    },
    {
      "submission": { "..." : "..." },
      "slot": null
    }
  ]
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not the activity's docente
- `422` `{ "error": "activity_not_defensa_type" }` — activity tipo !== 'defensa_oral'

---

### PUT /submissions/:sid/defensa-slot
Assign or update a defensa slot for a submission. Idempotent: creates if absent, overwrites if present.

**Role**: `docente`

**Request body** (`application/json`):
```json
{
  "scheduled_at": "2026-06-10T09:00:00Z",
  "location": "Aula 301"
}
```

**Response `200 OK`**:
```json
{
  "submission_id": "uuid",
  "scheduled_at": "2026-06-10T09:00:00Z",
  "location": "Aula 301"
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not the activity's docente
- `404` `{ "error": "not_found" }` — submission not found
- `409` `{ "error": "slot_conflict" }` — scheduled_at conflicts with another student's slot (backend validates)

---

## Student — View Defensa Slot

### GET /actividades/:aid/defensa-slot/me
Retrieve the authenticated student's assigned defensa slot for an activity.

**Role**: `estudiante`

**Response `200 OK`** (slot assigned):
```json
{
  "submission_id": "uuid",
  "scheduled_at": "2026-06-10T09:00:00Z",
  "location": "Aula 301"
}
```

**Response `200 OK`** (no slot yet):
```json
null
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — not enrolled
- `404` `{ "error": "not_found" }` — student has no submission for this activity

---

## `src/api/grading.ts` Function Signatures

```typescript
// Docente — grade management
createGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>
updateGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>

// Student — exam
submitExamAnswer(aid: string, answer: string): Promise<SubmissionResponse>

// Docente — stage
updateStage(sid: string, stage: InvestigaciónStage): Promise<{ stage: InvestigaciónStage }>

// Docente — defensa scheduling
listDefensaSlots(cid: string, aid: string): Promise<DefensaSlotsResponse>
assignDefensaSlot(sid: string, scheduledAt: string, location: string | null): Promise<DefensaSlot>

// Student — defensa slot
getMyDefensaSlot(aid: string): Promise<DefensaSlot | null>
```

`createGrade` → `POST /submissions/:sid/grade` (201)
`updateGrade` → `PATCH /submissions/:sid/grade` (200)
`submitExamAnswer` → `POST /actividades/:aid/submission/exam` (201)
`updateStage` → `PATCH /submissions/:sid/stage` (200)
`listDefensaSlots` → `GET /cursos/:cid/actividades/:aid/defensa-slots` (200)
`assignDefensaSlot` → `PUT /submissions/:sid/defensa-slot` (200)
`getMyDefensaSlot` → `GET /actividades/:aid/defensa-slot/me` (200, can return null)
