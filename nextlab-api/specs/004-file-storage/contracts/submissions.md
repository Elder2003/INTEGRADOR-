# API Contract: M4 — File Storage

All endpoints require a valid JWT Bearer token unless noted. Role enforcement is identical to M3: middleware reads the `role` claim from the JWT and the service layer enforces data-scope rules.

---

## Student Submission Endpoints

### POST /actividades/:aid/submission
**Role**: `estudiante`
**Description**: Upload a file for an activity. Student must be enrolled in the course that owns this activity. Only one active submission is permitted per student per activity.

**Request**: `multipart/form-data`
| Field | Type | Required | Notes |
|---|---|---|---|
| `file` | file part | Yes | Max 50 MB. Any content type accepted. |

**Responses**:

`201 Created`
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "original_filename": "proyecto_v1.pdf",
  "file_size_bytes": 2048576,
  "mime_type": "application/pdf",
  "submitted_at": "2026-05-19T14:30:00Z",
  "is_late": false,
  "edit_window_expires_at": "2026-05-19T15:00:00Z",
  "is_locked": false
}
```

`400 Bad Request` — file missing or exceeds 50 MB
`403 Forbidden` — student not enrolled in course
`409 Conflict` — a submission already exists for this student+activity (use PUT to replace)
`422 Unprocessable Entity` — activity type does not accept submissions, or deadline passed with `late_submission_allowed = false`
`503 Service Unavailable` — storage backend unreachable

---

### PUT /actividades/:aid/submission
**Role**: `estudiante`
**Description**: Replace the active submission within the edit window. Rejected if the window has expired or if the submission is late.

**Request**: `multipart/form-data`
| Field | Type | Required | Notes |
|---|---|---|---|
| `file` | file part | Yes | Max 50 MB. |

**Responses**:

`200 OK`
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "original_filename": "proyecto_v2.pdf",
  "file_size_bytes": 2150400,
  "mime_type": "application/pdf",
  "submitted_at": "2026-05-19T14:45:00Z",
  "is_late": false,
  "edit_window_expires_at": "2026-05-19T15:00:00Z",
  "is_locked": false,
  "last_replaced_at": "2026-05-19T14:45:00Z"
}
```

`403 Forbidden` — student not enrolled, or edit window expired, or submission is late (locked on receipt)
`404 Not Found` — no active submission exists; use POST first
`422 Unprocessable Entity` — file missing or exceeds limit

---

### DELETE /actividades/:aid/submission
**Role**: `estudiante`
**Description**: Delete the active submission within the edit window. After deletion the student may re-upload (POST) which starts a fresh 30-minute window.

**Request**: empty body

**Responses**:

`204 No Content` — submission deleted; student may upload again

`403 Forbidden` — edit window expired or submission is late
`404 Not Found` — no active submission

---

### GET /actividades/:aid/submission
**Role**: `estudiante`
**Description**: Retrieve own submission metadata for an activity.

**Responses**:

`200 OK`
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "original_filename": "proyecto_v2.pdf",
  "file_size_bytes": 2150400,
  "mime_type": "application/pdf",
  "submitted_at": "2026-05-19T14:45:00Z",
  "is_late": false,
  "edit_window_expires_at": "2026-05-19T15:00:00Z",
  "is_locked": false,
  "last_replaced_at": "2026-05-19T14:45:00Z"
}
```

`404 Not Found` — no submission exists yet

---

### GET /actividades/:aid/submission/download
**Role**: `estudiante`
**Description**: Download the student's own submitted file. The API streams the file content from storage; no direct storage URL is exposed.

**Responses**:

`200 OK` — file stream with headers:
```
Content-Type: application/pdf          (or actual MIME type)
Content-Disposition: attachment; filename="proyecto_v2.pdf"
Content-Length: 2150400
```

`404 Not Found` — no submission exists

---

## Docente & Co-Docente Submission Endpoints

### GET /cursos/:cid/actividades/:aid/submissions
**Role**: `docente` (must own the course) | `co_docente` (must hold active grant for this activity)
**Description**: List all student submissions for an activity. Includes late flag, lock status, and file metadata for each submission.

**Responses**:

`200 OK`
```json
{
  "actividad_id": "uuid",
  "submissions": [
    {
      "id": "uuid",
      "student": {
        "id": "uuid",
        "name": "Ana Gutierrez",
        "email": "ana.gutierrez@unifranz.edu.bo"
      },
      "original_filename": "proyecto_v2.pdf",
      "file_size_bytes": 2150400,
      "mime_type": "application/pdf",
      "submitted_at": "2026-05-19T14:45:00Z",
      "is_late": false,
      "is_locked": true,
      "last_replaced_at": "2026-05-19T14:45:00Z"
    }
  ],
  "total": 1
}
```

`403 Forbidden` — docente does not own course, or co_docente has no active grant
`404 Not Found` — activity not found

---

### GET /submissions/:sid/download
**Role**: `docente` (must own the course that contains this submission's activity) | `co_docente` (must hold active grant for the activity)
**Description**: Download a specific student's submission file. The API streams the file from storage; no storage URL is exposed.

**Responses**:

`200 OK` — file stream with:
```
Content-Type: application/pdf
Content-Disposition: attachment; filename="proyecto_v2.pdf"
Content-Length: 2150400
```

`403 Forbidden` — insufficient access
`404 Not Found` — submission not found

---

## Activity Config Endpoints

### PUT /cursos/:cid/actividades/:aid
**Role**: `docente` (must own course)
**Description**: Extended from M3. Now also accepts `late_submission_allowed` in the request body. If the value changes, an audit log entry is written atomically.

**Request body** (all fields optional — only provided fields are updated):
```json
{
  "deadline": "2026-06-01T23:59:00Z",
  "instructions": "Updated instructions...",
  "late_submission_allowed": false
}
```

**Responses**:

`200 OK` — updated activity (includes `late_submission_allowed` in response)
`403 Forbidden` — docente does not own course
`409 Conflict` — title or tipo change attempted after submissions exist

---

### GET /cursos/:cid/actividades/:aid/config-changes
**Role**: `docente` (must own course) | `admin`
**Description**: Retrieve the immutable audit log of changes to this activity's configuration flags.

**Responses**:

`200 OK`
```json
{
  "actividad_id": "uuid",
  "changes": [
    {
      "id": "uuid",
      "changed_by": {
        "id": "uuid",
        "name": "Prof. Ramirez"
      },
      "field_name": "late_submission_allowed",
      "previous_value": "true",
      "new_value": "false",
      "changed_at": "2026-05-20T09:00:00Z"
    }
  ]
}
```

`403 Forbidden` — unauthorized role or docente does not own course

---

## Error Response Format

All error responses use the same envelope established in M2:

```json
{
  "error": "late_submissions_disabled",
  "message": "This activity does not accept submissions after the deadline."
}
```

---

## Summary Table

| Method | Path | Role | Description |
|---|---|---|---|
| `POST` | `/actividades/:aid/submission` | estudiante | Upload first file |
| `PUT` | `/actividades/:aid/submission` | estudiante | Replace within edit window |
| `DELETE` | `/actividades/:aid/submission` | estudiante | Remove within edit window |
| `GET` | `/actividades/:aid/submission` | estudiante | Get own submission metadata |
| `GET` | `/actividades/:aid/submission/download` | estudiante | Download own file |
| `GET` | `/cursos/:cid/actividades/:aid/submissions` | docente, co_docente | List all submissions |
| `GET` | `/submissions/:sid/download` | docente, co_docente | Download a specific file |
| `PUT` | `/cursos/:cid/actividades/:aid` | docente | Extended: now accepts `late_submission_allowed` |
| `GET` | `/cursos/:cid/actividades/:aid/config-changes` | docente, admin | Activity config audit log |
