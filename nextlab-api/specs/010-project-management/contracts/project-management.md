# API Contracts: M10 — Project Management

**Base path**: `/api/v1`  
**Auth**: Bearer JWT in `Authorization` header. Role embedded in token claims (`role: docente | student`).

---

## Investigacion Stages (US1)

### POST `/actividades/:actividadId/stages`
Create a stage on an `investigacion` actividad. Docente only.

**Request**
```json
{
  "name": "Propuesta de tema",
  "display_order": 1,
  "deadline": "2026-06-15T23:59:00Z",
  "description": "Entregar una propuesta de 1 página."
}
```

**Response 201**
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "name": "Propuesta de tema",
  "display_order": 1,
  "deadline": "2026-06-15T23:59:00Z",
  "description": "Entregar una propuesta de 1 página.",
  "created_at": "2026-05-22T10:00:00Z"
}
```

**Errors**
- `400` — missing `name` or `deadline`, or actividad is not tipo `investigacion`
- `403` — caller is not the docente of the curso
- `409` — stage with same name already exists on this actividad

---

### GET `/actividades/:actividadId/stages`
List stages in display order. Docente and enrolled students.

**Response 200**
```json
{
  "stages": [
    {
      "id": "uuid",
      "name": "Propuesta de tema",
      "display_order": 1,
      "deadline": "2026-06-15T23:59:00Z",
      "description": "Entregar una propuesta de 1 página."
    }
  ]
}
```

---

### PUT `/actividades/:actividadId/stages/:stageId`
Update stage name, order, deadline, or description. Docente only.

**Request** — any subset of fields:
```json
{
  "name": "Propuesta revisada",
  "deadline": "2026-06-20T23:59:00Z"
}
```

**Response 200** — updated stage object (same shape as POST 201).

**Errors**
- `403` — not docente
- `404` — stage not found
- `409` — name conflict

---

### DELETE `/actividades/:actividadId/stages/:stageId`
Delete a stage. Blocked if any submission exists. Docente only.

**Response 204** — no body.

**Errors**
- `403` — not docente
- `404` — not found
- `409` — `{ "error": "stage_has_submissions" }` (FR-006)

---

### POST `/actividades/:actividadId/stages/:stageId/submit`
Student submits a file to a stage. Multipart form upload. Enrolled students only.

**Request**: `multipart/form-data`, field `file`.

**Response 201**
```json
{
  "id": "uuid",
  "stage_id": "uuid",
  "student_id": "uuid",
  "original_filename": "propuesta.pdf",
  "is_late": false,
  "submitted_at": "2026-06-10T14:32:00Z"
}
```

**Errors**
- `400` — no file uploaded
- `403` — caller is not an enrolled student
- `404` — stage not found

Note: Re-submission overwrites the prior file (UPSERT). A late submission is accepted with `is_late: true` (Principle VII).

---

### GET `/actividades/:actividadId/stage-progress`
Docente view: per-student stage status matrix. Docente only.

**Response 200**
```json
{
  "actividad_id": "uuid",
  "stages": [
    { "id": "uuid", "name": "Propuesta de tema", "deadline": "2026-06-15T23:59:00Z" }
  ],
  "students": [
    {
      "student_id": "uuid",
      "student_name": "Ana García",
      "stages": [
        { "stage_id": "uuid", "status": "submitted", "submitted_at": "2026-06-10T14:32:00Z", "is_late": false, "grade": null },
        { "stage_id": "uuid", "status": "pending", "submitted_at": null, "is_late": false, "grade": null },
        { "stage_id": "uuid", "status": "submitted_late", "submitted_at": "2026-07-02T08:00:00Z", "is_late": true, "grade": 75.0 }
      ]
    }
  ]
}
```

---

### PUT `/actividades/:actividadId/stages/:stageId/submissions/:studentId/grade`
Docente assigns grade and optional note to a specific stage submission.

**Request**
```json
{
  "grade": 88.5,
  "grade_note": "Buen planteamiento, falta más bibliografía."
}
```

**Response 200**
```json
{
  "stage_id": "uuid",
  "student_id": "uuid",
  "grade": 88.5,
  "grade_note": "Buen planteamiento, falta más bibliografía.",
  "graded_at": "2026-06-18T11:00:00Z"
}
```

**Errors**
- `400` — grade out of 0–100 range
- `403` — not docente
- `404` — submission not found (student never submitted to this stage)

---

## Git Analysis (US2)

### POST `/actividades/:actividadId/git-submit`
Student submits a Git repository URL. Enrolled students only.

**Request**
```json
{
  "repo_url": "https://github.com/student/my-project"
}
```

**Response 202** — accepted; analysis job enqueued.
```json
{
  "git_submission_id": "uuid",
  "actividad_id": "uuid",
  "student_id": "uuid",
  "repo_url": "https://github.com/student/my-project",
  "submitted_at": "2026-06-01T09:00:00Z",
  "analysis_status": "pending"
}
```

**Errors**
- `400` — missing or malformed `repo_url`
- `403` — not enrolled student
- `404` — actividad not found or not tipo `git_analysis`

Note: Re-submission (same student, same actividad) replaces the previous URL and enqueues a new analysis job.

---

### GET `/actividades/:actividadId/git-report/:studentId`
Docente retrieves the Git analysis report for a student. Docente only (FR-013; student gets 403).

**Response 200**
```json
{
  "git_submission_id": "uuid",
  "student_id": "uuid",
  "repo_url": "https://github.com/student/my-project",
  "analysis": {
    "status": "success",
    "commit_count": 47,
    "authors": ["Ana García", "ana.garcia@gmail.com"],
    "first_commit_at": "2026-03-01T10:00:00Z",
    "last_commit_at": "2026-06-01T08:55:00Z",
    "deadline_rush_pct": 72.3,
    "consistency_score": 34,
    "flags": ["deadline_rush"],
    "llm_advisory": "The repository shows a concentrated commit pattern...",
    "llm_status": "success",
    "fail_reason": null
  },
  "grade": null,
  "override_reason": ""
}
```

If analysis is still running:
```json
{ "analysis": { "status": "pending" } }
```

If clone failed:
```json
{ "analysis": { "status": "failed", "fail_reason": "clone_error" } }
```

**Errors**
- `403` — caller is a student (FR-013)
- `404` — submission not found

---

### POST `/actividades/:actividadId/git-grade/:studentId`
Docente assigns a final grade after reviewing the Git analysis. Docente only.

**Request**
```json
{
  "grade": 65.0,
  "override_reason": "Student explained the late commits were due to a merge."
}
```

`override_reason` is optional; required only when grade differs significantly from LLM advisory range (same rule as M8).

**Response 200**
```json
{
  "git_submission_id": "uuid",
  "student_id": "uuid",
  "grade": 65.0,
  "override_reason": "Student explained the late commits were due to a merge.",
  "graded_at": "2026-06-05T14:00:00Z"
}
```

**Errors**
- `400` — grade out of 0–100; `override_reason` missing when required
- `403` — not docente
- `404` — analysis not found or still pending

---

## Oral Defense (US3)

### PUT `/actividades/:actividadId/defensa/:studentId/schedule`
Docente sets (or replaces) a student's defense appointment. Docente only.

**Request**
```json
{
  "scheduled_at": "2026-07-10T10:00:00Z"
}
```

**Response 200**
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "student_id": "uuid",
  "scheduled_at": "2026-07-10T10:00:00Z",
  "grade": null,
  "graded_at": null
}
```

**Errors**
- `400` — `scheduled_at` missing or invalid
- `403` — not docente
- `404` — student not enrolled in the actividad's curso

---

### GET `/actividades/:actividadId/defensa/:studentId`
Get a student's defense appointment. Student sees their own; docente sees any.

**Response 200**
```json
{
  "actividad_id": "uuid",
  "student_id": "uuid",
  "scheduled_at": "2026-07-10T10:00:00Z",
  "grade": null
}
```

If no appointment scheduled yet:
```json
{ "actividad_id": "uuid", "student_id": "uuid", "scheduled_at": null, "grade": null }
```

**Errors**
- `403` — student requesting another student's appointment

---

### PUT `/actividades/:actividadId/defensa/:studentId/grade`
Docente records grade for a student's oral defense. Docente only.

**Request**
```json
{
  "grade": 82.0
}
```

**Response 200**
```json
{
  "actividad_id": "uuid",
  "student_id": "uuid",
  "grade": 82.0,
  "graded_at": "2026-07-10T11:15:00Z"
}
```

**Errors**
- `400` — grade out of 0–100
- `403` — not docente
- `404` — no defense appointment exists for this student

---

## Shared Error Shape

```json
{
  "error": "error_code",
  "message": "Human-readable description",
  "correlation_id": "uuid"
}
```
