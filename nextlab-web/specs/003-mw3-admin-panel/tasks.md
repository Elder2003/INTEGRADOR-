# Tasks: MW3 — Admin Panel

**Feature**: MW3 — Admin Panel  
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/admin.md](contracts/admin.md)  
**Total tasks**: 18 | **Last updated**: 2026-05-26

---

## Phase 1 — Setup

**Goal**: Lay the shared foundation (types, API module, page stubs, router) before any user story work begins. All tasks in this phase must be complete before any Phase 2+ task starts.

- [x] T001 [P] Add MW3 types to `src/types/academic.ts` — first verify that existing `Materia` interface has `code: string` and `semester: number` fields (add them if missing); then add 3 new response interfaces: `Carrera { id, code, name, is_active, created_at, updated_at }`, `StudentProfile { id, email, carrera_id, semester, student_code: string | null, is_active, created_at, updated_at }`; and 6 new request interfaces: `CreateCarreraRequest { code, name }`, `UpdateCarreraRequest { name?, is_active? }`, `CreateMateriaRequest { carrera_id, code, name, semester }`, `UpdateMateriaRequest { name?, semester?, is_active? }`, `CreateStudentProfileRequest { email, carrera_id, semester, student_code? }`, `UpdateStudentProfileRequest { semester?, student_code?: string | null, is_active? }` — all shapes defined in data-model.md

- [x] T002 [P] Create minimal placeholder stubs for 3 new admin page files so the router can import them before full implementation: `src/pages/admin/CarrerasPage.tsx`, `src/pages/admin/MateriasPage.tsx`, `src/pages/admin/StudentProfilesPage.tsx` — each exports a default React component returning a single `<p>` with the page name; these will be replaced in Phase 2–4

- [x] T003 Create `src/api/admin.ts` — implement all 12 admin CRUD functions using the same `toApiError` helper pattern as `src/api/academic.ts`; import `api` from `../lib/api` and types from `../types/academic`; functions: `getCarreras(): Promise<Carrera[]>` → `GET /admin/carreras`, `createCarrera(data: CreateCarreraRequest): Promise<Carrera>` → `POST /admin/carreras` (201), `updateCarrera(id, data: UpdateCarreraRequest): Promise<Carrera>` → `PUT /admin/carreras/:id`, `deleteCarrera(id): Promise<void>` → `DELETE /admin/carreras/:id` (204); `getAdminMaterias(): Promise<Materia[]>` → `GET /admin/materias`, `createMateria(data: CreateMateriaRequest): Promise<Materia>` → `POST /admin/materias` (201), `updateMateria(id, data: UpdateMateriaRequest): Promise<Materia>` → `PUT /admin/materias/:id`, `deleteMateria(id): Promise<void>` → `DELETE /admin/materias/:id` (204); `getStudentProfiles(): Promise<StudentProfile[]>` → `GET /admin/student-profiles`, `createStudentProfile(data: CreateStudentProfileRequest): Promise<StudentProfile>` → `POST /admin/student-profiles` (201), `updateStudentProfile(id, data: UpdateStudentProfileRequest): Promise<StudentProfile>` → `PUT /admin/student-profiles/:id`, `deactivateStudentProfile(id): Promise<StudentProfile>` → `PUT /admin/student-profiles/:id` with body `{ is_active: false }` — **NOTE**: there is NO DELETE for student profiles; full signatures in contracts/admin.md. Depends on T001.

- [x] T004 Update `src/router/index.tsx` — add 3 child routes inside the existing admin `ProtectedRoute` children array, importing the stub pages from T002: `{ path: 'carreras', element: <CarrerasPage /> }`, `{ path: 'materias', element: <MateriasPage /> }`, `{ path: 'student-profiles', element: <StudentProfilesPage /> }`. Depends on T002.

---

## Phase 2 — US1: Manage Carreras (P1) 🎯 MVP

**Story goal**: Admin opens the Carreras page, sees the full list, can create a new carrera with a code + name, edit an existing carrera's name inline, and deactivate it via a confirm modal.

**Independent test criteria**: Navigate to `/admin/carreras` as admin → list renders from `GET /admin/carreras`; create form (code + name) submits and new carrera appears; edit name inline and save; confirm modal appears before deactivation; 409 conflict shown inline when deactivation is blocked; empty state visible when no carreras exist.

- [x] T005 [US1] Implement `src/pages/admin/CarrerasPage.tsx` — replace stub from T002 with full implementation. State: `items: Carrera[]`, `loading: boolean`, `error: string | null`, `actionError: string | null`, `editingId: string | null`, `editName: string`, `showCreateForm: boolean`, `createCode: string`, `createName: string`, `isSubmitting: boolean`, `showConfirm: boolean`, `selected: Carrera | null`. On mount: call `getCarreras()`, show `<LoadingSpinner />` while loading, show `<ErrorMessage>` on fetch failure. Create form (toggled via "Nueva Carrera" button): two inputs (Code, Name), both required; disable submit while either empty or `isSubmitting`; on submit call `createCarrera({ code, name })`, prepend result to `items`, reset form, hide form; on failure show `<ErrorMessage actionError>` below page heading. Table: one row per carrera — when `editingId === carrera.id` render inline edit form with Name input (code is read-only), Guardar/Cancelar buttons; on Guardar call `updateCarrera(id, { name: editName })`, update `items` in place, clear `editingId`; when not editing show Code, Name, "Editar" button (sets `editingId`), "Desactivar" button (sets `selected` + `showConfirm`). `<ConfirmModal isOpen={showConfirm} title="Desactivar carrera" message={"¿Desactivar " + selected?.name + "? Solo es posible si no tiene materias activas."} confirmLabel="Desactivar" isLoading={isSubmitting} onConfirm={handleDeactivate} onCancel={closeConfirm}>` — `handleDeactivate` calls `deleteCarrera(selected.id)`, removes from `items` on 204, on 409 closes modal and sets `actionError`; empty state when `items.length === 0`: "No hay carreras. ¡Crea la primera!" + button to show create form. Tailwind dark styling: `bg-[#1a1a1a] border border-[#2e2e2e]` table, `text-[#f0f0f0]` text, `bg-[#E85D24]` primary buttons. Depends on T003, T004.

---

## Phase 3 — US2: Manage Materias (P1)

**Story goal**: Admin opens the Materias page, sees a flat table (Code | Name | Semester | Carrera), creates a materia with 4 required fields (code, name, semester 1–10, carrera dropdown), edits name and semester inline, and deactivates.

**Independent test criteria**: Navigate to `/admin/materias` → flat table with Code, Name, Semester, Carrera columns; create form with 4 fields; carrera dropdown populated from `GET /admin/carreras`; if no carreras exist, submit is disabled with message; 409 shown for duplicate code; 409 shown for materia with active cursos.

- [x] T006 [US2] Implement `src/pages/admin/MateriasPage.tsx` — replace stub from T002. On mount: call `getAdminMaterias()` AND `getCarreras()` in parallel (both needed for table + dropdown). Create form fields: Code (text input), Name (text input), Semester (`<select>` with 10 options: `{Array.from({ length: 10 }, (_, i) => <option key={i+1} value={i+1}>{i+1}° Semestre</option>)}`), Carrera (`<select>` populated from `carreras` state); if `carreras.length === 0` disable submit and show: "No hay carreras disponibles. Crea una primero." with `<Link to="/admin/carreras">`; submit calls `createMateria({ code, name, semester: Number(semester), carrera_id })`, prepend to `items`, reset form. Table columns: Code, Name, Semester, Carrera name (look up `carrera.name` from local `carreras` array by `carrera_id`); inline edit row shows Name input + Semester select (both editable; code and carrera immutable); on Guardar call `updateMateria(id, { name, semester })`, update `items`; Desactivar opens `<ConfirmModal>` → calls `deleteMateria(id)` → removes from `items`; 409 shown inline via `actionError`; empty state when no materias. Same Tailwind styling as CarrerasPage. Depends on T003, T004.

---

## Phase 4 — US3: Manage Student Profiles (P2)

**Story goal**: Admin opens the Student Profiles page, sees list with email/carrera/semester/student_code columns, creates a profile with @unifranz.edu.bo email + carrera + semester (+ optional student_code), edits semester and student_code inline, and deactivates via confirm modal with "Desactivar" wording.

**Independent test criteria**: Navigate to `/admin/student-profiles` → list shows email, carrera, semester, student_code; create form submits; non-@unifranz.edu.bo email shows server validation error; duplicate email shows 409 inline; deactivate modal says "Desactivar" not "Eliminar"; edit saves semester/student_code; deactivated profile disappears from list.

- [x] T007 [US3] Implement `src/pages/admin/StudentProfilesPage.tsx` — replace stub from T002. On mount: call `getStudentProfiles()` AND `getCarreras()` in parallel (carrera dropdown + table display). Create form fields: Email (text input, `type="email"`), Carrera (`<select>` from carreras), Semester (`<select>` 1–10), Student Code (text input, optional); submit disabled while email, carrera_id, or semester empty; on submit call `createStudentProfile({ email, carrera_id, semester, student_code: studentCode || undefined })`, prepend to `items`, reset form; API error (including `"email must be a @unifranz.edu.bo address"` or `"already exists"`) shown via `actionError`. Table columns: Email, Carrera (looked up by carrera_id), Semester, Student Code (shows "—" if null); inline edit row: Semester select + Student Code input (email and carrera immutable); on Guardar call `updateStudentProfile(id, { semester, student_code })`, update `items`. Deactivate: `<ConfirmModal title="Desactivar perfil" message={"¿Desactivar el perfil de " + selected?.email + "?"} confirmLabel="Desactivar">` → calls `deactivateStudentProfile(id)` (`PUT { is_active: false }`), removes from `items` on success; failure shown via `actionError`; **NO** `deleteStudentProfile` — this is a `deactivateStudentProfile` call only. Depends on T003, T004.

---

## Phase 5 — US4: Admin Dashboard (P3)

**Story goal**: Admin lands on `/admin/dashboard` and sees navigation cards linking to the 3 management pages.

**Independent test criteria**: Log in as admin → `/admin/dashboard` shows 3 clickable navigation cards for Carreras, Materias, and Student Profiles; clicking each navigates to the correct page.

- [x] T008 [US4] Update `src/pages/admin/DashboardPage.tsx` — replace existing placeholder content with 3 navigation cards. Each card: `<Link to="/admin/carreras">` etc., using the same `bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 hover:border-[#E85D24] transition-colors` styling as `CourseCard`; card content: heading (e.g., "Carreras"), description (e.g., "Gestionar carreras académicas"); layout: `grid grid-cols-1 md:grid-cols-3 gap-4`; page heading: "Panel de Administración". No API calls needed — navigation only.

---

## Phase 6 — Polish & Quality Gate

**Goal**: Ensure implementation quality before marking MW3 complete.

- [x] T009 Run React Doctor scan from repo root (`npx react-doctor@latest .` or equivalent in `nextlab-web/`) — resolve ALL findings with ERROR severity before proceeding; for WARNING findings add a one-line code comment with justification if deferring
- [x] T010 Audit empty states across all 3 list pages — verify each of CarrerasPage, MateriasPage, and StudentProfilesPage shows a descriptive message and a relevant action button when the list is empty (no items returned by API); update any missing or generic empty states
- [x] T011 Audit ConfirmModal usage — verify `<ConfirmModal>` is used for: deactivate carrera (in CarrerasPage), deactivate materia (in MateriasPage), deactivate student profile (in StudentProfilesPage); confirm each modal has an appropriate `title` and `message`; specifically confirm StudentProfilesPage modal says "Desactivar" (not "Eliminar") since it uses PUT, not DELETE

---

## Dependencies

```
Phase 1 (Setup) → ALL other phases

T001 → T003 (api module needs types)
T002 → T004 (router needs stub imports)
T001 + T002 can run in parallel
T003 + T004 can run in parallel (after their respective prerequisites)

Phase 2 (US1) → depends on Phase 1 complete
Phase 3 (US2) → depends on Phase 1 complete; independent of Phase 2
Phase 4 (US3) → depends on Phase 1 complete; independent of Phases 2–3
Phase 5 (US4) → depends on T004 only (router + page stubs); can run in parallel with US1–US3
Phase 6 (Polish) → depends on all phases complete
```

Within Phase 1:
```
T001 [P] + T002 [P]  →  T003 [P] + T004 [P]  →  Phase 2+
```

---

## Parallel Execution Examples

**Phase 1 first wave** (fully independent):
- T001 (types in academic.ts)
- T002 (page stub files)

**Phase 1 second wave** (after wave 1):
- T003 (admin.ts — after T001)
- T004 (router — after T002)

**Phases 2–5** (after Phase 1 complete, fully parallel):
- T005 (CarrerasPage)
- T006 (MateriasPage)
- T007 (StudentProfilesPage)
- T008 (DashboardPage)

**Phase 6** (after all pages done):
- T009, T010, T011 can run sequentially

---

## Implementation Strategy

**MVP scope (US1 only):** Complete Phase 1 + Phase 2 (T001–T005) to deliver a working Carreras management page. This alone unblocks docentes from creating courses (they need materias, which need carreras).

**Incremental delivery order:**
1. Foundation (T001–T004) — unblocks all further work
2. US1 Carreras (T005) — immediately useful; docentes can start seeing a carrera dropdown
3. US2 Materias (T006) — completes the catalog needed for course creation
4. US3 Student Profiles (T007) — enables Google OAuth student onboarding
5. US4 Dashboard (T008) — additive, no regressions
6. Polish (T009–T011) — quality gate before marking MW3 done
