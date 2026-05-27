# Implementation Plan: MW4 — Student Portal

**Branch**: `013-appeals-final-audit` | **Date**: 2026-05-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/004-mw4-student-portal/spec.md`
**Research**: `specs/004-mw4-student-portal/research.md`
**Data Model**: `specs/004-mw4-student-portal/data-model.md`
**Contracts**: `specs/004-mw4-student-portal/contracts/student.md`

---

## Summary

Build the student-facing portal: a course list dashboard, a course activity list page, and an activity detail page. All data is fetched live from `nextlab-api`; no mocks. The existing Bloque 4 scaffold pages (10 unused mocked files) are deleted. Key decision: the activity detail page has no dedicated API endpoint — it re-fetches `GET /me/cursos/:id/actividades` and finds the activity by `:aid`, ensuring direct URL navigation works.

Key decisions from research (see research.md for full rationale):
- `StudentCursoEntry` and `StudentActividadEntry` types added to `src/types/academic.ts`
- New `src/api/student.ts` for student-facing API calls (separate from docente academic.ts and admin admin.ts)
- Student dashboard (`/student/dashboard`) IS the course list — no separate `/student/cursos` route
- Activity detail page re-fetches `GET /me/cursos/:id/actividades` and finds by `:aid`
- 10 old Bloque 4 mock files deleted before new pages are added

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: N/A — frontend only; all data fetched from `nextlab-api` via HTTP

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW4.

**Target Platform**: Browser (React SPA at `http://localhost:5173`)

**Project Type**: Web application — frontend display layer

**Performance Goals**: Course list loads in under 3 seconds (SC-001)

**Constraints**: No external UI component library. No business logic on the frontend (constitution Principle VII). No deadline enforcement — display whatever the API returns.

**Scale/Scope**: ~3 pages (1 updated, 2 new), 1 new API module, ~3 new types, 10 files deleted. Small milestone.

---

## Constitution Check

*GATE: Must pass before implementation. Re-checked after design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | All 3 student routes inside the existing `ProtectedRoute allowedRoles={['estudiante']}` block. Non-student users redirected to their own dashboard. Role derived from JWT via `AuthContext`. |
| II. API Client Isolation | ✅ Pass | `src/api/student.ts` imports `api` from `../lib/api`. No raw `fetch()`. Two functions: `getStudentCursos()` and `getStudentActividades(cursoId)`. |
| III. React Quality Gate | ✅ Pass | React Doctor scan required after all MW4 pages complete, before milestone marked done. |
| IV. Token Security | ✅ Pass | No token handling changes. Access token stays in `AuthContext` memory. Axios interceptor handles auth automatically. |
| V. Read-Only Evidence | ✅ Pass | No evidence data in MW4 (no trace events, grades, or appeals). Not applicable. |
| VI. Brand Baseline | ✅ Pass | Unifranz palette throughout. No shadcn, MUI, or component library. Cards and tables built with Tailwind CSS v4. |
| VII. Thin Client | ✅ Pass | No business logic. The only client-side computation is: (a) checking `deadline < now` for visual styling, (b) filtering the activity list by `:aid` for detail page. Both are purely cosmetic/navigation — no enforcement logic. |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/004-mw4-student-portal/
├── plan.md           ← this file
├── research.md       ← Phase 0 output (6 decisions)
├── data-model.md     ← Phase 1 output
├── quickstart.md     ← Phase 1 output
├── contracts/
│   └── student.md    ← Phase 1 output
└── tasks.md          ← Phase 2 output (speckit-tasks — NOT created by speckit-plan)
```

### Source Code Changes

Files to **delete** (old Bloque 4 scaffold — not imported by router):

```text
src/pages/student/
├── Dashboard.tsx        # old mocked dashboard (different from DashboardPage.tsx)
├── Exam.tsx
├── Practice.tsx
├── Task.tsx
├── Report.tsx
├── Project.tsx
├── Defense.tsx
├── Grades.tsx
├── Appeal.tsx
└── ProjectStage.tsx
```

Files to **create** (new):

```text
src/
├── api/
│   └── student.ts              # getStudentCursos(), getStudentActividades(cursoId)
│
└── pages/
    └── student/
        ├── ActividadesPage.tsx     # Course activity list
        └── ActividadDetailPage.tsx # Activity detail (re-fetches list, finds by :aid)
```

Files to **update** (existing):

```text
src/
├── types/
│   └── academic.ts    # Add GrupoRef, StudentCursoEntry, StudentActividadEntry
│
├── pages/
│   └── student/
│       └── DashboardPage.tsx  # Replace stub with course list (US1)
│
└── router/
    └── index.tsx      # Add 2 new student routes
```

---

## Key Implementation Details

### `src/api/student.ts` — error handling pattern

Follows identical `toApiError` pattern as `admin.ts`:

```ts
export async function getStudentCursos(): Promise<StudentCursoEntry[]> {
  try {
    const res = await api.get<StudentCursoEntry[]>('/me/cursos')
    return res.data
  } catch (err) {
    toApiError(err)
  }
}

export async function getStudentActividades(cursoId: string): Promise<StudentActividadEntry[]> {
  try {
    const res = await api.get<StudentActividadEntry[]>(`/me/cursos/${cursoId}/actividades`)
    return res.data
  } catch (err) {
    toApiError(err)
  }
}
```

### `DashboardPage` — course cards

State: `items: StudentCursoEntry[]`, `loading: boolean`, `error: string | null`. On mount: call `getStudentCursos()`. Each card: `<Link to={/student/cursos/${curso.id}/actividades}>` wrapping card content. Grupo display: `curso.grupo?.name ?? 'Sin grupo'`.

### `ActividadesPage` — activity list

State: `items: StudentActividadEntry[]`, `loading`, `error`, `cursoId` from `useParams()`. On mount: call `getStudentActividades(cursoId)`. For each activity row: tipo badge (uses existing `ActividadTipo` values), formatted deadline (check `new Date(deadline) < new Date()` for "Vencido" styling), `trace_enabled` badge. Click row → navigate to `/student/cursos/${cursoId}/actividades/${actividad.id}`.

Note on 403 display: `toApiError` will throw `ApiError`; the catch block sets `error` state with the message "not enrolled in this curso". A friendly message wraps this: "No tienes acceso a este curso" + link back to dashboard.

### `ActividadDetailPage` — re-fetch pattern

State: `actividad: StudentActividadEntry | null`, `loading`, `error`, `cursoId` + `aid` from `useParams()`. On mount: call `getStudentActividades(cursoId)`, then `find(a => a.id === aid)`. If not found: set `error` = "not-found". If found: set `actividad`. Full instructions rendered with `whitespace-pre-wrap` or similar.

### `src/router/index.tsx` — student routes

Inside the existing `ProtectedRoute allowedRoles={['estudiante']}` children:

```ts
{ path: '/student/dashboard', element: <StudentDashboard /> },
{ path: '/student/cursos/:id/actividades', element: <ActividadesPage /> },
{ path: '/student/cursos/:id/actividades/:aid', element: <ActividadDetailPage /> },
```

### Deadline display helper

```ts
function formatDeadline(iso: string): { label: string; isPast: boolean } {
  const d = new Date(iso)
  const isPast = d < new Date()
  const label = d.toLocaleString('es-BO', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
  return { label, isPast }
}
```

---

## Scope Adjustments (vs. original constitution plan)

| Spec claim | Actual behavior | Source |
|-----------|-----------------|--------|
| Activity detail page | No `GET /me/cursos/:id/actividades/:aid` endpoint exists; detail re-fetches list and finds by ID | Research Decision 5 |
| 10 old student pages | Not imported by router; deleted as dead Bloque 4 scaffold | Research Decision 6 |

---

## Complexity Tracking

No constitution violations. No complexity justification required.
