# Feature Specification: MW3 — Admin Panel

**Feature Branch**: `013-appeals-final-audit`

**Created**: 2026-05-26

**Status**: Draft

**Input**: User description: "Lets do the mw3"

## Clarifications

### Session 2026-05-26

- Q: How should materias be organized in the list view — flat table with a Carrera column, grouped by carrera, or split view? → A: Flat table with a "Carrera" column (Option A).
- Q: How does the admin identify a student when creating a student profile — UUID, email, or other? → A: The student's institutional email address (`@unifranz.edu.bo`). Confirmed from API source: the API requires email, rejects non-Unifranz addresses, and matches this email when the student later logs in via Google OAuth to auto-generate their course enrollments.

---

## Overview

Administrators need a dedicated area to manage the institutional catalog: academic careers (carreras), subjects (materias), and student profiles. This milestone gives administrators the complete CRUD lifecycle for these three catalog entities, which are prerequisites for docentes to create courses and for students to sign in via Google and be auto-enrolled.

**How student onboarding works**: The admin pre-provisions a student profile with the student's Unifranz institutional email, their carrera, and their current semester. When the student signs in for the first time using their `@unifranz.edu.bo` Google account, the system automatically creates their account and enrolls them in the relevant courses — no additional steps required.

---

## Actors

- **Admin** — primary actor; creates and manages all catalog entities in this milestone.
- **Docente** — indirect beneficiary; relies on materias existing before they can create courses (covered in MW2).
- **Student** — indirect beneficiary; relies on a student profile existing with their institutional email before they can sign in via Google and be auto-enrolled.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Manage Carreras (Priority: P1)

An administrator can view the list of academic careers (carreras) at the institution and add new ones with a short code and a name, update existing names, or deactivate careers that are no longer offered.

**Why this priority**: Carreras are the top-level organizational unit. Materias belong to carreras; without carreras, no subjects can be created, which blocks docentes from creating courses. It must exist before any other catalog data can be entered.

**Independent Test**: Can be fully tested by navigating to the Carreras page, creating a carrera with a code and name, verifying it appears in the list, editing its name, and deactivating it — all without touching materias or student profiles.

**Acceptance Scenarios**:

1. **Given** an authenticated admin, **When** they open the Carreras page, **Then** the full list of active carreras is displayed, each showing its short code and full name.
2. **Given** an admin on the Carreras page, **When** they provide a unique code (e.g., "IS") and a full name, and submit the form, **Then** the new carrera appears in the list.
3. **Given** an admin, **When** they submit a carrera creation form with a code that already exists, **Then** the system rejects the request and displays the conflict error inline.
4. **Given** an admin viewing an existing carrera, **When** they edit its name and save, **Then** the updated name is reflected in the list immediately.
5. **Given** an admin, **When** they attempt to deactivate a carrera that has active materias, **Then** the system rejects the request and displays the error message inline.
6. **Given** an admin, **When** they deactivate a carrera that has no active materias, **Then** the carrera is removed from the active list after a confirmation step.
7. **Given** no carreras exist, **When** the admin opens the Carreras page, **Then** a clear empty state with a "Create your first carrera" call-to-action is shown.

---

### User Story 2 — Manage Materias (Priority: P1)

An administrator can view all subjects (materias) in the catalog, create new subjects with a code, name, semester, and career assignment, update existing ones, and deactivate subjects that are no longer taught.

**Why this priority**: Materias are what docentes select when creating courses. Without materias, docentes cannot set up any courses, making the platform inoperable for teaching. P1 alongside carreras because they form an inseparable catalog foundation.

**Independent Test**: Can be fully tested by creating a materia linked to an existing carrera, verifying it appears in the flat list with its carrera column, editing it, and deactivating it.

**Acceptance Scenarios**:

1. **Given** an authenticated admin, **When** they open the Materias page, **Then** a flat table is shown with columns: Code, Name, Carrera, Semester — one row per materia.
2. **Given** an admin, **When** they provide a unique code, a name, a semester (1–10), select a carrera, and submit the creation form, **Then** the new materia appears in the table.
3. **Given** an admin, **When** they submit a materia creation form with a code that already exists, **Then** the system rejects the request and displays the conflict error inline.
4. **Given** an admin viewing an existing materia, **When** they edit its name or semester and save, **Then** the updated values are reflected in the table immediately.
5. **Given** an admin, **When** they attempt to deactivate a materia that has active courses linked to it, **Then** the system rejects the request and displays the error message inline.
6. **Given** an admin, **When** they deactivate a materia with no active courses, **Then** the materia is removed from the active list after a confirmation step.
7. **Given** no carreras exist, **When** the admin opens the Materias creation form, **Then** the carrera dropdown is empty, the submit button is disabled, and a message directs the admin to create a carrera first.
8. **Given** no materias exist, **When** the admin opens the Materias page, **Then** a descriptive empty state with a "Create your first materia" call-to-action is shown.

---

### User Story 3 — Manage Student Profiles (Priority: P2)

An administrator pre-provisions student records by entering each student's Unifranz institutional email, their carrera, their current semester, and optionally their student code. This record enables the student to sign in via Google and be auto-enrolled in the appropriate courses.

**Why this priority**: Student profiles are necessary for Google OAuth sign-in and auto-enrollment to work, but they can be created after the carrera and materia catalog is in place.

**Independent Test**: Can be fully tested by creating a student profile with a `@unifranz.edu.bo` email, a carrera, and a semester; verifying it appears in the list; editing the semester; and deactivating it.

**Acceptance Scenarios**:

1. **Given** an authenticated admin, **When** they open the Student Profiles page, **Then** all active student profiles are listed showing: email, carrera, semester, and student code (if set).
2. **Given** an admin, **When** they enter a `@unifranz.edu.bo` email, select a carrera, enter a semester (1–10), and submit the form, **Then** the new profile appears in the list.
3. **Given** an admin, **When** they submit a student profile form with a non-Unifranz email (e.g., `@gmail.com`), **Then** the system rejects the request and displays the validation error inline.
4. **Given** an admin, **When** they submit a student profile form for an email that already has a profile, **Then** the system rejects the request and displays the conflict error inline.
5. **Given** an admin viewing an existing profile, **When** they update the semester or student code and save, **Then** the updated values are reflected in the list immediately.
6. **Given** an admin, **When** they deactivate a student profile, **Then** a confirmation step is shown before the profile is removed from the active list.
7. **Given** no student profiles exist, **When** the admin opens the Student Profiles page, **Then** a descriptive empty state with a "Create your first student profile" call-to-action is shown.

---

### User Story 4 — Admin Dashboard Overview (Priority: P3)

An administrator landing on the admin dashboard sees a summary of the catalog with quick navigation links to each management section.

**Why this priority**: The dashboard provides orientation and quick navigation but is not required for catalog operations, which function independently through their own pages.

**Independent Test**: Can be tested by logging in as admin and confirming the dashboard renders with links to the three management pages.

**Acceptance Scenarios**:

1. **Given** an authenticated admin, **When** they land on the admin dashboard, **Then** they see quick navigation links to Carreras, Materias, and Student Profiles management pages.
2. **Given** any authenticated admin, **When** they click a navigation link on the dashboard, **Then** they are taken to the corresponding management page.

---

### Edge Cases

- What happens when the carrera dropdown on the Materias creation form is empty (no carreras exist yet)? → The submit button is disabled and a message directs the admin to create a carrera first.
- What happens when the system is unavailable during a deactivation operation? → The confirmation modal closes and the error is shown inline; the item remains in the list.
- What happens when an admin submits a creation form with a duplicate code? → The system returns a conflict error and the form displays it without navigating away.
- What happens when an admin enters a student email without the `@unifranz.edu.bo` domain? → The server rejects the request and the form displays the validation error.
- What happens when an admin navigates directly to a management page URL without being authenticated? → They are redirected to the sign-in screen.
- What happens when an authenticated non-admin user navigates to an admin page URL? → They are redirected to their own role's home screen.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST display the list of active carreras on the Carreras management page, showing each carrera's short code and full name.
- **FR-002**: System MUST allow an admin to create a new carrera by providing a unique short code and a full name; both fields are required.
- **FR-003**: System MUST allow an admin to update the name of an existing carrera; the code is immutable after creation.
- **FR-004**: System MUST require admin confirmation via a modal before deactivating a carrera.
- **FR-005**: System MUST display the server's error message inline if a deactivation is rejected (e.g., carrera has active materias).
- **FR-006**: System MUST display the list of active materias as a flat table with columns: Code, Name, Carrera, Semester.
- **FR-007**: System MUST allow an admin to create a new materia by providing a unique code, a name, a semester (1–10), and selecting a carrera; all four fields are required.
- **FR-008**: System MUST allow an admin to update the name and semester of an existing materia; the code and carrera are immutable after creation.
- **FR-009**: System MUST require admin confirmation via a modal before deactivating a materia.
- **FR-010**: System MUST display the list of active student profiles showing: email, carrera, semester, and student code (if present).
- **FR-011**: System MUST allow an admin to create a student profile by providing a `@unifranz.edu.bo` email, selecting a carrera, entering a semester (1–10), and optionally a student code; email, carrera, and semester are required.
- **FR-012**: System MUST reject student profile creation if the email does not end in `@unifranz.edu.bo` and display the validation error.
- **FR-013**: System MUST allow an admin to update the semester and student code of an existing student profile; email and carrera are immutable after creation.
- **FR-014**: System MUST require admin confirmation via a modal before deactivating a student profile.
- **FR-015**: System MUST redirect unauthenticated users attempting to access any admin page to the sign-in screen.
- **FR-016**: System MUST redirect authenticated non-admin users attempting to access any admin page to their own role's home screen.
- **FR-017**: System MUST show descriptive empty states with a relevant creation call-to-action when any catalog list is empty.

### Key Entities

- **Carrera**: An academic career or program. Has a unique short code (e.g., `"IS"`) and a full name. A carrera can have many materias. Deactivating a carrera is blocked if it has active materias.
- **Materia**: An academic subject within a career. Has a unique code, a name, a semester (1–10), and belongs to exactly one carrera. Deactivating a materia is blocked if it has active courses.
- **Student Profile**: A pre-provisioned enrollment record. Stores the student's `@unifranz.edu.bo` institutional email, their carrera, their current semester (1–10), and an optional student code. Used by the system to auto-enroll the student when they first sign in via Google OAuth.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An admin can complete the full catalog setup — create a carrera, create a materia under it, and create a student profile — in under 5 minutes without assistance.
- **SC-002**: All list pages load and display their content in under 3 seconds under normal conditions.
- **SC-003**: 100% of deactivation operations require an explicit confirmation step; no item is deactivated on a single click.
- **SC-004**: 100% of server-rejected operations surface the server's error message inline without losing the list state.
- **SC-005**: 0% of non-admin authenticated users can reach any admin management page.
- **SC-006**: All empty states include a visible call-to-action that navigates directly to the creation flow.

---

## Assumptions

- Carreras, materias, and student profiles are managed exclusively by admins; docentes and students have no write access to these entities.
- "Deleting" an entity means soft-deactivation (`is_active = false`) — records are not physically removed from the database. The UI reflects this by removing deactivated items from the active list.
- Carrera codes and materia codes are immutable after creation; only names (and semester for materias) can be updated.
- Student profile emails and carrera assignments are immutable after creation; only semester and student code can be updated.
- Pagination is not required for this milestone; all lists load in a single request.
- The platform runs in a browser environment; native mobile is out of scope.
- Student profiles represent pre-enrollment records, not user accounts. Deactivating a profile does not delete the student's login credentials.
- The `@unifranz.edu.bo` email domain restriction is enforced server-side; the UI displays whatever validation error the server returns.
