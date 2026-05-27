# Feature Specification: MW4 — Student Portal

**Feature Branch**: `013-appeals-final-audit`

**Created**: 2026-05-26

**Status**: Draft

**Input**: User description: "lets do the mw4"

## Clarifications

### Session 2026-05-26

- Q: How should the activity detail page get its data, given there is no `GET /me/cursos/:id/actividades/:aid` endpoint? → A: Re-fetch `GET /me/cursos/:id/actividades` on the detail page and find the entry by `:aid`. Direct URL navigation works; no redirect-on-missing-state required.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Enrolled Courses (Priority: P1)

A student logs in and is taken to their dashboard. The dashboard shows all courses they are currently enrolled in — including the subject name, docente's email, the semester label, the section, and their assigned group (if any). If they have no enrollments yet, a clear empty state is displayed.

**Why this priority**: This is the entry point for the entire student experience. Without seeing their courses, students cannot navigate to activities, submit work, or track grades. The admin pre-provisions student profiles and docentes enroll students in groups; MW4 surfaces this to the student.

**Independent Test**: Navigate to `/student/dashboard` as a student with active inscripciones → course list renders from `GET /me/cursos`; each card shows materia name, docente email, semester label, section, and grupo name (or "Sin grupo"); empty state shown when no enrollments.

**Acceptance Scenarios**:

1. **Given** a student is logged in with active course enrollments, **When** they navigate to the student dashboard, **Then** they see one card per enrolled course showing: subject name, docente email, semester label, section, and grupo name (or "Sin grupo").
2. **Given** a student is logged in but has no enrollments, **When** they navigate to the dashboard, **Then** they see an empty state message: "Aún no estás inscrito en ningún curso."
3. **Given** a student has enrollments, **When** the page loads, **Then** courses load in under 3 seconds and a loading indicator is shown during fetch.
4. **Given** the API returns an error, **When** the course list fails to load, **Then** an error message is displayed with the API's error text.

---

### User Story 2 - View Course Activities (Priority: P2)

A student clicks on one of their enrolled courses and sees a list of all published activities for that course. Each activity shows its title, type (práctica, tarea, informe, examen, defensa oral), deadline, and whether trace monitoring is required. The activity list is read-only; students cannot edit or delete activities.

**Why this priority**: Viewing activities is the core interaction before submitting work. Students must know what tasks are assigned, when they are due, and whether a monitored session is required (trace_enabled).

**Independent Test**: Navigate to `/student/cursos/:id/actividades` as an enrolled student → activity list renders from `GET /me/cursos/:id/actividades`; each item shows title, tipo badge, formatted deadline, and a "monitoreado" indicator when trace_enabled; accessing a course the student is not enrolled in redirects or shows a forbidden message.

**Acceptance Scenarios**:

1. **Given** a student is enrolled in a course, **When** they navigate to that course's activity list, **Then** they see all activities with title, tipo label, deadline, and a monitoring indicator if trace_enabled is true.
2. **Given** a course has no activities, **When** the student opens the course, **Then** they see an empty state: "Este curso aún no tiene actividades publicadas."
3. **Given** a student tries to access a course they are NOT enrolled in, **When** the API returns 403, **Then** they see a "No tienes acceso a este curso" message and a link back to their dashboard.
4. **Given** an activity has a past deadline, **When** the list renders, **Then** the deadline is visually distinguished (e.g., shown in muted color or marked as "Vencido").

---

### User Story 3 - View Activity Detail (Priority: P3)

A student clicks on an activity to open a detail page. The detail page shows the full instructions, the tipo, the deadline, and whether the activity requires trace monitoring. This is a read-only view; submitting work is handled by MW5.

**Why this priority**: Students need to read the full instructions before starting work. The activity list shows a summary; the detail page shows the complete picture. Trace monitoring being visible here sets the student's expectation before they open a session in MW6.

**Independent Test**: Navigate to `/student/cursos/:id/actividades/:aid` → detail page re-fetches `GET /me/cursos/:id/actividades`, finds the entry by `:aid`, and renders title, tipo, full instructions, formatted deadline, and monitoring badge; direct URL navigation (e.g., bookmarked link) works correctly without redirect; a "Volver" link returns to the activity list.

**Acceptance Scenarios**:

1. **Given** a student navigates to an activity detail URL, **When** the page loads, **Then** they see: title, tipo, full instructions, formatted deadline, and a "Requiere monitoreo" badge (or none if trace_enabled is false).
2. **Given** the activity detail page is open, **When** the student clicks "Volver al curso", **Then** they are taken back to the activity list for that course.
3. **Given** the student directly navigates to `/student/cursos/:id/actividades/:aid` (e.g., via bookmark), **When** the page loads, **Then** the app re-fetches the activity list and finds the matching entry by ID — no redirect required.
4. **Given** the `:aid` does not match any activity in the fetched list (not found or 403), **When** the page loads, **Then** a not-found/forbidden message is shown with a link back to the activity list.

---

### Edge Cases

- What happens when the student's token expires mid-page? → The axios interceptor handles 401 silently (token refresh or redirect to login) — handled by existing MW1 infrastructure.
- What if `grupo` is null? → Display "Sin grupo" in the course card.
- What if `deadline` is in the past? → Display it in muted color or with a "Vencido" suffix; no submission blocking (that's MW5/backend responsibility).
- What if a student types `/student/cursos/:id/actividades` for a course they're not enrolled in? → API returns 403; page shows access-denied message.
- What if materia or docente data is missing (empty strings)? → Display whatever the API returns; no client-side fallback logic.
- What if the `:aid` in the URL does not match any activity in the re-fetched list? → Display a not-found message with a link back to the activity list.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display all active course enrollments for the authenticated student on the student dashboard, fetched from `GET /me/cursos`.
- **FR-002**: Each course card MUST show: subject name (materia_name), docente email (docente_email), semester label (semester_label), section, and grupo name (or "Sin grupo" if null).
- **FR-003**: System MUST display a loading indicator while course data is being fetched.
- **FR-004**: System MUST display an empty state ("Aún no estás inscrito en ningún curso.") when the course list is empty.
- **FR-005**: Clicking a course card MUST navigate to the course activity list at `/student/cursos/:id/actividades`.
- **FR-006**: System MUST display all activities for the selected course, fetched from `GET /me/cursos/:id/actividades`.
- **FR-007**: Each activity row MUST show: title, tipo label, formatted deadline, and a monitoring indicator when trace_enabled is true.
- **FR-008**: System MUST display an empty state when a course has no activities.
- **FR-009**: System MUST display an access-denied message when the student is not enrolled in the requested course (API returns 403).
- **FR-010**: Clicking an activity MUST navigate to the activity detail page at `/student/cursos/:id/actividades/:aid`.
- **FR-011**: The activity detail page MUST re-fetch `GET /me/cursos/:id/actividades` on load, find the entry matching `:aid`, and display: title, tipo, full instructions, formatted deadline, and a monitoring badge (if trace_enabled is true). This ensures direct URL navigation works without relying on route state.
- **FR-012**: The activity detail page MUST include a "Volver" link back to the course activity list.
- **FR-013**: All student routes MUST be protected under the `estudiante` role guard — non-students must be redirected to their own dashboard.
- **FR-014**: Deadline display MUST visually distinguish past deadlines from future ones (muted color or "Vencido" label).
- **FR-015**: If the `:aid` does not match any activity in the re-fetched list, the detail page MUST display a not-found message with a link back to the activity list.

### Key Entities

- **StudentCursoEntry**: Enrolled course view — id, materia_name, docente_email, semester_label, section, grupo (nullable object with id and name).
- **StudentActividadEntry**: Activity summary/detail — id, title, tipo, deadline, instructions, trace_enabled, posted_at, last_edited_at.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Students can navigate from login to their course list in under 3 seconds on a standard connection.
- **SC-002**: Students can reach an activity detail page from the dashboard in 2 clicks or fewer.
- **SC-003**: All student portal routes correctly enforce role isolation — a docente or admin navigating to `/student/...` is redirected to their own dashboard.
- **SC-004**: Empty states and error messages are present on all 3 pages (dashboard, activity list, activity detail) — zero "blank page" scenarios.
- **SC-005**: The activity list correctly shows a monitoring indicator for all activities with trace_enabled = true and hides it for trace_enabled = false.
- **SC-006**: The activity detail page works correctly when navigated to directly via bookmark or shared URL (no reliance on navigation state).

## Assumptions

- Student enrollments are created by the admin (via student profiles + grupo assignments in MW3) and by docentes assigning students to grupos — the student portal displays what the API returns without modification.
- `trace_enabled` on an activity is a read-only flag set by the docente; students see it but cannot change it. Starting a monitored session (MW6) will be a separate action on the detail page.
- The activity detail page in MW4 is read-only — the submit-work button is out of scope (MW5).
- The student portal UI reuses `LoadingSpinner`, `ErrorMessage`, and `ConfirmModal` from `src/components/common/` per MW3 patterns.
- The existing mocked pages in `src/pages/student/` (Dashboard.tsx, Exam.tsx, etc.) are from the previous Bloque 4 scaffold and will be replaced by the real API-connected pages in MW4.
- `deadline` is an ISO timestamp string; the frontend formats it for display only (e.g., "26 may 2026 · 23:59") — no deadline-enforcement logic on the client.
- There is no pagination — `GET /me/cursos` and `GET /me/cursos/:id/actividades` return the full list; the API does not paginate these responses.
- The activity detail page fetches `GET /me/cursos/:id/actividades` (not a single-activity endpoint, which does not exist for students) and locates the activity by `:aid`. This is the sole data-fetching strategy for the detail page.
