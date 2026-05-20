# Feature Specification: M3 — Academic Management

**Feature Branch**: `003-academic-management`

**Created**: 2026-05-19

**Status**: Draft

**Input**: User description: "I want to do the Milestone 3"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Docente Manages a Course (Priority: P1)

A docente creates a new course for the current semester, associates it with a subject from the catalog, creates student groups within that course, and sets up the activities students will complete.

**Why this priority**: The course is the fundamental container for all academic work. Nothing else — submissions, analysis, grades — can exist without it. This flow must work end-to-end before any other M3 story is testable.

**Independent Test**: A docente logs in, creates a course linked to an existing subject, adds two groups, and creates one activity. The course appears in their dashboard and no other docente sees it.

**Acceptance Scenarios**:

1. **Given** an authenticated docente, **When** they create a course providing subject, semester label, and section, **Then** the course appears in their course list and is invisible to all other docentes.
2. **Given** an existing course owned by the docente, **When** the docente adds a student group with a name and optional capacity, **Then** the group is listed under that course.
3. **Given** an existing course, **When** the docente creates an activity specifying type, deadline, and instructions, **Then** the activity is visible to enrolled students in that course.
4. **Given** a docente with no courses, **When** they request their course list, **Then** they receive an empty list — not another docente's data.

---

### User Story 2 — Student Course Access and Group Assignment (Priority: P2)

A student logs in with their institutional Google account (`@unifranz.edu.bo`). The system knows their degree program and current semester from their Google identity and automatically makes the matching courses visible to them. The docente then organizes the pre-populated course roster into grupos for activities.

**Why this priority**: Course access and group organization are required before students can submit work. Role isolation (Constitution Principle IV) depends entirely on correct course scoping — this is the gate that controls what each student sees. The automatic enrollment model means no manual data entry and no risk of a docente enrolling the wrong student.

**Independent Test**: A student with carrera = "Ingeniería de Sistemas" and semester = 4 logs in via their Google account. The system automatically shows them all active cursos for that carrera and semester. The docente then creates two grupos and assigns students from the roster. The student can see their group and the course's activities.

**Acceptance Scenarios**:

1. **Given** a student whose institutional Google account carries carrera and semester information, **When** they log in for the first time, **Then** their account is created and they are automatically associated with all active cursos matching their carrera and semester.
2. **Given** a student associated with a curso, **When** they request their course list, **Then** only cursos matching their carrera and semester appear.
3. **Given** a student not yet associated with any curso, **When** they request their course list, **Then** they receive an empty list.
4. **Given** a student associated with curso A, **When** they attempt to access an activity in curso B (different carrera or semester), **Then** the request is denied.
5. **Given** a pre-populated course roster, **When** the docente creates a grupo and assigns students from that roster to it, **Then** those students appear in the grupo and the docente can use the grupo for activity organization.

---

### User Story 3 — Co-Teaching Grant for a Specific Activity (Priority: P3)

A docente grants a co_docente user access to a single activity. The co_docente can view that activity's details but cannot access the course, other activities, or the student roster.

**Why this priority**: Co-teaching scoping is a direct constitution requirement (Principle IV). The co_docente role has no meaningful boundary without per-activity grants, making every other co_docente operation undefined.

**Independent Test**: Grant a co_docente access to activity X. Log in as co_docente and confirm they see activity X. Attempt to access the course list, a different activity, or the student roster and receive denials.

**Acceptance Scenarios**:

1. **Given** an activity, **When** the docente grants co-teaching access to a user with role co_docente, **Then** that user can view the activity's details.
2. **Given** a co_docente with a grant for activity X in course C, **When** they request any other activity in course C, **Then** the request is denied.
3. **Given** a co_docente with a grant for activity X, **When** they request the course's student roster, **Then** the request is denied.
4. **Given** an active co-teaching grant, **When** the docente revokes it, **Then** the co_docente can no longer access that activity.

---

### User Story 4 — Admin Manages the Subject Catalog (Priority: P4)

An admin creates and maintains the catalog of degree programs (carreras) and subjects (materias) that docentes draw from when creating courses. Non-admin users cannot modify the catalog.

**Why this priority**: The catalog is a prerequisite for course creation but changes infrequently and can be bootstrapped with a seed. Correct operational flows (US1–US3) deliver more immediate value.

**Independent Test**: Admin creates a carrera and adds a materia. A docente then selects that materia when creating a new course. A non-admin attempt to create a materia is denied.

**Acceptance Scenarios**:

1. **Given** an authenticated admin, **When** they create a carrera with name and code, **Then** the carrera appears in the catalog and docentes can reference it.
2. **Given** an existing carrera, **When** the admin adds a materia with code, name, and semester number, **Then** docentes can select it when creating a course.
3. **Given** a non-admin user (docente, estudiante, co_docente), **When** they attempt to create or delete a materia, **Then** the request is denied.
4. **Given** an existing materia assigned to one or more courses, **When** the admin deactivates it, **Then** existing courses are unaffected but new courses cannot reference the deactivated materia.

---

### Edge Cases

- What happens if a docente creates two courses for the same subject, semester, and section? (System should reject as a duplicate.)
- What happens if a student is added to a second group within the same course? (Single-group enrollment per course — the system should reject or replace.)
- What happens if an activity deadline is set in the past? (Allow it — the docente may be recording a past event; a warning is acceptable.)
- What happens if a docente tries to hard-delete a course that already has student submissions? (System must block deletion and return an error; deactivation is the only permitted path.)
- What if a co_docente's account role is later changed to estudiante while an active grant exists? (The grant becomes unreachable; access should be denied based on current role.)
- What happens if the docente who owns a course is deactivated? (Courses remain; an admin must reassign or archive them.)
- What happens if the docente edits an activity's deadline or instructions after submissions exist? (Allowed — the change is recorded; the activity displays both the original posted date and the last-edited date.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A docente MUST be able to create, read, update, and deactivate courses referencing a subject from the catalog.
- **FR-002**: A docente MUST be able to create, read, update, and delete student groups within their own courses.
- **FR-003**: A docente MUST be able to create, read, update, and delete activities within their own courses, specifying type, deadline, and instructions. Once at least one student submission exists for an activity, only the deadline and instructions MAY be modified; the title and type are locked and deletion is blocked.
- **FR-004**: Activity type MUST be one of: `práctica`, `tarea`, `informe`, `examen`, `defensa_oral`.
- **FR-005**: Activities of type `examen` or `defensa_oral` MUST be automatically marked as trace-enabled; all other types MUST NOT be trace-enabled.
- **FR-006**: The system MUST automatically associate a student with all active cursos that match their degree program (carrera) and academic semester, using information provided by their institutional Google identity at login. No manual enrollment action by the docente is required.
- **FR-006b**: A docente MUST be able to create grupos and assign students from the pre-populated course roster to those grupos. Grupos are used for activity organization (e.g., project teams, lab sections) and do not affect which courses the student can see.
- **FR-007**: A docente MUST be able to grant and revoke co-teaching access to a user with role co_docente for a specific activity.
- **FR-008**: A docente MUST only access courses, groups, activities, and enrollment records belonging to their own account.
- **FR-009**: A co_docente MUST only access the specific activities for which they hold an active grant — the course, other activities, and the student roster MUST be inaccessible.
- **FR-010**: An estudiante MUST only see the courses in which they are enrolled, the groups they belong to, and the activities within those courses.
- **FR-011**: An estudiante MUST NOT be able to view AI analysis reports or alert status related to their own submissions.
- **FR-012**: An admin MUST be able to create, read, update, and deactivate carreras and materias.
- **FR-013**: A course MUST reference a subject from the catalog; free-text subject names are not permitted.
- **FR-014**: The system MUST prevent hard deletion of courses that have associated student submissions; deactivation MUST be used instead.
- **FR-015**: The system MUST prevent assignment of the same student to more than one grupo within the same curso.
- **FR-019**: Every inscripción MUST record a creation timestamp and, if the association is deactivated (e.g., student's semester advances and they leave a curso), a revocation timestamp and reason. These timestamps MUST be accessible to admins for dispute investigation.
- **FR-020**: Every co-docente grant MUST record a creation timestamp and, upon revocation, a revocation timestamp and the identity of who revoked it. These records MUST be accessible to the owning docente and to admins.
- **FR-016**: Every activity MUST record a creation timestamp and a last-modified timestamp. Both MUST be returned when the docente views activity details, allowing the docente to see when an activity was originally posted and when it was last edited.
- **FR-017**: The system MUST retrieve a student's carrera and current semester from their institutional Google account (`@unifranz.edu.bo`) at first login, and use these values to determine their course memberships.
- **FR-018**: A student MUST be able to authenticate using their institutional Google account. The system MUST create their internal account automatically on first login if one does not already exist, using the carrera and semester data from their Google identity.

### Key Entities

- **Carrera**: Degree program at the university. Key attributes: unique code, name, active flag.
- **Materia**: Academic subject belonging to a carrera. Key attributes: unique code, name, semester number, carrera reference, active flag.
- **Curso**: A specific offering of a materia by a docente in a given cycle. Key attributes: materia reference, docente reference, semester label, section, active flag.
- **Grupo**: A named subgroup of students within a curso. Key attributes: name, optional capacity, curso reference.
- **Actividad**: A task or evaluation event within a curso. Key attributes: title, type, deadline, instructions, trace-enabled flag (derived from type), curso reference, creation timestamp, last-modified timestamp.
- **Inscripción**: A system-generated record linking a student to a curso, created automatically when the student's Google identity is matched to active cursos for their carrera and semester. Key attributes: student reference, curso reference, carrera at time of enrollment, semester at time of enrollment, created timestamp, revoked timestamp (nullable), revocation reason (nullable), active flag.
- **Asignación de Grupo**: A docente-managed record placing an enrolled student into a specific grupo within a curso. Key attributes: student reference, grupo reference, assigned-by (docente reference), assigned date.
- **Co-docente Grant**: An explicit permission record linking a co_docente to a specific actividad. Key attributes: co_docente reference, actividad reference, granted-by (docente reference), granted timestamp, revoked timestamp (nullable), revoked-by (nullable), active flag.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A docente can complete the full course setup flow (create course → add group → enroll student → create activity) in under 5 minutes.
- **SC-002**: No response to any docente request contains data belonging to a different docente's courses, groups, activities, or students.
- **SC-003**: A co_docente with a grant for activity X receives an authorization denial for every other resource in the same course (verified across all endpoints).
- **SC-004**: An enrolled student's course list contains exactly the courses they are enrolled in — no more, no less.
- **SC-005**: All role-based data isolation violations detected during testing are zero before the milestone is marked complete.
- **SC-006**: Admin catalog operations (create carrera, create materia) complete and are immediately available for course creation without a restart or refresh.

## Clarifications

### Session 2026-05-19

- Q: If a student has already submitted work for an activity, can the docente still edit or delete it? → A: Partial edit allowed (deadline and instructions only); title and type are locked; deletion is blocked once submissions exist. Every activity records creation and last-modified timestamps visible to the docente.
- Q: Should the system support bulk student enrollment by the docente, or is enrollment automatic? → A: Enrollment is fully automatic — the system reads carrera and semester from each student's institutional Google account (`@unifranz.edu.bo`) and associates them with matching cursos on login. Docentes do not enroll students; they organize the pre-populated roster into grupos for activity purposes. The Google integration design is a prerequisite before M3 implementation begins.
- Q: Should the system keep a history of enrollment and co-teaching grant changes for audit purposes? → A: Simple audit trail — creation and revocation timestamps are recorded on inscripciones and co-docente grants. No full change-history log (deferred to M5+ event log).

## Design Prerequisite — Google Identity Integration

**This section must be resolved before `/speckit-plan` is run.**

The enrollment model depends on reading carrera and semester data from each student's institutional Google account (`@unifranz.edu.bo`). Before planning tasks, the following design decisions must be made:

- **Where does Google store carrera and semester?** Options include Google Workspace custom attributes (set by the institution's admin), Google Directory API fields, or custom claims in the Google OAuth token. The Unifranz IT administrator must confirm which field carries this data.
- **When is enrollment triggered?** On first login only, or re-evaluated on every login (to handle semester progression). Recommendation: re-evaluate on every login and update the student's active cursos if their semester has advanced.
- **What happens if Google data is missing or malformed?** The student should be able to log in but see an empty course list, with an error message prompting them to contact the institution.
- **Docente and admin accounts**: These use email/password authentication (M2). Google OAuth is for students only.

Once these decisions are made, they should be recorded in the plan's Technical Context and inform the M2 extension tasks (adding Google OAuth as a second auth provider).

## Assumptions

- Authentication and role middleware from M2 are operational for docente and admin accounts (email/password + JWT). All M3 endpoints enforce role-based access via the existing middleware.
- Students authenticate via Google OAuth using their `@unifranz.edu.bo` accounts; this is an extension to the M2 auth system that must be designed before M3 implementation begins.
- Google Workspace custom attributes (or an equivalent field confirmed by the Unifranz IT admin) carry each student's `carrera_code` and `semester_number` and are accessible via the Google OAuth token or Directory API at login time.
- At least one admin user exists in the system before M3 setup begins (seeded during M2).
- Co_docente accounts are created by an admin (existing M2 mechanism); co_docente authentication uses email/password, not Google OAuth.
- A student may belong to at most one grupo per curso for group-assignment purposes; this does not affect their curso membership.
- Co-teaching grants are scoped per activity, not per curso; curso-level co-teaching access is out of scope for M3.
- Soft deletion (deactivation) is the default for cursos with existing submissions; hard delete is permitted only when no submissions exist.
- The trace-enabled flag on an activity is derived automatically from its type and is not manually configurable by the docente.
