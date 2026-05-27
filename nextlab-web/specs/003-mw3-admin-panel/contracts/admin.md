# API Contracts: MW3 — Admin Panel

**Feature**: MW3 — Admin Panel  
**Date**: 2026-05-26  
**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)  
**Auth**: Bearer token via `Authorization` header (injected by axios interceptor in `src/lib/api.ts`)  
**Role guard**: All `/admin/*` endpoints require `role = "admin"`; server returns `403` otherwise.

---

## Carreras

### GET /admin/carreras
Returns all active carreras.

**Response** `200 OK`:
```json
[
  { "id": "uuid", "code": "IS", "name": "Ingeniería de Sistemas", "is_active": true, "created_at": "...", "updated_at": "..." }
]
```
Empty list `[]` if no carreras exist.

---

### POST /admin/carreras
Creates a new carrera.

**Request body**:
```json
{ "code": "IS", "name": "Ingeniería de Sistemas" }
```

**Response** `201 Created`: Carrera object.

**Error responses**:
- `400 Bad Request` `{ "error": "validation_error", "message": "code and name are required" }` — missing fields
- `409 Conflict` `{ "error": "conflict", "message": "a carrera with that code already exists" }` — duplicate code

---

### PUT /admin/carreras/:id
Updates carrera name or active status.

**Request body** (all fields optional):
```json
{ "name": "New Name", "is_active": true }
```

**Response** `200 OK`: Updated Carrera object.

**Error responses**:
- `400 Bad Request` — invalid body
- `404 Not Found` — carrera not found

---

### DELETE /admin/carreras/:id
Soft-deactivates a carrera (sets `is_active = false`). Blocked if active materias exist.

**Response** `204 No Content` on success.

**Error responses**:
- `409 Conflict` `{ "error": "conflict", "message": "carrera has active materias; deactivate them first" }` — has active materias
- `400 Bad Request` — invalid UUID

---

## Materias

### GET /admin/materias
Returns all active materias. Supports optional filters: `?carrera_id=<uuid>` and `?semester=<int>`. For MW3, call without filters to list all.

**Response** `200 OK`:
```json
[
  { "id": "uuid", "carrera_id": "uuid", "code": "MAT101", "name": "Matemáticas I", "semester": 1, "is_active": true, "created_at": "...", "updated_at": "..." }
]
```

---

### POST /admin/materias
Creates a new materia.

**Request body**:
```json
{ "carrera_id": "uuid", "code": "MAT101", "name": "Matemáticas I", "semester": 1 }
```

**Response** `201 Created`: Materia object.

**Error responses**:
- `400 Bad Request` `{ "error": "validation_error", "message": "carrera_id, code, name, and semester (1-10) are required" }` — missing/invalid fields
- `409 Conflict` `{ "error": "conflict", "message": "a materia with that code already exists" }` — duplicate code
- `404 Not Found` `{ "error": "not_found", "message": "carrera not found" }` — invalid `carrera_id`

---

### PUT /admin/materias/:id
Updates materia name, semester, or active status.

**Request body** (all optional):
```json
{ "name": "New Name", "semester": 2, "is_active": true }
```

**Response** `200 OK`: Updated Materia object.

**Error responses**:
- `404 Not Found` — materia not found

---

### DELETE /admin/materias/:id
Soft-deactivates a materia. Blocked if active cursos reference it.

**Response** `204 No Content` on success.

**Error responses**:
- `409 Conflict` `{ "error": "conflict", "message": "materia has active cursos; deactivate them first" }` — has active cursos

---

## Student Profiles

### GET /admin/student-profiles
Returns all active student profiles. Supports optional filters: `?carrera_id=<uuid>` and `?semester=<int>`. For MW3, call without filters.

**Response** `200 OK`:
```json
[
  { "id": "uuid", "email": "elder@unifranz.edu.bo", "carrera_id": "uuid", "semester": 5, "student_code": "2020-IS-001", "is_active": true, "created_at": "...", "updated_at": "..." }
]
```
`student_code` may be absent (omitempty) — treat as `null`.

---

### POST /admin/student-profiles
Creates a student profile.

**Request body**:
```json
{ "email": "elder@unifranz.edu.bo", "carrera_id": "uuid", "semester": 5, "student_code": "2020-IS-001" }
```
`student_code` is optional.

**Response** `201 Created`: StudentProfile object.

**Error responses**:
- `400 Bad Request` `{ "error": "validation_error", "message": "email, carrera_id, and semester (1-10) are required" }` — missing fields
- `400 Bad Request` `{ "error": "validation_error", "message": "email must be a @unifranz.edu.bo address" }` — wrong domain
- `409 Conflict` `{ "error": "conflict", "message": "a student profile with that email already exists" }` — duplicate email

---

### PUT /admin/student-profiles/:id
Updates semester, student_code, or active status. Email and carrera_id are immutable.

**Request body** (all optional):
```json
{ "semester": 6, "student_code": "2020-IS-001", "is_active": false }
```

**Response** `200 OK`: Updated StudentProfile object.

**Error responses**:
- `404 Not Found` — profile not found

> **Note**: There is no `DELETE /admin/student-profiles/:id`. Deactivation is done via `PUT` with `{ "is_active": false }`. The frontend `deactivateStudentProfile(id)` function wraps this.

---

## `src/api/admin.ts` Function Signatures

```typescript
// Carreras
getCarreras(): Promise<Carrera[]>
createCarrera(data: CreateCarreraRequest): Promise<Carrera>
updateCarrera(id: string, data: UpdateCarreraRequest): Promise<Carrera>
deleteCarrera(id: string): Promise<void>          // DELETE /admin/carreras/:id → 204

// Materias (admin full CRUD — distinct from getMaterias() in academic.ts)
getAdminMaterias(): Promise<Materia[]>
createMateria(data: CreateMateriaRequest): Promise<Materia>
updateMateria(id: string, data: UpdateMateriaRequest): Promise<Materia>
deleteMateria(id: string): Promise<void>          // DELETE /admin/materias/:id → 204

// Student Profiles
getStudentProfiles(): Promise<StudentProfile[]>
createStudentProfile(data: CreateStudentProfileRequest): Promise<StudentProfile>
updateStudentProfile(id: string, data: UpdateStudentProfileRequest): Promise<StudentProfile>
deactivateStudentProfile(id: string): Promise<StudentProfile>   // PUT with { is_active: false }
```
