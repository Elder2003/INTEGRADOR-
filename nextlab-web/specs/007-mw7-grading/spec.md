# Feature Specification: MW7 — Grading

**Feature Branch**: `007-mw7-grading`

**Created**: 2026-05-26

**Status**: Draft

**Input**: User description: "lets do mw7"

## Clarifications

### Session 2026-05-26

- Q: Does the exam form present individual questions from a backend list, or a single free-form text area? → A: Single free-form text area — student writes all answers in one field; backend handles structure.
- Q: After the docente saves a grade, when does the student see it? → A: Immediately — student can see the grade as soon as the docente saves it; no draft/publish step.
- Q: Does the GradePanel include an explicit pass/fail verdict field in addition to the score? → A: Score + feedback only — no separate verdict; pass/fail is derived by the backend from the score threshold.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Docente Grades a Submission (Priority: P1)

A docente navigates to a student's submission from the submissions list, reviews the file, and uses the grade panel to assign a numeric score (0–100) and written feedback. The grade is submitted to the backend and immediately reflected in the UI without page reload.

**Why this priority**: Grading is the core deliverable of MW7. Without it, students cannot see results and no other grading-related feature makes sense. The constitution defines `grade_decisions` as a first-class entity editable only through the GradePanel.

**Independent Test**: Sign in as docente → open a submission detail page → fill in GradePanel with score 85 and feedback text → submit → grade badge appears on submission. Then sign in as student → navigate to the same activity → see grade 85 and feedback displayed.

**Acceptance Scenarios**:

1. **Given** a docente is on a submission detail page with no existing grade, **When** they enter a score (0–100) and feedback and click "Calificar", **Then** the grade is saved and the GradePanel shows the recorded score and feedback in read-only mode.
2. **Given** a grade already exists for a submission, **When** the docente revisits the submission detail page, **Then** the existing score and feedback are displayed in the GradePanel.
3. **Given** the docente submits a grade with a score out of the 0–100 range, **When** the form is submitted, **Then** the API returns an error and an inline error message is shown — the grade is not saved.
4. **Given** a co-docente views a graded submission, **When** they open the submission detail page, **Then** they see the grade in read-only mode with no edit controls.

---

### User Story 2 — Student Views Their Grades (Priority: P2)

A student navigates to an activity detail page and sees their grade (score + feedback) immediately after the docente saves it — no publish step is required. Before grading, a "Pendiente de calificación" indicator is shown in place of the grade.

**Why this priority**: The student's ability to see their own grade closes the educational feedback loop. Without this, grading is invisible to the person it affects most.

**Independent Test**: Sign in as student → navigate to an activity where the submission has been graded → see score and feedback displayed below the submission card. Navigate to an ungraded submission → see "Pendiente de calificación".

**Acceptance Scenarios**:

1. **Given** a student's submission has been graded, **When** they navigate to the activity detail page, **Then** they see the numeric score (e.g., "85 / 100") and the docente's feedback text displayed below the submission card.
2. **Given** a student's submission has NOT been graded yet, **When** they view the activity detail page, **Then** they see "Pendiente de calificación" in the grade area — no score or feedback shown.
3. **Given** a student has no submission for a traced-enabled activity, **When** they view the activity, **Then** the grade section is not shown.

---

### User Story 3 — Exam Session Form for Student (Priority: P2)

For activities of type `examen`, the student sees an in-browser answer form rather than (or in addition to) the file upload panel. The form allows the student to write text answers. Once submitted, the answers are recorded and the student cannot re-submit.

**Why this priority**: Exam activities have a distinct input method compared to file-based submissions. Supporting the exam flow is essential for completeness of the student experience across all activity types.

**Independent Test**: Sign in as student → navigate to an `examen`-type activity with tracing active → start the monitoring session → see the answer form → type responses → submit → confirmation shown. Verify student cannot re-submit (form becomes read-only).

**Acceptance Scenarios**:

1. **Given** a student opens an `examen`-type activity with `trace_enabled = true`, **When** they have an active monitoring session, **Then** they see the exam answer form (text area) instead of the file upload panel.
2. **Given** the student completes the exam form and submits, **When** the submission succeeds, **Then** the form transitions to a read-only view showing their answers with the submission timestamp.
3. **Given** an `examen`-type activity where the deadline has passed, **When** the student views it, **Then** the form shows "Examen cerrado" and the answers (if any) in read-only mode.
4. **Given** an `examen`-type activity with `trace_enabled = false`, **When** the student views it, **Then** the answer form is accessible directly without requiring a monitoring session.

---

### User Story 4 — Investigación Stage Tracking (Priority: P3)

For activities of type `informe`, the docente can view and update the investigación stage of each student's submission. Stages progress linearly: `propuesta → desarrollo → revisión → aprobado`. The docente selects the current stage from a dropdown and saves. Students see their current stage in read-only mode.

**Why this priority**: Research (informe) activities are long-running and require incremental feedback. Stage tracking gives the docente a lightweight tool to communicate progress without a full grade.

**Independent Test**: Sign in as docente → open a submission detail for an `informe`-type activity → see the stage selector → advance stage from `propuesta` to `desarrollo` → confirm the update. Sign in as student → navigate to the activity → see current stage displayed.

**Acceptance Scenarios**:

1. **Given** an `informe`-type submission, **When** the docente opens the submission detail, **Then** they see a stage selector showing the current stage (defaulting to `propuesta`).
2. **Given** the docente selects a new stage and clicks "Actualizar etapa", **When** the save succeeds, **Then** the stage label updates immediately on the page.
3. **Given** a student views their `informe` activity, **When** a stage has been set, **Then** they see the current stage label ("Etapa: Desarrollo") in read-only mode.

---

### User Story 5 — Defensa Oral Scheduling (Priority: P3)

For activities of type `defensa_oral`, the docente assigns a specific date and time slot to each student (or student group). Students view their assigned slot from the activity detail page.

**Why this priority**: Oral defense activities require coordination between docente and student. Without scheduling, the `defensa_oral` activity type cannot be completed.

**Independent Test**: Sign in as docente → open a `defensa_oral` activity → navigate to the scheduling panel → assign a date/time to a student → confirm. Sign in as student → open the same activity → see their assigned slot displayed.

**Acceptance Scenarios**:

1. **Given** a docente opens a `defensa_oral` activity detail page, **When** they access the scheduling panel, **Then** they see a list of enrolled students with date/time assignment fields.
2. **Given** the docente enters a valid date/time for a student and saves, **When** the save succeeds, **Then** the slot is recorded and shown in the scheduling list.
3. **Given** a student navigates to a `defensa_oral` activity where they have been assigned a slot, **When** they view the activity detail, **Then** they see their scheduled date and time.
4. **Given** no slot has been assigned to a student, **When** they view the activity, **Then** they see "Defensa aún no programada" in the scheduling area.

---

### Edge Cases

- What happens when a docente tries to submit a grade for a locked submission? The grade form is still accessible — locking controls replacement but not grading.
- What happens if the student's exam submission fails (network error)? An inline error is shown; the form remains editable and the student can retry.
- What happens if a docente edits a grade that already exists? The new values overwrite the previous ones — the API accepts a PATCH and returns the updated grade.
- What if the exam deadline passes while the student is mid-form? The student finishes and submits — the backend marks it as late using `is_late`; the frontend displays the "Entrega tardía" badge.
- What if a defensa oral slot conflicts with another student's assignment? Conflict detection is the backend's responsibility; the frontend displays the error returned by the API.

## Requirements *(mandatory)*

### Functional Requirements

**Grading (all activity types)**:

- **FR-001**: Docente MUST be able to enter a numeric score (integer 0–100) and a text feedback comment for any student submission via a GradePanel on the submission detail page. No separate pass/fail verdict field is shown — pass/fail status is derived by the backend from the score.
- **FR-002**: Docente MUST be able to update (overwrite) an existing grade for a submission.
- **FR-003**: GradePanel MUST be read-only for co-docente — co-docente sees the grade and feedback but cannot edit or submit new grades.
- **FR-004**: GradePanel MUST NOT be shown to students — students see only their resulting grade display, not the docente's input form.
- **FR-005**: System MUST display a "Calificando..." loading state while the grade submission is in-flight and disable the submit button to prevent duplicate submissions.
- **FR-006**: System MUST show an inline success indicator (e.g., "Calificación guardada") for at least 3 seconds after a successful grade save.

**Student grade display**:

- **FR-007**: Student activity detail page MUST display the grade (score + feedback) for their submission once it has been graded.
- **FR-008**: Student MUST see "Pendiente de calificación" when their submission exists but has no grade yet.
- **FR-009**: Grade display MUST be read-only for the student — no editing controls.

**Exam session form**:

- **FR-010**: For `examen`-type activities, the student's activity detail page MUST render a single multi-line free-form text area instead of the file upload panel. The student writes all answers into one field; the backend handles any structure or validation. No per-question rendering is required on the frontend.
- **FR-011**: Exam form submission MUST be locked by the monitoring session gate when `trace_enabled = true` — the same gating logic as file uploads.
- **FR-012**: After the student submits the exam form, the answer area MUST become read-only — re-submission is not possible.
- **FR-013**: The exam answer form MUST show the submission timestamp after a successful submit.

**Investigación stages (informe activities)**:

- **FR-014**: For `informe`-type submissions, the docente's submission detail page MUST display a stage selector with options: `propuesta`, `desarrollo`, `revisión`, `aprobado`.
- **FR-015**: Docente MUST be able to save a new stage value; the stage label MUST update on the page without a reload.
- **FR-016**: Student activity detail MUST display the current stage in read-only mode.

**Defensa oral scheduling (defensa_oral activities)**:

- **FR-017**: The docente activity detail page for `defensa_oral` activities MUST include a scheduling panel listing all enrolled students with date/time assignment inputs.
- **FR-018**: Docente MUST be able to assign or update a date/time slot per student — one slot per student per activity.
- **FR-019**: Student activity detail page for `defensa_oral` activities MUST show their assigned slot or "Defensa aún no programada" if no slot exists.

### Key Entities

- **GradeDecision**: Links a submission to a numeric score (0–100) and text feedback. No separate verdict field — pass/fail is computed server-side. Belongs to a submission; one grade per submission.
- **InvestigaciónStage**: Current stage label (`propuesta | desarrollo | revisión | aprobado`) associated with an informe submission. Updated by docente.
- **ExamSubmission**: Student's in-browser text answer for an examen activity — a single free-form text block submitted as one unit. Treated as a submission; one per student per activity. Locked (read-only) after first submit.
- **DefensaSlot**: Date/time assignment for a student in a defensa_oral activity. Belongs to an actividad + user pair. Created/updated by docente; read-only for student.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Docente can open a submission, enter a grade, and save it in under 30 seconds on a standard connection.
- **SC-002**: Student sees their grade within 1 second of navigating to the activity detail page after grading has been recorded.
- **SC-003**: Grades are immediately visible to students after the docente saves — no publish step, no manual refresh required. 100% of graded submissions display the score and feedback to the student on the next page load.
- **SC-004**: Exam form submission completes in under 5 seconds from button click to confirmation shown.
- **SC-005**: Investigación stage update is visible to the docente immediately (under 2 seconds) after saving.
- **SC-006**: All grade-related UI controls (GradePanel edit fields) are absent from student role views — verified by role-based rendering tests.

## Assumptions

- The backend exposes `grade_decisions` as a resource linked to submission ID — the frontend retrieves it alongside the submission or as a separate GET.
- Exam activities use the same `POST /actividades/:aid/submissions` endpoint as file uploads but accept a `text/plain` or `application/json` body with the student's answers instead of a multipart file.
- Investigación stages are stored on the submission record itself (e.g., `stage` field) or as a separate sub-resource — the frontend does not infer stage ordering; the API returns valid stage values.
- Defensa oral scheduling is a separate resource (`/actividades/:aid/slots`) managed by the docente; the student reads their slot from `GET /actividades/:aid/slots/me`.
- Co-docente has read access to grades and stages for activities they are granted access to — the same ProtectedRoute + co-docente role guard that applies to submissions also applies here.
- Grade overwrite (updating an existing grade) is permitted by the API via a PATCH request — the frontend always uses the same "save" button regardless of whether a grade already exists.
- `StudentActividadEntry` already includes `trace_enabled` — no additional fields are needed to detect exam vs. file activity on the student side (activity type `examen` drives the form switch).
- Defensa oral and investigación stage features are not visible for other activity types (practica, tarea) — the frontend renders these sections only when `actividad.tipo === 'defensa_oral'` or `actividad.tipo === 'informe'`.
