# Tasks: MW4 — Student Portal

**Feature**: MW4 — Student Portal
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/student.md](contracts/student.md)
**Total tasks**: 11 | **Last updated**: 2026-05-26

---

## Phase 1 — Setup

**Goal**: Remove dead code and lay the shared foundation (types, stubs) before any user story work begins. All tasks in this phase can run in parallel.

- [X] T001 [P] Delete 10 old Bloque 4 mocked pages from `src/pages/student/` — these are NOT imported by the router and must be removed before new pages are added to avoid confusion: `Dashboard.tsx`, `Exam.tsx`, `Practice.tsx`, `Task.tsx`, `Report.tsx`, `Project.tsx`, `Defense.tsx`, `Grades.tsx`, `Appeal.tsx`, `ProjectStage.tsx`. Verify none are imported anywhere with a quick grep for each filename before deleting.

- [X] T002 [P] Add MW4 types to `src/types/academic.ts` — add 3 new interfaces after the existing type declarations: `GrupoRef { id: string; name: string }`, `StudentCursoEntry { id: string; materia_name: string; docente_email: string; semester_label: string; section: string; grupo: GrupoRef | null }`, `StudentActividadEntry { id: string; title: string; tipo: ActividadTipo; deadline: string; instructions: string; trace_enabled: boolean; posted_at: string; last_edited_at: string }` — reuse the existing `ActividadTipo` union type for the `tipo` field; all shapes confirmed in data-model.md.

- [X] T003 [P] Create minimal placeholder stubs for 2 new student page files: `src/pages/student/ActividadesPage.tsx` (exports named `ActividadesPage` returning `<p>ActividadesPage</p>`) and `src/pages/student/ActividadDetailPage.tsx` (exports named `ActividadDetailPage` returning `<p>ActividadDetailPage</p>`). These will be replaced in Phases 4–5.

---

## Phase 2 — Foundational

**Goal**: Wire up the API module and router so all routes compile. Both tasks can run in parallel after their prerequisites.

- [X] T004 [P] Create `src/api/student.ts` — implement 2 student-facing API functions using the same `toApiError` helper pattern as `src/api/admin.ts`; import `api` from `../lib/api` and types from `../types/academic`: `getStudentCursos(): Promise<StudentCursoEntry[]>` → `GET /me/cursos`; `getStudentActividades(cursoId: string): Promise<StudentActividadEntry[]>` → `GET /me/cursos/${cursoId}/actividades`. Both functions catch all errors via `toApiError(err)`. Full signatures in contracts/student.md. Depends on T002.

- [X] T005 [P] Update `src/router/index.tsx` — import `ActividadesPage` and `ActividadDetailPage` from the stub files created in T003; add 2 new routes inside the existing `ProtectedRoute allowedRoles={['estudiante']}` children array (after the existing `/student/dashboard` route): `{ path: '/student/cursos/:id/actividades', element: <ActividadesPage /> }` and `{ path: '/student/cursos/:id/actividades/:aid', element: <ActividadDetailPage /> }`. Depends on T003.

---

## Phase 3 — US1: Student Course Dashboard (P1) 🎯 MVP

**Story goal**: Student logs in and sees all their enrolled courses on the dashboard. Each course shows subject name, docente email, semester label, section, and group name (or "Sin grupo"). Empty state shown when no enrollments.

**Independent test criteria**: Navigate to `/student/dashboard` as a student with enrollments → course cards render from `GET /me/cursos`; each card shows materia_name, docente_email, semester_label, section, and grupo name ("Sin grupo" if null); clicking a card navigates to `/student/cursos/:id/actividades`; empty state visible when no enrollments; loading spinner shown during fetch; error shown on API failure.

- [X] T006 [US1] Implement `src/pages/student/DashboardPage.tsx` — replace current stub with full course list. State: `items: StudentCursoEntry[]`, `loading: boolean`, `error: string | null`. On mount: call `getStudentCursos()`, show `<LoadingSpinner />` while loading, show `<ErrorMessage>` on fetch failure. Course grid: `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4`; each card is a `<Link to={/student/cursos/${curso.id}/actividades}>` with `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 hover:border-[#E85D24] transition-colors`; card content: `<h2 className="text-[#f0f0f0] font-semibold">{curso.materia_name}</h2>`, `<p className="text-sm text-[#888888]">{curso.docente_email}</p>`, `<p className="text-sm text-[#888888]">{curso.semester_label} · Sección {curso.section}</p>`, `<p className="text-sm text-[#888888]">{curso.grupo?.name ?? 'Sin grupo'}</p>`. Empty state when `items.length === 0`: `<p>Aún no estás inscrito en ningún curso.</p>`. Page heading: "Mis Cursos". Depends on T004, T005.

---

## Phase 4 — US2: Course Activity List (P2)

**Story goal**: Student clicks a course card and sees the list of published activities. Each activity shows title, tipo badge, formatted deadline (visually marking past deadlines as "Vencido"), and a monitoring badge when trace_enabled. Access denied shown if not enrolled. Clicking an activity navigates to the detail page.

**Independent test criteria**: Navigate to `/student/cursos/:id/actividades` as enrolled student → list renders from `GET /me/cursos/:id/actividades`; each row shows title, tipo label, formatted deadline, monitoring badge for trace_enabled; past deadline shows "Vencido" label; empty state when no activities; 403 shows access-denied message with link back to dashboard.

- [X] T007 [US2] Implement `src/pages/student/ActividadesPage.tsx` — replace stub. State: `items: StudentActividadEntry[]`, `loading: boolean`, `error: string | null`. Get `cursoId` from `useParams<{ id: string }>()`. On mount: call `getStudentActividades(cursoId)`, show `<LoadingSpinner />` while loading. Error handling: catch errors and check message — if the API returns "not enrolled" (403), set `error` with the user-friendly message "No tienes acceso a este curso." + render a `<Link to="/student/dashboard">` back link alongside the `<ErrorMessage>`. Activity list (when loaded): use a `<ul>` or table; each row/item is a `<Link to={/student/cursos/${cursoId}/actividades/${act.id}>` block showing: `act.title` (primary text), tipo badge using a `tipoLabel` map (`{ practica: 'Práctica', tarea: 'Tarea', informe: 'Informe', examen: 'Examen', defensa_oral: 'Defensa Oral' }`), deadline formatted as `new Date(act.deadline).toLocaleString('es-BO', { ... })` — if `new Date(act.deadline) < new Date()` add a `text-[#888888]` "Vencido" span, and a "Requiere monitoreo" badge (`bg-[#E85D24]/20 text-[#E85D24] text-xs px-2 py-0.5 rounded`) when `act.trace_enabled`. Empty state when `items.length === 0`: "Este curso aún no tiene actividades publicadas." Include a `<Link to="/student/dashboard">← Mis Cursos</Link>` back link in the page header. Page heading: shows materia name if available (or "Actividades del Curso" as fallback — materia name is NOT in this API response, so use the generic heading). Depends on T004, T005.

---

## Phase 5 — US3: Activity Detail (P3)

**Story goal**: Student navigates to an activity detail page (from the list or directly via URL) and sees the full activity — title, tipo, deadline with past-deadline indicator, full instructions, and monitoring badge. Direct URL navigation works because the page re-fetches the activity list and finds by `:aid`.

**Independent test criteria**: Navigate to `/student/cursos/:id/actividades/:aid` (from link and directly via bookmark) → full instructions, tipo, deadline, monitoring badge rendered; "Volver al curso" navigates back to activity list; non-existent `:aid` shows not-found message; 403 shows access-denied message.

- [X] T008 [US3] Implement `src/pages/student/ActividadDetailPage.tsx` — replace stub. State: `actividad: StudentActividadEntry | null`, `loading: boolean`, `error: string | null`. Get `cursoId` + `aid` from `useParams<{ id: string; aid: string }>()`. On mount: call `getStudentActividades(cursoId)`, then `const found = list.find(a => a.id === aid)` — if not found after successful fetch, set `error` = "Actividad no encontrada."; if 403, set `error` = "No tienes acceso a este curso."; otherwise set `actividad = found`. Show `<LoadingSpinner />` while loading. Error state: show `<ErrorMessage message={error}>` with a `<Link to={/student/cursos/${cursoId}/actividades}>← Volver al curso</Link>`. Detail view (when actividad loaded): page heading `actividad.title`; tipo badge (same `tipoLabel` map as T007); deadline block — formatted deadline string + "Vencido" label if past; "Requiere monitoreo" badge if `actividad.trace_enabled`; instructions block with `<pre className="whitespace-pre-wrap text-[#f0f0f0] text-sm">{actividad.instructions}</pre>` or equivalent; `<Link to={/student/cursos/${cursoId}/actividades} className="...">← Volver al curso</Link>`. Tailwind dark styling matching other pages. Depends on T004, T005.

---

## Phase 6 — Polish & Quality Gate

**Goal**: Ensure implementation quality before marking MW4 complete.

- [X] T009 Run React Doctor scan from repo root (`npx react-doctor@latest .`) — resolve ALL findings with ERROR severity before proceeding; for WARNING findings add a one-line code comment with justification if deferring.

- [X] T010 Audit all 3 pages — verify each of `DashboardPage`, `ActividadesPage`, and `ActividadDetailPage` shows: (a) a loading indicator during fetch, (b) an error message on API failure, (c) an appropriate empty state when the list is empty or activity not found. Update any missing or generic states.

- [X] T011 Verify role guard — sign in as a docente or admin and navigate directly to `http://localhost:5173/student/dashboard` → must be redirected to the role's own dashboard (not shown a blank page or 403); verify the `ProtectedRoute allowedRoles={['estudiante']}` block correctly rejects non-student roles.

---

## Dependencies

```
Phase 1 (Setup) → Phase 2 (Foundational) → Phases 3–5 (User Stories)

T001 [P]  ─────────────────────┐
T002 [P]  → T004 [P]  ──────────┤→ T006 [US1] → T007 [US2] → T008 [US3] → Phase 6
T003 [P]  → T005 [P]  ──────────┘

T001 + T002 + T003 can run in parallel (Phase 1 first wave)
T004 + T005 can run in parallel (Phase 2, after their respective prereqs)
T006, T007, T008 are sequential (each builds on previous for testability)
T009, T010, T011 sequential (Phase 6, after all pages done)
```

Within Phase 1:
```
T001 [P] + T002 [P] + T003 [P]  →  T004 [P] + T005 [P]  →  US phases
```

---

## Parallel Execution Examples

**Phase 1 first wave** (fully independent):
- T001 (delete old pages)
- T002 (types in academic.ts)
- T003 (stub page files)

**Phase 2 second wave** (after wave 1):
- T004 (student.ts — after T002)
- T005 (router — after T003)

**Phases 3–5** (sequential, each independently testable):
- T006 (DashboardPage — MVP)
- T007 (ActividadesPage)
- T008 (ActividadDetailPage)

**Phase 6** (after all pages done):
- T009, T010, T011 run sequentially

---

## Implementation Strategy

**MVP scope (US1 only):** Complete Phase 1 + Phase 2 + Phase 3 (T001–T006) to deliver a working student dashboard. Students can see their enrolled courses and navigate into them (even though the activity list page will just show the stub until T007).

**Incremental delivery order:**
1. Foundation (T001–T005) — unblocks all further work
2. US1 Dashboard (T006) — immediately useful; students can log in and see their courses
3. US2 Activity List (T007) — completes course navigation
4. US3 Activity Detail (T008) — enables students to read full instructions
5. Polish (T009–T011) — quality gate before marking MW4 done
