# Feature Specification: MW2 — Docente Course Management

**Feature**: MW2 — Docente Course Management  
**Date**: 2026-05-24  
**Status**: Draft  
**Priority**: P1

---

## Clarifications

### Session 2026-05-24

- Q: Should the active tab on the course detail screen persist through browser back-navigation? → A: Yes — encode the active tab as a URL query param (`?tab=actividades` etc.) so back-button navigation and shareable links both work correctly.
- Q: Should the activity creation form open as a modal overlay or navigate to a separate page? → A: Separate page at `/docente/cursos/:id/actividades/new`; pressing back returns to the Actividades tab.

---

## Overview

Docentes need a dedicated area to manage their academic courses, organise students into groups, and publish activities. This milestone gives docentes the complete course lifecycle: create courses, view enrolled students, manage groups, create and configure activities, and grant co-teaching access to co-docentes.

This milestone consumes the backend's course and activity endpoints (M2/M3) and results in a working docente dashboard where all course management tasks can be completed without leaving the app.

---

## Actors

- **Docente** — primary actor; creates and manages all content in this milestone
- **Co-docente** — secondary actor; receives and can view activities they have been granted access to (read-only, no management UI in this milestone)

---

## User Stories

### US1 — View my courses (P1)
**As a docente**, I want to see a list of all my active courses so that I know what I am teaching this semester.

**Acceptance criteria:**
1. After logging in, the docente sees a dashboard listing all courses assigned to them.
2. Each course shows: materia name, semester label, section, and number of enrolled students.
3. If no courses exist yet, a clear empty state is shown with a call-to-action to create one.
4. Clicking a course navigates to the course detail screen.

---

### US2 — Create and manage a course (P1)
**As a docente**, I want to create a new course linked to a materia so that I can start enrolling students and publishing activities.

**Acceptance criteria:**
1. A "New course" button is visible on the dashboard.
2. The creation form asks for: materia (selected from the available list), semester label, and section.
3. Submitting the form creates the course and navigates to its detail screen.
4. The docente can edit the semester label or section of an existing course.
5. The docente can delete a course that has no enrolled students (the API enforces this constraint; the UI shows the API's error message if the delete is rejected).

---

### US3 — View course detail with tabs (P2)
**As a docente**, I want to see all relevant information about a course in one place, organised into tabs, so that I can switch between managing activities, students, and groups without losing context.

**Acceptance criteria:**
1. The course detail screen has three tabs: **Actividades**, **Estudiantes**, **Grupos**.
2. The Actividades tab shows the list of activities for the course (see US5).
3. The Estudiantes tab shows the student roster: email, enrollment date, and which group they belong to (if any).
4. The Grupos tab shows all groups for the course with member counts.
5. The active tab is encoded in the URL as a query parameter (`?tab=actividades` | `?tab=estudiantes` | `?tab=grupos`), so it survives browser back-navigation and the URL is shareable. Navigating to an activity and pressing back restores the correct tab.

---

### US4 — Manage groups (P2)
**As a docente**, I want to create and manage groups within a course so that I can organize students into teams for group activities.

**Acceptance criteria:**
1. From the Grupos tab, the docente can create a new group with a name and optional capacity.
2. Existing groups can be renamed or have their capacity changed.
3. A group can be deleted only if it has no members (the API enforces this; the UI shows the error).
4. From the Grupos tab or the Estudiantes tab, the docente can assign a student to a group.
5. A student can be removed from a group.
6. Each group shows its current member count and capacity (if set).

---

### US5 — Create and manage activities (P1)
**As a docente**, I want to create activities for a course so that students receive assignments to complete.

**Acceptance criteria:**
1. From the Actividades tab, a "New activity" button navigates to a dedicated creation page (`/docente/cursos/:id/actividades/new`). The form collects: title, activity type (tarea, examen, informe, practica, investigacion, git_analysis, defensa_oral), deadline, and instructions.
2. After creation, the user is redirected back to the course detail page (Actividades tab) and the new activity appears in the list.
3. Clicking an activity navigates to the activity detail screen.
4. The docente can edit any field of an existing activity.
5. The docente can delete an activity; the app asks for confirmation before sending the delete request.
6. The activity type cannot be changed after creation (the API enforces this; the UI disables the type field on edit).

---

### US6 — View activity detail (P2)
**As a docente**, I want to view the full configuration of an activity so that I can confirm it is set up correctly before students see it.

**Acceptance criteria:**
1. The activity detail screen shows: title, type badge, deadline, instructions, and whether late submissions are allowed.
2. The docente can toggle "allow late submissions" from this screen.
3. An edit button opens the activity edit form (reusing the creation form with pre-filled values).

---

### US7 — Grant and revoke co-docente access (P3)
**As a docente**, I want to grant a co-docente access to a specific activity so that they can assist with reviewing and grading.

**Acceptance criteria:**
1. From the activity detail screen, the docente can add a co-docente by entering their user ID.
2. Granted co-docentes appear in a list on the activity detail screen.
3. The docente can revoke access; the co-docente entry is removed from the list immediately.
4. If the user ID does not correspond to a co-docente account, the API returns an error and the UI displays it.

---

## Functional Requirements

**FR-001**: The docente dashboard (GET /cursos) displays all courses belonging to the authenticated docente, ordered by most recently created.

**FR-002**: The course creation form (POST /cursos) requires materia selection from a dropdown populated by calling GET /admin/materias; materias are read-only data created by admin.

**FR-003**: The student roster (GET /cursos/:id/students) shows each enrolled student's email, enrollment date, and group assignment; pagination is not required for this milestone.

**FR-004**: Group management (CRUD on /cursos/:id/grupos) and student-to-group assignment (POST/DELETE /cursos/:id/grupos/:gid/students) are all done via the existing API endpoints; no client-side state management of group membership.

**FR-005**: Activity creation uses a dedicated route (`/docente/cursos/:id/actividades/new`). The form POSTs to /cursos/:id/actividades with title, tipo, deadline (ISO 8601), and instructions; on success the user is redirected to `/docente/cursos/:id?tab=actividades`. The frontend does not validate field contents beyond "non-empty" — all business rules are enforced by the API.

**FR-006**: The Confirm before delete modal (`ConfirmModal` component) is used for all destructive actions: delete course, delete group, delete activity.

**FR-007**: Co-docente grants (POST/DELETE /actividades/:aid/grants) use a simple input for the co-docente user ID; no user search UI is needed in this milestone.

**FR-008**: All API error responses are displayed via the existing `ErrorMessage` component; no error-specific branching in the UI beyond showing the message the API returns.

**FR-009**: The activity type field uses a `<select>` with the seven valid values: tarea, examen, informe, practica, investigacion, git_analysis, defensa_oral.

---

## Edge Cases / Error Handling

- **Empty states**: Each list (courses, activities, students, groups) shows a descriptive empty-state message with a relevant action button when no items exist.
- **Delete rejected by API**: When the API rejects a delete (e.g., course with enrolled students), the confirmation modal closes and the API error message is shown inline.
- **Materia dropdown empty**: If no materias exist yet, the course creation form shows "No materias available — ask an admin to create one" and disables the submit button.
- **Deadline in the past**: The API may reject a deadline in the past; the UI shows the API's error without adding its own validation.
- **Network errors**: All forms handle network errors via the existing `ErrorMessage` component with the standard "Unable to reach the server" message.

---

## Scope Boundaries

**In scope:**
- Docente CRUD for courses, groups, and activities
- Student roster view (read-only listing)
- Group membership management (assign/remove students)
- Co-docente grant management
- `ConfirmModal` component shared across the app

**Out of scope (later milestones):**
- Student-facing views of courses and activities (MW4)
- File submissions (MW5)
- Grading (MW7)
- Activity analytics (MW9)
- Admin creation of carreras/materias (MW3)

---

## Success Criteria

1. A docente can log in and see their course list in under 2 seconds.
2. A docente can complete the full flow — create a course, add a student to a group, and create an activity — in under 5 minutes without help.
3. All destructive actions require an explicit confirmation step before execution.
4. API errors are always visible to the user; no silent failures.
5. React Doctor scan reports zero ERRORs after MW2 is complete.

---

## Assumptions

1. Materias are already present in the system (created by admin); docentes select from existing materias — they do not create materias.
2. Students are already enrolled in a course before the docente sees them in the roster; enrollment is handled at the database level (the API endpoint returns whoever is enrolled).
3. The co-docente user ID is known to the docente (no user search). A search feature may be added in a later milestone.
4. Activity type (`tipo`) is immutable after creation — the edit form disables the select element for existing activities.
5. Pagination is not required for this milestone; all lists load in a single request.

---

## Dependencies

- MW1 (Foundation & Auth) must be complete — this milestone uses `AuthContext`, `lib/api.ts`, `ErrorMessage`, `LoadingSpinner`, `Layout`, `ProtectedRoute`, and the role-gated router.
- The backend API (M2/M3 endpoints) must be running and accessible at `VITE_API_URL`.
