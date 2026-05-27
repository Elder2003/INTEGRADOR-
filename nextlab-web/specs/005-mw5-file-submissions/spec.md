# Feature Specification: MW5 — File Submissions

**Feature Branch**: `014-mw5-file-submissions`

**Created**: 2026-05-26

**Status**: Draft

**Input**: User description: "Lets do the mw5"

## Clarifications

### Session 2026-05-26

- Q: When a student returns to an activity they already submitted, what controls are shown? → A: Two separate actions — a "Replace" button (opens file picker, replaces via PUT) and a "Delete" button (with confirm modal, deletes via DELETE) — both visible when the submission is not locked.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Submits a File (Priority: P1)

A student navigates to an activity detail page, sees whether they already have a submission, and uploads a file for the first time. They must confirm an integrity declaration before the file is sent. Once uploaded, the page shows their current submission (filename, size, submitted-at, lock status, and whether it was late). If the deadline has passed and late submissions are disabled, the upload is rejected with a clear message.

**Why this priority**: File upload is the core student action of the entire submission workflow — nothing else is meaningful without it.

**Independent Test**: Upload a file to a `tarea`-type activity as a student → submission appears on the page with filename, size, and submitted-at date.

**Acceptance Scenarios**:

1. **Given** a student is on an activity detail page with no submission, **When** they check the integrity declaration and upload a valid file (≤ 50 MB), **Then** the page shows the uploaded file's metadata and a success indicator.
2. **Given** a student already has a submission that is not locked, **When** they click "Replace" and select a new file, **Then** the new file metadata is shown and the old file is gone.
3. **Given** a student already has a submission that is not locked, **When** they click "Delete" and confirm the modal, **Then** the submission is removed and the upload form is shown again.
4. **Given** a student's edit window has expired (submission is locked), **When** they view the activity page, **Then** the "Replace" and "Delete" buttons are hidden and the submission is shown as read-only.
4. **Given** the activity deadline has passed and `late_submission_allowed = false`, **When** a student tries to upload, **Then** a "late submissions are not allowed" message is shown.
5. **Given** an activity of type `examen` or `defensa_oral` (invalid for file submission), **When** a student tries to upload, **Then** an "invalid activity type" error is shown.
6. **Given** a student selects a file larger than 50 MB, **When** they attempt to upload, **Then** a "file too large" error is shown before the request is sent.
7. **Given** a student has no submission, **When** they view the activity page, **Then** only the upload form (with integrity declaration) is shown — no Replace or Delete buttons.

---

### User Story 2 — Docente Reviews Submissions List (Priority: P2)

A docente opens an activity and sees a paginated table of all student submissions — each row shows the student name, email, filename, file size, submission date, whether it was late, and whether it is locked. The docente can click any row to view submission details and download the file.

**Why this priority**: Docentes need to review submissions to grade — this is the primary read path for the submission data.

**Independent Test**: Navigate to `/docente/cursos/:cid/actividades/:aid/submissions` as a docente → table renders with all student submissions from the API.

**Acceptance Scenarios**:

1. **Given** a docente is on the activity submissions page, **When** the page loads, **Then** a table shows all student submissions with name, filename, size, submitted-at, late badge, and locked status.
2. **Given** no submissions exist yet, **When** the page loads, **Then** an empty state message is shown.
3. **Given** submissions are listed, **When** a docente clicks a row, **Then** they navigate to the submission detail page for that student.
4. **Given** a co-docente has been granted access to the activity, **When** they navigate to the same submissions page, **Then** they see the same list (read-only, no grade actions yet in MW5).

---

### User Story 3 — Docente Downloads a Submission (Priority: P3)

A docente opens the submission detail page for a specific student and sees the submission metadata (filename, size, submitted-at, late/locked status). They click "Download" and the file is saved to their device.

**Why this priority**: Download is the concrete deliverable for the docente — required to actually review submitted work.

**Independent Test**: Open a submission detail page as a docente and click download → the file is received with the correct filename and MIME type.

**Acceptance Scenarios**:

1. **Given** a docente is on the submission detail page, **When** they click "Download", **Then** the file downloads with the original filename and correct content type.
2. **Given** a co-docente opens the same page, **When** they click "Download", **Then** the file downloads successfully (co-docente has the same download permission).
3. **Given** the submission detail page is opened, **When** it renders, **Then** it shows: student name, filename, file size, submitted-at, whether it was late, whether it is locked, and the edit window expiry if present.

---

### Edge Cases

- Student uploads a file when storage is temporarily unavailable → "storage unavailable" error shown, no partial upload retained.
- Student navigates directly to the activity URL after the edit window expires → submission shown as read-only; "Replace" and "Delete" buttons are not rendered.
- Docente navigates to submissions page for an activity with no tipo that accepts file uploads → page renders empty state (server controls which tipos are valid).
- A `git_analysis` or `investigacion` tipo activity is displayed — file upload is not shown (handled by future milestones, not MW5).
- Config-changes audit log visible to docente on the submissions page (endpoint `GET /cursos/:cid/actividades/:aid/config-changes`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Student MUST be able to upload a file (multipart `file` field) to `POST /actividades/:aid/submission` from the activity detail page.
- **FR-002**: Student MUST confirm an integrity declaration checkbox before the upload button becomes active.
- **FR-003**: After a successful upload, the page MUST display the submission's filename, file size, submitted-at date, late status, locked status, and edit window expiry.
- **FR-004**: When a submission exists and is not locked, a "Replace" button MUST be shown; clicking it opens a file picker and sends `PUT /actividades/:aid/submission`.
- **FR-005**: When a submission exists and is not locked, a "Delete" button MUST be shown; clicking it opens a confirm modal and, on confirmation, sends `DELETE /actividades/:aid/submission`.
- **FR-005a**: When a submission is locked (`is_locked = true`), both the "Replace" and "Delete" buttons MUST be hidden; the submission is displayed read-only.
- **FR-006**: System MUST block upload, replace, and delete with clear error messages when the submission is locked, late submissions are disabled, the activity type is invalid, or the file exceeds 50 MB.
- **FR-007**: Docente and co-docente MUST be able to view a submissions list at `GET /cursos/:cid/actividades/:aid/submissions` showing all student submissions.
- **FR-008**: Each submission list row MUST show: student name, student email, filename, file size, submitted-at, late badge, locked status.
- **FR-009**: Docente and co-docente MUST be able to download any submission via `GET /submissions/:sid/download`.
- **FR-010**: A submission detail page MUST show all submission metadata and a download button.
- **FR-011**: Docente MUST be able to view the activity config-change audit log via `GET /cursos/:cid/actividades/:aid/config-changes`.
- **FR-012**: The student's current submission status MUST be fetched on page load via `GET /actividades/:aid/submission` (404 = no submission yet).
- **FR-013**: File upload progress MUST be indicated to the user during upload.

### Key Entities

- **Submission**: One file per student per activity. Fields: id, actividad_id, original_filename, file_size_bytes, mime_type, submitted_at, is_late, edit_window_expires_at, is_locked, last_replaced_at.
- **SubmissionListItem**: Docente-facing view — adds student name and email to the submission fields.
- **ConfigChangeResponse**: Audit record of who changed which actividad field, previous/new value, and timestamp.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Student can upload a file and see confirmation within 5 seconds for files under 10 MB.
- **SC-002**: All student-side error states (locked, late, invalid type, too large, storage unavailable) are shown with distinct, user-readable messages — 0 silent failures.
- **SC-003**: Docente submission list loads all entries for an activity in under 3 seconds.
- **SC-004**: Docente can download a submission and receive the file within 5 seconds for files under 10 MB.
- **SC-005**: Integrity declaration checkbox is required — upload button is disabled until the checkbox is checked.
- **SC-006**: React Doctor scan after MW5 returns 0 ERROR-severity findings.

## Assumptions

- File size limit (50 MB) and allowed MIME types are enforced by the backend; the frontend only validates size client-side before sending.
- The integrity declaration is a UI-only gate (checkbox must be checked) — it is not sent to the API as a separate field.
- `examen` and `defensa_oral` activity types do not accept file submissions; the student activity detail page will not show the upload form for those types.
- `git_analysis` and `investigacion` types are out of scope for MW5 (handled in MW7+).
- Edit window expiry logic is enforced by the backend; the frontend reads `is_locked` from the API response and shows/hides controls accordingly.
- Co-docente has the same read/download permissions as docente for submissions; grading is out of scope for MW5.
- The student activity detail page (`/student/cursos/:id/actividades/:aid`) from MW4 will be extended in MW5 — not replaced.
