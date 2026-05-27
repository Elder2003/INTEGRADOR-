# API Contracts: MW5 — File Submissions

**Feature**: MW5 — File Submissions
**Date**: 2026-05-26
**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)
**Auth**: Bearer token via `Authorization` header (injected by axios interceptor)

---

## Student — Submission Lifecycle

### POST /actividades/:aid/submission
Upload a file for the first time.

**Role**: `estudiante`
**Content-Type**: `multipart/form-data`
**Field**: `file` (the file)

**Response `201 Created`**:
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "original_filename": "trabajo.pdf",
  "file_size_bytes": 204800,
  "mime_type": "application/pdf",
  "submitted_at": "2026-06-01T10:00:00Z",
  "is_late": false,
  "edit_window_expires_at": "2026-06-01T10:30:00Z",
  "is_locked": false,
  "last_replaced_at": null
}
```

**Error responses**:
- `400` `{ "error": "missing_file" }` — multipart field missing
- `403` `{ "error": "forbidden" }` — not enrolled
- `409` `{ "error": "submission_exists" }` — already submitted
- `413` / `400` `{ "error": "file_too_large" }` — > 50 MB
- `422` `{ "error": "late_submissions_disabled" }` — deadline passed, late not allowed
- `422` `{ "error": "invalid_activity_type" }` — tipo not in valid set

---

### GET /actividades/:aid/submission
Fetch the student's current submission (if any).

**Role**: `estudiante`

**Response `200 OK`**: `SubmissionResponse` (same shape as POST 201)

**Error responses**:
- `404` `{ "error": "not_found" }` — no submission yet

---

### PUT /actividades/:aid/submission
Replace an existing submission within the edit window.

**Role**: `estudiante`
**Content-Type**: `multipart/form-data`
**Field**: `file`

**Response `200 OK`**: Updated `SubmissionResponse`

**Error responses**:
- `400` `{ "error": "missing_file" }`
- `400` `{ "error": "file_too_large" }`
- `403` `{ "error": "submission_locked" }` — edit window expired
- `404` `{ "error": "not_found" }` — no submission to replace

---

### DELETE /actividades/:aid/submission
Delete the student's submission within the edit window.

**Role**: `estudiante`

**Response `204 No Content`**

**Error responses**:
- `403` `{ "error": "submission_locked" }`
- `404` `{ "error": "not_found" }`

---

### GET /actividades/:aid/submission/download
Student downloads their own submission.

**Role**: `estudiante`

**Response `200 OK`**: Binary file stream
- `Content-Type`: MIME type of the file
- `Content-Disposition`: `attachment; filename="<original_filename>"`
- `Content-Length`: file size in bytes

**Error responses**:
- `404` `{ "error": "not_found" }`
- `503` `{ "error": "storage_unavailable" }`

---

## Docente & Co-docente — Submission Review

### GET /cursos/:cid/actividades/:aid/submissions
List all submissions for an activity.

**Role**: `docente`, `co_docente`
**Authorization**: Docente must own the actividad; co-docente must have a grant for it.

**Response `200 OK`**:
```json
{
  "actividad_id": "uuid",
  "submissions": [
    {
      "id": "uuid",
      "student": { "id": "uuid", "name": "Juan Pérez", "email": "juan@unifranz.edu.bo" },
      "original_filename": "trabajo.pdf",
      "file_size_bytes": 204800,
      "mime_type": "application/pdf",
      "submitted_at": "2026-06-01T10:00:00Z",
      "is_late": false,
      "is_locked": true,
      "last_replaced_at": null,
      "edit_window_expires_at": "2026-06-01T10:30:00Z"
    }
  ],
  "total": 1
}
```

**Error responses**:
- `403` `{ "error": "forbidden" }` — caller not authorized for this actividad

---

### GET /submissions/:sid/download
Docente or co-docente downloads a specific submission.

**Role**: `docente`, `co_docente`
**Path param**: `:sid` — submission UUID

**Response `200 OK`**: Binary file stream (same headers as student download)

**Error responses**:
- `403` `{ "error": "forbidden" }` — submission not in caller's scope
- `404` `{ "error": "not_found" }`
- `503` `{ "error": "storage_unavailable" }`

---

### GET /cursos/:cid/actividades/:aid/config-changes
List all config change audit records for an activity.

**Role**: `docente`, `co_docente`

**Response `200 OK`**:
```json
{
  "actividad_id": "uuid",
  "changes": [
    {
      "id": "uuid",
      "changed_by": { "id": "uuid", "name": "Dr. García" },
      "field_name": "deadline",
      "previous_value": "2026-05-30T23:59:00Z",
      "new_value": "2026-06-01T23:59:00Z",
      "changed_at": "2026-05-28T09:00:00Z"
    }
  ]
}
```

---

## `src/api/submissions.ts` Function Signatures

```typescript
// Student — submission lifecycle
getSubmission(aid: string): Promise<SubmissionResponse | null>
uploadSubmission(aid: string, file: File, onProgress?: (pct: number) => void): Promise<SubmissionResponse>
replaceSubmission(aid: string, file: File, onProgress?: (pct: number) => void): Promise<SubmissionResponse>
deleteSubmission(aid: string): Promise<void>
studentDownloadSubmission(aid: string, filename: string): Promise<void>

// Docente / Co-docente — review
listSubmissions(cid: string, aid: string): Promise<SubmissionListResponse>
downloadSubmission(sid: string, filename: string): Promise<void>
listConfigChanges(cid: string, aid: string): Promise<ConfigChangesResponse>
```

`getSubmission` returns `null` on 404 (no submission yet) and throws `ApiError` on other errors.
`studentDownloadSubmission` and `downloadSubmission` trigger browser download via blob URL; they do not return file data.
