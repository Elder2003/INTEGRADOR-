# Data Model: MW5 — File Submissions

**Feature**: MW5 — File Submissions
**Date**: 2026-05-26
**Source**: `internal/submission/model.go` (nextlab-api)

---

## Frontend Types (to add to `src/types/academic.ts`)

### SubmissionResponse
Student-facing view of their own submission.

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID |
| `actividad_id` | `string` | UUID of the actividad |
| `original_filename` | `string` | Filename as uploaded |
| `file_size_bytes` | `number` | File size in bytes |
| `mime_type` | `string` | MIME type (e.g., `application/pdf`) |
| `submitted_at` | `string` | ISO 8601 timestamp |
| `is_late` | `boolean` | True if uploaded after deadline |
| `edit_window_expires_at` | `string \| null` | ISO 8601; null if locked from the start (late submission) |
| `is_locked` | `boolean` | True if edit window has expired or never opened |
| `last_replaced_at` | `string \| null` | ISO 8601 or null |

**State transitions**:
- No submission → after upload → `SubmissionResponse` (not locked)
- Not locked → after replace → updated `SubmissionResponse` (same id, new file)
- Not locked → after delete → no submission (404)
- Not locked → after `edit_window_expires_at` → locked (`is_locked = true`)
- Locked → read-only forever (no client action possible)

---

### StudentInfo
Embedded in `SubmissionListItem` for docente display.

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID |
| `name` | `string` | Student full name |
| `email` | `string` | Student email |

---

### SubmissionListItem
Docente-facing row in the submissions list.

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID |
| `student` | `StudentInfo` | Embedded student identity |
| `original_filename` | `string` | |
| `file_size_bytes` | `number` | |
| `mime_type` | `string` | |
| `submitted_at` | `string` | ISO 8601 |
| `is_late` | `boolean` | |
| `is_locked` | `boolean` | |
| `last_replaced_at` | `string \| null` | |
| `edit_window_expires_at` | `string \| null` | |

---

### SubmissionListResponse
Response envelope for `GET /cursos/:cid/actividades/:aid/submissions`.

| Field | Type | Description |
|-------|------|-------------|
| `actividad_id` | `string` | UUID |
| `submissions` | `SubmissionListItem[]` | All submissions for this activity |
| `total` | `number` | Count |

---

### ConfigChangeActor
Actor embedded in config change records.

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID |
| `name` | `string` | Full name |

---

### ConfigChangeResponse
Single activity config change audit record.

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID |
| `changed_by` | `ConfigChangeActor` | Who made the change |
| `field_name` | `string` | Name of field changed (e.g., `"deadline"`) |
| `previous_value` | `string` | Old value as string |
| `new_value` | `string` | New value as string |
| `changed_at` | `string` | ISO 8601 timestamp |

---

### ConfigChangesResponse
Response envelope for `GET /cursos/:cid/actividades/:aid/config-changes`.

| Field | Type | Description |
|-------|------|-------------|
| `actividad_id` | `string` | UUID |
| `changes` | `ConfigChangeResponse[]` | All config changes for this activity |

---

## Key Constraints (from backend source)

- **One submission per student per activity** — enforced at DB level (unique constraint). Upload returns `409 Conflict` if one already exists.
- **Max file size**: 52 428 800 bytes (50 MB) — enforced in `service.go`.
- **Edit window**: 30 minutes from upload, capped at the activity deadline if late submissions are not allowed. `null` if submission is late (no edit window).
- **Lock**: `is_locked = true` when `edit_window_expires_at` is null OR `now > edit_window_expires_at`.
- **Valid tipos**: `practica`, `tarea`, `informe`, `examen`, `defensa_oral` — all five accept file submissions.
