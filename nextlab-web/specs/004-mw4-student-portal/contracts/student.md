# API Contracts: MW4 — Student Portal

**Feature**: MW4 — Student Portal
**Date**: 2026-05-26
**Base URL**: `VITE_API_URL` (default: `http://localhost:8080`)
**Auth**: Bearer token via `Authorization` header (injected by axios interceptor in `src/lib/api.ts`)
**Role guard**: All `/me/*` endpoints require `role = "estudiante"`; server returns `403` otherwise.

---

## Student Courses

### GET /me/cursos

Returns all active courses the authenticated student is enrolled in.

**Response** `200 OK`:
```json
[
  {
    "id": "uuid",
    "materia_name": "Matemáticas I",
    "docente_email": "docente@unifranz.edu.bo",
    "semester_label": "2026-1",
    "section": "A",
    "grupo": { "id": "uuid", "name": "Grupo 1" }
  }
]
```

`grupo` may be `null` if the student has not been assigned to a group.
Empty list `[]` if student has no active enrollments.

**Error responses**: none (returns [] on no enrollments)

---

## Student Activities

### GET /me/cursos/:id/actividades

Returns all activities for the specified course. The student must be enrolled in the course.

**Path parameter**: `:id` — curso UUID

**Response** `200 OK`:
```json
[
  {
    "id": "uuid",
    "title": "Práctica 1",
    "tipo": "practica",
    "deadline": "2026-06-01T23:59:00Z",
    "instructions": "Desarrollar el ejercicio...",
    "trace_enabled": true,
    "posted_at": "2026-05-20T10:00:00Z",
    "last_edited_at": "2026-05-20T10:00:00Z"
  }
]
```

Empty list `[]` if course has no activities.

**Error responses**:
- `400 Bad Request` `{ "error": "invalid_id", "message": "invalid curso id" }` — malformed UUID
- `403 Forbidden` `{ "error": "forbidden", "message": "not enrolled in this curso" }` — student not enrolled

---

## `src/api/student.ts` Function Signatures

```typescript
// Student courses
getStudentCursos(): Promise<StudentCursoEntry[]>

// Student activities (also used for detail page re-fetch)
getStudentActividades(cursoId: string): Promise<StudentActividadEntry[]>
```

Both functions follow the same `toApiError` helper pattern as `src/api/admin.ts` and `src/api/academic.ts`.
