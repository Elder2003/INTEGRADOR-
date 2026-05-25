# UI Contracts: MW2 — Docente Course Management

Interface contracts for the API layer and UI components built in MW2.

---

## src/api/academic.ts

All functions throw `ApiError` on failure. Callers are responsible for catching.

```ts
// Cursos
getCursos(): Promise<Curso[]>
getCurso(id: string): Promise<Curso>
createCurso(data: CreateCursoRequest): Promise<Curso>
updateCurso(id: string, data: UpdateCursoRequest): Promise<Curso>
deleteCurso(id: string): Promise<void>

// Materias (admin-only — returns [] on 403 instead of throwing)
getMaterias(): Promise<Materia[]>

// Actividades
getActividades(cursoId: string): Promise<Actividad[]>
getActividad(id: string): Promise<Actividad>
createActividad(cursoId: string, data: CreateActividadRequest): Promise<Actividad>
updateActividad(id: string, data: UpdateActividadRequest): Promise<Actividad>
deleteActividad(id: string): Promise<void>

// Grupos
getGrupos(cursoId: string): Promise<Grupo[]>
createGrupo(cursoId: string, data: CreateGrupoRequest): Promise<Grupo>
updateGrupo(cursoId: string, grupoId: string, data: UpdateGrupoRequest): Promise<Grupo>
deleteGrupo(cursoId: string, grupoId: string): Promise<void>

// Student roster (read-only)
getRoster(cursoId: string): Promise<RosterEntry[]>

// Group membership
addStudentToGrupo(cursoId: string, grupoId: string, userId: string): Promise<void>
removeStudentFromGrupo(cursoId: string, grupoId: string, userId: string): Promise<void>

// Co-docente grants
grantAccess(actividadId: string, userId: string): Promise<CodocenteGrant>
revokeAccess(actividadId: string, userId: string): Promise<void>
```

**Special case — `getMaterias()`:**
- Calls `GET /admin/materias`
- On 200: returns the array
- On 403: returns `[]` (docente receives 403 — caller shows manual UUID input)
- On any other error: throws `ApiError` normally

---

## ConfirmModal

**File:** `src/components/common/ConfirmModal.tsx`

```ts
interface ConfirmModalProps {
  isOpen: boolean
  title: string
  message: string
  confirmLabel?: string   // default: "Delete"
  isLoading: boolean
  onConfirm: () => void
  onCancel: () => void
}
```

**Behaviour:**
- Renders nothing when `isOpen = false`
- Disables both buttons and shows a spinner on the confirm button when `isLoading = true`
- Does NOT manage open/closed state internally — parent controls `isOpen`
- Does NOT display error messages — parent renders errors via `<ErrorMessage>` after modal closes
- Pressing Escape or clicking the backdrop calls `onCancel`

---

## DashboardPage (docente)

**File:** `src/pages/docente/DashboardPage.tsx`  
**Route:** `/docente/dashboard`

**Behaviour:**
- On mount: calls `getCursos()`
- Displays `<LoadingSpinner />` while loading
- On success: renders a list of `<CourseCard>` components
- Empty state: "You have no courses yet" + "New Course" button → `/docente/cursos/new`
- On error: renders `<ErrorMessage>`

---

## CursoDetailPage

**File:** `src/pages/docente/CursoDetailPage.tsx`  
**Route:** `/docente/cursos/:id?tab=actividades|estudiantes|grupos`

```ts
// Uses useSearchParams() to read/write ?tab= param
// Default tab if param absent or invalid: "actividades"
```

**Behaviour:**
- On mount: calls `getCurso(id)`
- Shows course name (materia + semester + section) in the page header
- Three tab buttons: Actividades, Estudiantes, Grupos
- Clicking a tab updates the URL query param without a full navigation
- Tab content is rendered below the tab bar — only active tab's data is fetched
- Edit course button → `/docente/cursos/:id/edit`
- Delete course button → `<ConfirmModal>` → `deleteCurso(id)` → navigate back to `/docente/dashboard`

---

## CreateCursoPage

**File:** `src/pages/docente/CreateCursoPage.tsx`  
**Route:** `/docente/cursos/new`

**Behaviour:**
- On mount: calls `getMaterias()`
  - If returns non-empty array: render `<select>` with options
  - If returns empty array (403 or no materias): render text input + message "Contact your admin to get the materia ID"
- Form fields: materia (select or text), semester (text), section (text)
- Validation: all fields non-empty before enabling submit button
- On submit: calls `createCurso(data)` → navigates to `/docente/cursos/:id?tab=actividades`
- Cancel button → navigate back to `/docente/dashboard`

---

## EditCursoPage

**File:** `src/pages/docente/EditCursoPage.tsx`  
**Route:** `/docente/cursos/:id/edit`

**Behaviour:**
- On mount: calls `getCurso(id)` to pre-fill semester and section
- `materia_id` is not editable (shown as read-only text)
- On submit: calls `updateCurso(id, { semester, section })` → navigates back to `/docente/cursos/:id`
- Cancel → navigate back to `/docente/cursos/:id`

---

## CreateActividadPage

**File:** `src/pages/docente/CreateActividadPage.tsx`  
**Route:** `/docente/cursos/:id/actividades/new`

**Behaviour:**
- Form fields: title (text), tipo (select — 5 values), deadline (datetime-local), instructions (textarea)
- `tipo` uses a fixed select with these options: practica, tarea, informe, examen, defensa_oral
- `deadline`: `<input type="datetime-local">` — value is converted to ISO 8601 before POST:
  `deadline.replace('T', 'T') + ':00Z'` (append seconds + UTC suffix)
- Validation: all fields non-empty before enabling submit
- On submit: calls `createActividad(cursoId, data)` → navigates to `/docente/cursos/:id?tab=actividades`
- Back button → navigate to `/docente/cursos/:id?tab=actividades`

---

## ActividadDetailPage

**File:** `src/pages/docente/ActividadDetailPage.tsx`  
**Route:** `/docente/cursos/:id/actividades/:aid`

**Behaviour:**
- On mount: calls `getActividad(aid)`
- Shows: title, tipo badge, deadline, instructions, late_submission_allowed toggle
- Toggle calls `updateActividad(aid, { late_submission_allowed: !current })`
- Edit button → `/docente/cursos/:id/actividades/:aid/edit`
- Delete button → `<ConfirmModal>` → `deleteActividad(aid)` → navigate to `/docente/cursos/:id?tab=actividades`
- Co-docente grants section (tracked in component state, not fetched):
  - Input for user_id + "Grant Access" button
  - On grant: calls `grantAccess(aid, userId)` → appends returned `CodocenteGrant` to local state
  - On revoke: calls `revokeAccess(aid, userId)` → removes entry from local state
  - Shows note: "Co-docente list refreshes on page reload"

---

## EditActividadPage

**File:** `src/pages/docente/EditActividadPage.tsx`  
**Route:** `/docente/cursos/:id/actividades/:aid/edit`

**Behaviour:**
- On mount: calls `getActividad(aid)` to pre-fill form
- `title` input: disabled when `actividad.has_submissions = true`
- `tipo` select: disabled when `actividad.has_submissions = true`
- When disabled, shows note: "Locked: submissions exist"
- `deadline`, `instructions`, `late_submission_allowed`: always editable
- On submit: calls `updateActividad(aid, data)` → navigates back to `/docente/cursos/:id/actividades/:aid`
- Cancel → navigate back to `/docente/cursos/:id/actividades/:aid`

---

## Router additions

All routes below are nested inside the existing `docente` `ProtectedRoute` (role: `docente`):

| Path | Component |
|------|-----------|
| `/docente/dashboard` | `DashboardPage` (updated — real course list) |
| `/docente/cursos/new` | `CreateCursoPage` |
| `/docente/cursos/:id` | `CursoDetailPage` |
| `/docente/cursos/:id/edit` | `EditCursoPage` |
| `/docente/cursos/:id/actividades/new` | `CreateActividadPage` |
| `/docente/cursos/:id/actividades/:aid` | `ActividadDetailPage` |
| `/docente/cursos/:id/actividades/:aid/edit` | `EditActividadPage` |

**Note:** `/docente/cursos/new` MUST be listed before `/docente/cursos/:id` in the router
to prevent `"new"` from being matched as the `:id` param.

---

## CourseCard

**File:** `src/components/docente/CourseCard.tsx`

```ts
interface CourseCardProps {
  curso: Curso
  onClick: () => void
}
```

Displays: materia name, semester + section, enrolled student count.  
Full card is a clickable link → `/docente/cursos/:id`.

---

## ActividadTypeSelect

**File:** `src/components/docente/ActividadTypeSelect.tsx`

```ts
interface ActividadTypeSelectProps {
  value: ActividadTipo | ''
  onChange: (v: ActividadTipo) => void
  disabled?: boolean
}
```

Renders a `<select>` with exactly 5 options: practica, tarea, informe, examen, defensa_oral.
Displays human-readable labels (capitalised) while the value matches the API enum string.
