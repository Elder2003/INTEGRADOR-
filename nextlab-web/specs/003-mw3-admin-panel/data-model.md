# Data Model: MW3 — Admin Panel

**Feature**: MW3 — Admin Panel  
**Date**: 2026-05-26

All types are added to `src/types/academic.ts` alongside the existing `Materia`, `Curso`, etc.

---

## Response Types (from API)

```typescript
// Already exists in src/types/academic.ts — no changes needed
interface Materia {
  id: string
  carrera_id: string
  code: string
  name: string
  semester: number
  is_active: boolean
  created_at: string
  updated_at: string
}

// NEW — add to src/types/academic.ts
interface Carrera {
  id: string
  code: string
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

// NEW — add to src/types/academic.ts
interface StudentProfile {
  id: string
  email: string
  carrera_id: string
  semester: number
  student_code: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}
```

---

## Request Types

```typescript
// NEW — add to src/types/academic.ts
interface CreateCarreraRequest {
  code: string   // short unique identifier, e.g. "IS"
  name: string   // full name, e.g. "Ingeniería de Sistemas"
}

// NEW — add to src/types/academic.ts
interface UpdateCarreraRequest {
  name?: string
  is_active?: boolean
}

// NEW — add to src/types/academic.ts
interface CreateMateriaRequest {
  carrera_id: string   // UUID
  code: string
  name: string
  semester: number     // 1–10
}

// NEW — add to src/types/academic.ts
interface UpdateMateriaRequest {
  name?: string
  semester?: number
  is_active?: boolean
}

// NEW — add to src/types/academic.ts
interface CreateStudentProfileRequest {
  email: string        // must end in @unifranz.edu.bo
  carrera_id: string   // UUID
  semester: number     // 1–10
  student_code?: string
}

// NEW — add to src/types/academic.ts
interface UpdateStudentProfileRequest {
  semester?: number
  student_code?: string | null
  is_active?: boolean
}
```

---

## Notes

- `Materia` already exists — confirm `code` and `semester` fields are present; add them if the existing type is missing them.
- `student_code` is nullable (`string | null`) — the API returns `omitempty` so it may be absent; treat absence as `null`.
- All `id` fields are UUIDs represented as strings on the frontend.
- `semester` is always an integer 1–10; the UI uses a `<select>` with 10 hardcoded options.
