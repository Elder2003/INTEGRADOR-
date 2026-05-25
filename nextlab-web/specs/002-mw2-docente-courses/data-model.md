# Data Model: MW2 — Docente Course Management

Frontend-only types representing the shape of API responses and request payloads.
All new types go into `src/types/index.ts` under a `// MW2 types` block.

---

## ActividadTipo (corrected from Bloque 4 prototype)

```ts
type ActividadTipo = 'practica' | 'tarea' | 'informe' | 'examen' | 'defensa_oral'
```

Replaces the prototype type. The API ENUM has exactly 5 values — `investigacion` and
`git_analysis` are not in `actividades` (see research Decision 1).

---

## Materia

Returned by `GET /admin/materias`. Admin-only endpoint — docentes receive 403.

```ts
interface Materia {
  id: string        // UUID
  nombre: string
  codigo: string
  is_active: boolean
}
```

---

## Curso

Returned by `GET /cursos` (list) and `GET /cursos/:id` (single).

```ts
interface Curso {
  id: string             // UUID
  materia_id: string     // UUID
  materia_nombre: string // populated by API join
  semester: string       // e.g. "2026-1"
  section: string        // e.g. "A"
  docente_id: string     // UUID
  student_count: number
  created_at: string     // ISO 8601
}
```

---

## Actividad

Returned by `GET /cursos/:id/actividades` (list) and `GET /actividades/:id` (single).

```ts
interface Actividad {
  id: string
  curso_id: string
  title: string
  tipo: ActividadTipo
  deadline: string              // ISO 8601 with timezone
  instructions: string
  late_submission_allowed: boolean
  has_submissions: boolean
  created_at: string
}
```

**Field locking rule** (research Decision 3):
- `has_submissions = false` → `title` and `tipo` are editable
- `has_submissions = true` → only `deadline`, `instructions`, `late_submission_allowed` may change; the API returns 422 `field_locked` if `title` or `tipo` are sent

---

## Grupo

Returned by `GET /cursos/:id/grupos`.

```ts
interface Grupo {
  id: string
  curso_id: string
  nombre: string
  capacity: number | null
  member_count: number
  created_at: string
}
```

---

## RosterEntry

Returned by `GET /cursos/:id/students`.

```ts
interface RosterEntry {
  user_id: string
  email: string
  enrolled_at: string       // ISO 8601
  grupo_id: string | null
  grupo_nombre: string | null
}
```

---

## CodocenteGrant

Returned by `POST /actividades/:aid/grants`. No GET endpoint exists — grants are
tracked in React component state only and lost on page refresh (research Decision 5).

```ts
interface CodocenteGrant {
  id: string
  actividad_id: string
  user_id: string
  granted_at: string        // ISO 8601
}
```

---

## Request Shapes

### CreateCursoRequest

```ts
interface CreateCursoRequest {
  materia_id: string
  semester: string
  section: string
}
```

### UpdateCursoRequest

```ts
interface UpdateCursoRequest {
  semester: string
  section: string
}
```

### CreateActividadRequest

```ts
interface CreateActividadRequest {
  title: string
  tipo: ActividadTipo
  deadline: string    // ISO 8601 — frontend converts datetime-local to "YYYY-MM-DDTHH:mm:00Z"
  instructions: string
}
```

### UpdateActividadRequest

All fields optional. API ignores `title`/`tipo` when `has_submissions = true`.

```ts
interface UpdateActividadRequest {
  title?: string
  tipo?: ActividadTipo
  deadline?: string
  instructions?: string
  late_submission_allowed?: boolean
}
```

### CreateGrupoRequest / UpdateGrupoRequest

```ts
interface CreateGrupoRequest {
  nombre: string
  capacity?: number
}

interface UpdateGrupoRequest {
  nombre?: string
  capacity?: number
}
```

### GrantAccessRequest

```ts
interface GrantAccessRequest {
  user_id: string
}
```
