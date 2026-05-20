# API Contract: M3 — Academic Management

All protected endpoints require `Authorization: Bearer <access_token>` from M2.
Correlation IDs follow the same pattern as M2 (injected by middleware).

---

## Google OAuth — Student Authentication

### `GET /auth/google`

Initiates the Google OAuth flow. Generates a CSRF state token, stores it in Redis (60s TTL), and redirects to the Google consent screen.

**Auth**: None  
**Response**: `302 Redirect` → Google OAuth URL

---

### `GET /auth/google/callback?code=...&state=...`

Handles the Google OAuth callback. Validates state, exchanges the authorization code, verifies the ID token, looks up the student profile, syncs enrollment, and issues tokens.

**Auth**: None

**Success** `200 OK`:
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "base64url..."
}
```

**Errors**:
| Status | `error` | Condition |
|--------|---------|-----------|
| 400 | `invalid_state` | State missing, expired, or HMAC invalid |
| 400 | `oauth_exchange_failed` | Google rejected the authorization code |
| 403 | `account_not_provisioned` | Email not found in `student_profiles` |
| 403 | `account_inactive` | Student profile exists but `is_active = false` |
| 500 | `internal_error` | Unexpected failure |

---

## Admin — Student Profiles

### `POST /admin/student-profiles`

**Auth**: admin  
**Body**:
```json
{
  "email": "juan.perez@unifranz.edu.bo",
  "carrera_id": "uuid",
  "semester": 4,
  "student_code": "20221234"
}
```
**Success** `201 Created`: returns the created profile object.  
**Errors**: `400 validation_error` (invalid semester, missing fields), `409 conflict` (email already exists).

---

### `GET /admin/student-profiles`

**Auth**: admin  
**Query params**: `carrera_id` (filter), `semester` (filter), `page`, `limit`  
**Success** `200 OK`: paginated list of student profile objects.

---

### `PUT /admin/student-profiles/:id`

**Auth**: admin  
**Body**: any subset of `{ carrera_id, semester, student_code, is_active }`  
**Success** `200 OK`: updated profile object.  
**Note**: Changing `carrera_id` or `semester` does NOT automatically re-sync inscripciones. Re-sync occurs on the student's next login.

---

## Admin — Catalog

### `POST /admin/carreras`

**Auth**: admin  
**Body**: `{ "code": "IS", "name": "Ingeniería de Sistemas" }`  
**Success** `201 Created`: `{ id, code, name, is_active, created_at }`  
**Errors**: `400 validation_error`, `409 conflict` (duplicate code).

### `GET /admin/carreras`

**Auth**: admin  
**Success** `200 OK`: array of carrera objects (including inactive).

### `PUT /admin/carreras/:id`

**Auth**: admin  
**Body**: any subset of `{ name, is_active }`  
**Success** `200 OK`: updated carrera.

### `DELETE /admin/carreras/:id`

**Auth**: admin  
**Behavior**: Soft delete (`is_active = false`). Returns `204 No Content`.  
**Error**: `409 conflict` if carrera has active materias.

---

### `POST /admin/materias`

**Auth**: admin  
**Body**:
```json
{
  "carrera_id": "uuid",
  "code": "PROG3",
  "name": "Programación III",
  "semester": 4
}
```
**Success** `201 Created`: materia object.  
**Errors**: `400 validation_error` (semester out of range), `404 not_found` (carrera not found), `409 conflict` (duplicate code).

### `GET /admin/materias`

**Auth**: admin  
**Query params**: `carrera_id` (filter), `semester` (filter)  
**Success** `200 OK`: array of materia objects.

### `PUT /admin/materias/:id`

**Auth**: admin  
**Body**: any subset of `{ name, semester, is_active }`  
**Success** `200 OK`: updated materia.

### `DELETE /admin/materias/:id`

**Auth**: admin  
**Behavior**: Soft delete. `204 No Content`.  
**Error**: `409 conflict` if materia has active cursos.

---

## Docente — Cursos

### `POST /cursos`

**Auth**: docente  
**Body**:
```json
{
  "materia_id": "uuid",
  "semester_label": "2026-1",
  "section": "A"
}
```
**Success** `201 Created`:
```json
{
  "id": "uuid",
  "materia_id": "uuid",
  "materia_name": "Programación III",
  "docente_id": "uuid",
  "semester_label": "2026-1",
  "section": "A",
  "is_active": true,
  "created_at": "2026-05-19T..."
}
```
**Errors**: `404 not_found` (materia inactive or not found), `409 conflict` (duplicate).

### `GET /cursos`

**Auth**: docente  
**Returns**: Only the authenticated docente's cursos.  
**Success** `200 OK`: array of curso objects.

### `GET /cursos/:id`

**Auth**: docente (must own the curso)  
**Success** `200 OK`: single curso object.  
**Error**: `404 not_found` (not found or belongs to another docente).

### `PUT /cursos/:id`

**Auth**: docente (must own)  
**Body**: any subset of `{ semester_label, section }`  
**Success** `200 OK`: updated curso.

### `DELETE /cursos/:id`

**Auth**: docente (must own)  
**Behavior**: Soft delete if no submissions exist; `409 conflict` with `error: "has_submissions"` if any actividad in the curso has `has_submissions = true`.  
**Success**: `204 No Content`.

### `GET /cursos/:id/students`

**Auth**: docente (must own)  
**Returns**: All students with active inscripciones for this curso, including their grupo assignment if any.  
**Success** `200 OK`:
```json
[
  {
    "user_id": "uuid",
    "email": "student@unifranz.edu.bo",
    "enrolled_at": "...",
    "grupo": { "id": "uuid", "name": "Grupo A" }  // null if unassigned
  }
]
```

---

## Docente — Grupos

### `POST /cursos/:id/grupos`

**Auth**: docente (must own curso)  
**Body**: `{ "name": "Grupo A", "capacity": 5 }` (`capacity` optional)  
**Success** `201 Created`: grupo object.  
**Error**: `409 conflict` (duplicate name in curso).

### `GET /cursos/:id/grupos`

**Auth**: docente (must own)  
**Success** `200 OK`: array of grupo objects with student count.

### `PUT /cursos/:id/grupos/:gid`

**Auth**: docente (must own)  
**Body**: any subset of `{ name, capacity }`  
**Success** `200 OK`.

### `DELETE /cursos/:id/grupos/:gid`

**Auth**: docente (must own)  
**Behavior**: Hard delete. Also removes all `grupo_asignaciones` for this grupo (cascade).  
**Success**: `204 No Content`.

### `POST /cursos/:id/grupos/:gid/students`

Assign a student from the curso roster to a grupo.

**Auth**: docente (must own)  
**Body**: `{ "user_id": "uuid" }`  
**Success** `201 Created`: asignacion object.  
**Errors**:
- `404 not_found` — student not enrolled in this curso
- `409 conflict` — student already assigned to a grupo in this curso

### `DELETE /cursos/:id/grupos/:gid/students/:uid`

**Auth**: docente (must own)  
**Success**: `204 No Content`.

---

## Docente — Actividades

### `POST /cursos/:id/actividades`

**Auth**: docente (must own)  
**Body**:
```json
{
  "title": "Examen Parcial 1",
  "tipo": "examen",
  "deadline": "2026-06-15T18:00:00Z",
  "instructions": "Resolver todos los ejercicios..."
}
```
**Success** `201 Created`:
```json
{
  "id": "uuid",
  "curso_id": "uuid",
  "title": "Examen Parcial 1",
  "tipo": "examen",
  "deadline": "2026-06-15T18:00:00Z",
  "instructions": "...",
  "trace_enabled": true,
  "has_submissions": false,
  "created_at": "...",
  "updated_at": "..."
}
```
**Error**: `400 validation_error` (invalid `tipo` value).

### `GET /cursos/:id/actividades`

**Auth**: docente (must own)  
**Success** `200 OK`: array of actividad objects.

### `GET /cursos/:id/actividades/:aid`

**Auth**: docente (must own)  
**Success** `200 OK`: single actividad with `created_at` and `updated_at` (visible to docente as "posted on" and "last edited").

### `PUT /cursos/:id/actividades/:aid`

**Auth**: docente (must own)  
**Body**: any subset of `{ title, tipo, deadline, instructions }`  
**Behavior**:
- If `has_submissions = false`: all fields editable.
- If `has_submissions = true`: only `deadline` and `instructions` are editable; `title` and `tipo` are locked.

**Errors**:
- `422 validation_error` with `error: "field_locked"` if attempting to change `title` or `tipo` after submissions exist.

**Success** `200 OK`: updated actividad (includes new `updated_at`).

### `DELETE /cursos/:id/actividades/:aid`

**Auth**: docente (must own)  
**Error**: `409 conflict` with `error: "has_submissions"` if `has_submissions = true`.  
**Success**: `204 No Content`.

---

## Docente — Co-Teaching Grants

### `POST /actividades/:aid/grants`

**Auth**: docente (must own the actividad's curso)  
**Body**: `{ "codocente_id": "uuid" }`  
**Validation**: The target user must have `role = "co_docente"`.  
**Success** `201 Created`:
```json
{
  "id": "uuid",
  "actividad_id": "uuid",
  "codocente_id": "uuid",
  "granted_by": "uuid",
  "granted_at": "..."
}
```
**Errors**: `400 validation_error` (target user is not a co_docente), `404 not_found` (actividad not owned by docente), `409 conflict` (grant already active).

### `DELETE /actividades/:aid/grants/:uid`

**Auth**: docente (must own)  
**Behavior**: Sets `revoked_at`, `revoked_by`, `is_active = false`. Never hard-deletes.  
**Success**: `204 No Content`.

---

## Co-Docente — Activity View

### `GET /grants/actividades/:aid`

**Auth**: co_docente (must have an active grant for this actividad)  
**Returns**: The actividad details the co_docente is granted access to.  
**Success** `200 OK`: actividad object (same shape as docente view, excluding `has_submissions`).  
**Error**: `403 forbidden` — no active grant for this actividad.

---

## Student — Course and Activity View

### `GET /me/cursos`

**Auth**: estudiante  
**Returns**: All cursos the student is actively enrolled in (via `inscripciones`).  
**Success** `200 OK`:
```json
[
  {
    "id": "uuid",
    "materia_name": "Programación III",
    "docente_email": "docente@unifranz.edu.bo",
    "semester_label": "2026-1",
    "section": "A",
    "grupo": { "id": "uuid", "name": "Grupo A" }  // null if unassigned
  }
]
```

### `GET /me/cursos/:id/actividades`

**Auth**: estudiante (must be enrolled in the curso)  
**Returns**: All actividades for the curso. Analysis reports and alert status are **not** included.  
**Success** `200 OK`:
```json
[
  {
    "id": "uuid",
    "title": "Examen Parcial 1",
    "tipo": "examen",
    "deadline": "2026-06-15T18:00:00Z",
    "instructions": "...",
    "trace_enabled": true,
    "posted_at": "2026-05-19T...",
    "last_edited_at": "2026-05-20T..."
  }
]
```
**Error**: `403 forbidden` — student not enrolled in this curso.
