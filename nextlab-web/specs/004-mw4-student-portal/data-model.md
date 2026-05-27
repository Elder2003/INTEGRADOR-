# Data Model: MW4 — Student Portal

**Feature**: MW4 — Student Portal
**Date**: 2026-05-26

All types are added to `src/types/academic.ts`.

---

## New Types

### GrupoRef (nested in StudentCursoEntry)

```typescript
export interface GrupoRef {
  id: string
  name: string
}
```

### StudentCursoEntry

The enrolled course view returned by `GET /me/cursos`. One entry per inscription.

```typescript
export interface StudentCursoEntry {
  id: string              // curso UUID
  materia_name: string    // e.g. "Matemáticas I"
  docente_email: string   // e.g. "docente@unifranz.edu.bo"
  semester_label: string  // e.g. "2026-1"
  section: string         // e.g. "A"
  grupo: GrupoRef | null  // null if student not assigned to a group
}
```

### StudentActividadEntry

The activity view returned by `GET /me/cursos/:id/actividades`. Used for both the list and detail pages.

```typescript
export interface StudentActividadEntry {
  id: string            // actividad UUID
  title: string
  tipo: ActividadTipo   // reuse existing: 'practica' | 'tarea' | 'informe' | 'examen' | 'defensa_oral'
  deadline: string      // ISO 8601 timestamp — formatted for display only
  instructions: string  // full text, shown on detail page
  trace_enabled: boolean
  posted_at: string     // ISO 8601
  last_edited_at: string // ISO 8601
}
```

> **Note**: `ActividadTipo` already exists in `src/types/academic.ts` — reuse it for `tipo`.
> `GrupoRef` needs to be added as a new interface.

---

## Existing Types (unchanged)

`ActividadTipo`, `Materia`, `Carrera`, `StudentProfile`, `Curso`, `Actividad`, `Grupo`, `RosterEntry` — all remain as-is.

---

## No request types needed

MW4 is read-only. No POST/PUT/DELETE request bodies are sent by the student portal.
