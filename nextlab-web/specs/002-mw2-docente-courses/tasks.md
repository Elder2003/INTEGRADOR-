# Tasks: MW2 — Docente Course Management

**Feature**: MW2 — Docente Course Management  
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/academic.md](contracts/academic.md)  
**Total tasks**: 24 | **Last updated**: 2026-05-25

---

## Phase 1 — Setup

**Goal**: Lay the shared foundation (types, API module, common component, page stubs, router) before any user story work begins. All tasks in this phase must be complete before any Phase 3+ task starts.

- [X] T001 Add MW2 types to src/types/index.ts — update `ActividadTipo` to exactly 5 values (`'practica' | 'tarea' | 'informe' | 'examen' | 'defensa_oral'`), add interfaces: `Materia`, `Curso`, `Actividad`, `Grupo`, `RosterEntry`, `CodocenteGrant`, `CreateCursoRequest`, `UpdateCursoRequest`, `CreateActividadRequest`, `UpdateActividadRequest`, `CreateGrupoRequest`, `UpdateGrupoRequest`, `GrantAccessRequest` — shapes defined in data-model.md
- [X] T002 Create src/api/academic.ts — implement all academic domain functions using the same `toApiError` helper pattern as src/api/auth.ts: `getMaterias()` returns `[]` on 403 (swallows the error, throws on all other failures), `getCursos / getCurso / createCurso / updateCurso / deleteCurso`, `getActividades(cursoId) / getActividad / createActividad / updateActividad / deleteActividad`, `getGrupos / createGrupo / updateGrupo / deleteGrupo`, `getRoster`, `addStudentToGrupo / removeStudentFromGrupo`, `grantAccess / revokeAccess` — full signatures and API paths in contracts/academic.md
- [X] T003 [P] Create src/components/common/ConfirmModal.tsx — props: `{ isOpen: boolean; title: string; message: string; confirmLabel?: string; isLoading: boolean; onConfirm: () => void; onCancel: () => void }` — renders nothing when `isOpen=false`; when `isLoading=true` disables both buttons and shows a spinner on the confirm button; backdrop click and Escape key call `onCancel`; no error display (parent shows errors via ErrorMessage after modal closes); Tailwind dark-surface overlay with `bg-black/60` backdrop
- [X] T004 [P] Create minimal placeholder stubs for 6 new docente page files so the router can import them before full implementation: `src/pages/docente/CursoDetailPage.tsx`, `src/pages/docente/CreateCursoPage.tsx`, `src/pages/docente/EditCursoPage.tsx`, `src/pages/docente/CreateActividadPage.tsx`, `src/pages/docente/ActividadDetailPage.tsx`, `src/pages/docente/EditActividadPage.tsx` — each exports a default React component returning a single `<p>` with the page name; these will be replaced in later phases
- [X] T005 [P] Create src/components/docente/ActividadTypeSelect.tsx — props: `{ value: ActividadTipo | ''; onChange: (v: ActividadTipo) => void; disabled?: boolean }` — renders a `<select>` with exactly 5 options: `practica` → "Práctica", `tarea` → "Tarea", `informe` → "Informe", `examen` → "Examen", `defensa_oral` → "Defensa Oral"; value equals the API enum string; Tailwind dark styling `bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0]`; when `disabled=true` adds `opacity-50 cursor-not-allowed`
- [X] T006 Update src/router/index.tsx — add 6 new child routes inside the existing `docente` ProtectedRoute children array, importing the stub pages from T004: `{ path: 'cursos/new', element: <CreateCursoPage /> }`, `{ path: 'cursos/:id', element: <CursoDetailPage /> }`, `{ path: 'cursos/:id/edit', element: <EditCursoPage /> }`, `{ path: 'cursos/:id/actividades/new', element: <CreateActividadPage /> }`, `{ path: 'cursos/:id/actividades/:aid', element: <ActividadDetailPage /> }`, `{ path: 'cursos/:id/actividades/:aid/edit', element: <EditActividadPage /> }` — CRITICAL: `cursos/new` MUST appear before `cursos/:id` to prevent "new" matching as a UUID param

---

## Phase 3 — US1: View my courses (P1)

**Story goal**: Docente opens the dashboard and sees all their courses in a list; empty state with CTA when none exist.

**Independent test criteria**: Navigate to `/docente/dashboard` as a logged-in docente → course list renders from `GET /cursos`; empty state visible when no courses exist; clicking a course card navigates to `/docente/cursos/:id`.

- [X] T007 [US1] Update src/pages/docente/DashboardPage.tsx — replace placeholder with real implementation: call `getCursos()` on mount, show `<LoadingSpinner />` while loading, map results to `<CourseCard>` with `onClick={() => navigate('/docente/cursos/' + c.id)}`, show empty state ("No tienes cursos aún. ¡Crea tu primer curso!") with a "Nuevo Curso" button (`→ /docente/cursos/new`) when array is empty, show `<ErrorMessage>` on failure; import `getCursos` from src/api/academic.ts and `CourseCard` from src/components/docente/CourseCard.tsx
- [X] T008 [P] [US1] Create src/components/docente/CourseCard.tsx — props: `{ curso: Curso; onClick: () => void }` — display `curso.materia_nombre` as heading, `curso.semester + ' · ' + curso.section` as subtitle, `curso.student_count + ' estudiantes'` count; entire card is a `<button>` calling `onClick`; Tailwind: `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 w-full text-left hover:border-[#E85D24] transition-colors`

---

## Phase 4 — US2: Create and manage a course (P1)

**Story goal**: Docente can create a new course (materia dropdown or UUID fallback), edit semester/section of an existing course, and delete a course from its detail page.

**Independent test criteria**: Submit the creation form → course appears in dashboard list; edit form pre-fills from GET /cursos/:id and saves via PUT; delete confirm modal calls DELETE and navigates back on success; API rejection (409 has_submissions) shown inline via ErrorMessage.

- [X] T009 [US2] Implement src/pages/docente/CreateCursoPage.tsx — on mount call `getMaterias()`; if result is non-empty render a `<select>` for `materia_id` populated from the Materia array; if result is empty render a text input for `materia_id` with a note "Contacta a tu admin para obtener el ID de materia"; text inputs for `semester` and `section`; disable submit button until all three fields are non-empty; on submit call `createCurso(data)` then navigate to `/docente/cursos/${newCurso.id}?tab=actividades`; cancel button navigates back to `/docente/dashboard`; show `<ErrorMessage>` on API failure
- [X] T010 [P] [US2] Implement src/pages/docente/EditCursoPage.tsx — on mount call `getCurso(id)` from `useParams()` to pre-fill `semester` and `section` inputs; show materia name as read-only text (not editable); on submit call `updateCurso(id, { semester, section })` then navigate back to `/docente/cursos/${id}`; cancel navigates back to `/docente/cursos/${id}`; show `<ErrorMessage>` on API failure; show `<LoadingSpinner />` during initial fetch

---

## Phase 5 — US5: Create and manage activities (P1)

**Story goal**: Docente navigates to the dedicated activity creation page, fills the form, and the activity appears in the Actividades tab after creation. Edit form locks title/tipo when submissions exist.

**Independent test criteria**: POST to `/cursos/:id/actividades` creates the activity and redirects to `?tab=actividades`; 5 tipo options shown in select; datetime-local value converted to ISO 8601 before POST; edit form shows `has_submissions` lock state on title and tipo.

- [X] T011 [US5] Implement src/pages/docente/CreateActividadPage.tsx — `cursoId` from `useParams()`; form fields: title (text input), tipo (`<ActividadTypeSelect>` from T005), deadline (`<input type="datetime-local">`), instructions (`<textarea>`); convert deadline to ISO 8601 before posting: `deadlineLocal + ':00Z'`; disable submit until all fields non-empty; on submit call `createActividad(cursoId, data)` then navigate to `/docente/cursos/${cursoId}?tab=actividades`; back button navigates to `/docente/cursos/${cursoId}?tab=actividades`; show `<ErrorMessage>` on failure
- [X] T012 [P] [US5] Implement src/pages/docente/EditActividadPage.tsx — `cursoId` and `aid` from `useParams()`; on mount call `getActividad(aid)` to pre-fill all fields; `title` input and `<ActividadTypeSelect>` disabled when `actividad.has_submissions === true` with note "Bloqueado: existen entregas"; `deadline`, `instructions`, `late_submission_allowed` always editable; on submit call `updateActividad(aid, data)` then navigate to `/docente/cursos/${cursoId}/actividades/${aid}`; cancel navigates back to same detail URL; show `<ErrorMessage>` on failure including 422 `field_locked` message from API

---

## Phase 6 — US3: View course detail with tabs (P2)

**Story goal**: Docente can view a course's full detail across three tabs with URL-persisted active state.

**Independent test criteria**: URL `?tab=grupos` shows Grupos tab content; browser back from activity creation restores `?tab=actividades`; Actividades tab lists activities with delete option; Estudiantes tab shows read-only roster; Grupos tab shows group list.

- [X] T013 [US3] Implement src/pages/docente/CursoDetailPage.tsx — base structure: `cursoId` from `useParams()`; on mount call `getCurso(cursoId)`, show `<LoadingSpinner />` then display `curso.materia_nombre + ' · ' + curso.semester + ' · ' + curso.section` as page heading; `useSearchParams()` for tab state — default tab `'actividades'` when param is absent or invalid; three tab buttons updating URL via `setSearchParams({ tab }, { replace: false })`; "Editar curso" button → `/docente/cursos/${cursoId}/edit`; "Eliminar" button opens `<ConfirmModal>` → calls `deleteCurso(cursoId)` → navigate to `/docente/dashboard` on success; on 409 or other error close modal and show `<ErrorMessage>` inline; each tab renders its own section below the tab bar (implement as `{activeTab === 'actividades' && <ActividadesTab />}` pattern)
- [X] T014 [US3] Add Actividades tab section to CursoDetailPage.tsx — implement as an inline section or extracted component; on tab activation call `getActividades(cursoId)`, show `<LoadingSpinner />`, then list each actividad with: title, tipo badge (`bg-[#E85D24]/20 text-[#E85D24] text-xs px-2 py-0.5 rounded`), formatted deadline; "Ver detalle" link → `/docente/cursos/${cursoId}/actividades/${aid}`; "Eliminar" button per row opens `<ConfirmModal>` → calls `deleteActividad(aid)` → removes from list on success or shows inline error; empty state: "No hay actividades. Crea la primera." with "Nueva Actividad" button → `/docente/cursos/${cursoId}/actividades/new`
- [X] T015 [P] [US3] Add Estudiantes tab section to CursoDetailPage.tsx — on tab activation call `getRoster(cursoId)`, show `<LoadingSpinner />`, render a table with columns: Email, Fecha de inscripción (formatted date), Grupo (grupo_nombre or "—"); table is read-only (no add/remove student controls); empty state: "No hay estudiantes inscritos aún"
- [X] T016 [P] [US3] Add Grupos tab section to CursoDetailPage.tsx — on tab activation call `getGrupos(cursoId)`, show `<LoadingSpinner />`, render list of groups each showing: nombre, `member_count + '/' + (capacity ?? '∞') + ' miembros'`; empty state: "No hay grupos. Crea el primero."; group CRUD controls are added in Phase 7 (T017–T018)

---

## Phase 7 — US4: Manage groups (P2)

**Story goal**: Docente can create, rename, change capacity, delete groups, and assign/remove students from groups.

**Independent test criteria**: Create group form → POST `/cursos/:id/grupos` → group appears in list; delete shows ConfirmModal; student dropdown in Estudiantes tab assigns student → group member count updates.

- [X] T017 [US4] Add group create/edit form to Grupos tab in CursoDetailPage.tsx — "Nuevo Grupo" button toggles an inline form with: nombre (text input, required), capacity (number input, optional); submit calls `createGrupo(cursoId, { nombre, capacity })` and appends to local group list; each group row has an "Editar" button that opens an inline edit form pre-filled with current values; save calls `updateGrupo(cursoId, grupo.id, { nombre, capacity })` and updates local state; cancel collapses the form; show `<ErrorMessage>` on API failure
- [X] T018 [US4] Add group delete to Grupos tab in CursoDetailPage.tsx — each group row has a "Eliminar" button; opens `<ConfirmModal title="Eliminar grupo" message="¿Eliminar este grupo? Esta acción no se puede deshacer."`; on confirm calls `deleteGrupo(cursoId, grupo.id)` → removes from local state on success; on API failure (e.g. 409 has_members) close modal and show `<ErrorMessage>` inline in the Grupos tab
- [X] T019 [US4] Add student-to-group assignment to Estudiantes tab in CursoDetailPage.tsx — in the Grupo column of the roster table, render a `<select>` per student with options: "Sin grupo" (value = '') + one option per group (value = grupo.id, label = grupo.nombre); on change: if selecting a group call `addStudentToGrupo(cursoId, grupoId, userId)`, if deselecting (to "Sin grupo") call `removeStudentFromGrupo(cursoId, currentGrupoId, userId)`; update local roster state on success to reflect the new group assignment; show `<ErrorMessage>` inline on failure

---

## Phase 8 — US6: View activity detail (P2)

**Story goal**: Docente can view the full configuration of an activity, toggle late submissions, edit, and delete.

**Independent test criteria**: GET `/actividades/:aid` renders all fields; toggle late submissions calls PUT; edit button navigates to edit page; delete shows ConfirmModal.

- [X] T020 [US6] Implement src/pages/docente/ActividadDetailPage.tsx — `cursoId` and `aid` from `useParams()`; on mount call `getActividad(aid)`, show `<LoadingSpinner />` then display: title as page heading, tipo badge, deadline (formatted with date + time), instructions, `late_submission_allowed` toggle (`<button>` toggling a boolean, calls `updateActividad(aid, { late_submission_allowed: !current })` on click, optimistic update with rollback on error); "Editar" button → `/docente/cursos/${cursoId}/actividades/${aid}/edit`; "Eliminar" button opens `<ConfirmModal>` → calls `deleteActividad(aid)` → navigate to `/docente/cursos/${cursoId}?tab=actividades` on success; on failure close modal and show `<ErrorMessage>` inline

---

## Phase 9 — US7: Grant and revoke co-docente access (P3)

**Story goal**: Docente can add a co-docente by user ID and immediately see and revoke grants within the same page session.

**Independent test criteria**: POST `/actividades/:aid/grants` response appended to list; DELETE `/actividades/:aid/grants/:uid` removes from list; page refresh clears the list and shows "Grants refresh on reload" note.

- [X] T021 [US7] Add co-docente grants section to src/pages/docente/ActividadDetailPage.tsx — below the activity detail add a "Acceso co-docente" section; local state `grants: CodocenteGrant[]` initialized to `[]`; text input for `userId` + "Otorgar acceso" button; on click call `grantAccess(aid, userId)` → append returned grant to state → clear input; render grants list showing `grant.user_id` + "Revocar" button per entry; "Revocar" calls `revokeAccess(aid, grant.user_id)` → filter out of state; below the list show: "La lista de co-docentes se restablece al recargar la página"; show `<ErrorMessage>` on grant/revoke failure

---

## Phase 10 — Polish & Quality Gate

**Goal**: Ensure the implementation meets quality standards and all edge cases from the spec are covered before marking MW2 complete.

- [X] T022 Run React Doctor scan from repo root (`npx react-doctor@latest .`) — resolve ALL findings with ERROR severity before proceeding; for WARNING findings add a one-line code comment with justification at the affected site if deferring
- [X] T023 Audit empty states across all 4 lists — verify each list (courses in DashboardPage, activities in Actividades tab, students in Estudiantes tab, groups in Grupos tab) shows a descriptive message and a relevant action button when empty; update any generic or missing empty states
- [X] T024 Audit ConfirmModal usage — verify `<ConfirmModal>` is used for all 3 destructive actions: delete curso (in CursoDetailPage), delete grupo (in Grupos tab), delete actividad (in Actividades tab and ActividadDetailPage); confirm each modal has an appropriate `title` and `message` describing what will be deleted

---

## Dependencies

```
Phase 1 (Setup) → ALL other phases

Phase 3 (US1) → no inter-US dependencies
Phase 4 (US2) → depends on Phase 1 (needs getCurso from api/academic.ts)
Phase 5 (US5) → depends on Phase 1 (needs createActividad, ActividadTypeSelect)
Phase 6 (US3) → depends on Phase 3 (DashboardPage), Phase 5 (links to CreateActividadPage)
Phase 7 (US4) → depends on Phase 6 (US3) — group CRUD lives inside CursoDetailPage
Phase 8 (US6) → depends on Phase 5 (US5) — detail page complements edit page
Phase 9 (US7) → depends on Phase 8 (US6) — grants added to ActividadDetailPage
Phase 10 (Polish) → depends on all phases complete
```

Within Phase 1:
```
T001 → T002 → [T003, T004, T005 in parallel] → T006
```

---

## Parallel Execution Examples

**Phase 1 after T001+T002:**
- T003 (ConfirmModal), T004 (page stubs), T005 (ActividadTypeSelect) can run simultaneously

**Phase 3:**
- T007 (DashboardPage) and T008 (CourseCard) — T008 first, T007 imports it

**Phase 4:**
- T009 (CreateCursoPage) and T010 (EditCursoPage) — independent files, fully parallel

**Phase 5:**
- T011 (CreateActividadPage) and T012 (EditActividadPage) — independent files, fully parallel

**Phase 6:**
- T013 (CursoDetailPage base + Actividades tab) must complete before T014
- T015 (Estudiantes tab) and T016 (Grupos tab) can run in parallel after T013

---

## Implementation Strategy

**MVP scope (US1 + US2 + US5):** Complete Phases 1–5 (T001–T012) to deliver:
- Course list dashboard
- Course creation with materia selection or UUID fallback
- Course edit
- Activity creation and edit with field locking

This MVP is independently usable and verifiable. US3/US4 (tabbed detail + group management) can be layered on top without breaking the MVP.

**Incremental delivery order:**
1. Foundation + router (T001–T006) — unblocks all further work
2. US1 course list (T007–T008) — visible to docente immediately
3. US2 course CRUD (T009–T010) — full course lifecycle
4. US5 activity CRUD (T011–T012) — activities manageable via direct URL
5. US3 tabbed detail (T013–T016) — ties course + activity navigation together
6. US4 group management (T017–T019) — adds group layer to tabbed detail
7. US6 activity detail (T020) — full activity workflow
8. US7 co-docente grants (T021) — additive, no regressions
9. Polish + React Doctor gate (T022–T024)
