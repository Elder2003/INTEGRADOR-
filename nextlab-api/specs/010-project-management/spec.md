# Feature Specification: M10 — Project Management

**Feature Branch**: `010-project-management`

**Created**: 2026-05-22

**Status**: Draft

**Input**: User description: "lets do the milestone 10"

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Multi-Stage Research Report (Priority: P1)

A docente creates an **investigación** activity, which is a new activity type representing an academic research or investigation project. Instead of a single deadline, the docente defines a sequence of named stages — each with its own deadline and optional description (e.g., "Propuesta de tema", "Marco teórico", "Entrega final"). Students submit a file independently for each stage. The docente can view a progress overview per student showing which stages are submitted, late, or pending, and can assign a grade and note to each stage submission individually.

**Why this priority**: Multi-stage research report management is the structural foundation of M10 and delivers standalone value immediately — docentes can track semester-long investigations without waiting for the other features. Git analysis and oral defense are separate activity types and do not depend on this one.

**Independent Test**: Create an `investigacion` activity with three stages, enroll a student, submit to stage 1 on time and to stage 3 after its deadline. Verify the docente sees stage 1 as `submitted`, stage 2 as `pending`, and stage 3 as `submitted_late`. Assign a grade to stage 1. All verifiable without Git analysis or oral defense.

**Acceptance Scenarios**:

1. **Given** a docente creates an `investigacion` activity with three stages, **When** a student views their activities, **Then** they see all three stages listed with individual names, deadlines, and current status.
2. **Given** a student submits to stage 1 before its deadline and to stage 3 after its deadline, **When** the docente views the stage overview, **Then** stage 1 shows `submitted`, stage 2 shows `pending`, stage 3 shows `submitted_late` — and no submission is rejected (Principle VII).
3. **Given** stage submissions exist, **When** the docente opens a specific student's stage 2 submission, **Then** they can assign a numeric grade (0–100) and an optional note, saved independently from other stages.
4. **Given** a docente tries to delete a stage that already has at least one submission, **When** they send the delete request, **Then** the system returns an error and the stage is not deleted.
5. **Given** a docente publishes an `investigacion` activity, **When** the activity has zero stages, **Then** the publish request is rejected with a validation error.

---

### User Story 2 — Git Repository Analysis Activity (Priority: P2)

A docente creates a **git_analysis** activity — a new activity type that works like any regular homework assignment, except that the student's deliverable is a Git repository URL instead of a file upload. When a student submits their repo URL and the activity deadline passes (or the docente triggers analysis manually), the async worker clones the repository, extracts commit statistics, scores the commit history using heuristics, and then sends the structured result to the LLM for an advisory assessment — the same pipeline pattern used in M8. The docente reviews the resulting report (heuristic flags + LLM advisory text) and assigns a final grade. Students cannot see their own analysis report.

**Why this priority**: Git analysis is an independent activity type that reuses the existing detection and AI router infrastructure. It delivers meaningful academic integrity value for programming-oriented courses without requiring the stage scaffolding of US1.

**Independent Test**: Create a `git_analysis` activity. As a student, submit a public Git repository URL. Wait up to 3 minutes. As the docente, call the detection report endpoint for that submission's session and verify a report exists with: commit count, author list, heuristic flags, and LLM advisory text.

**Acceptance Scenarios**:

1. **Given** a student submits a valid public Git repository URL for a `git_analysis` activity, **When** the async worker processes the job, **Then** within 3 minutes a Git analysis record exists with: total commit count, list of distinct author names found in commits, date of first and last commit, and a consistency score (0–100).
2. **Given** a repository where more than 70% of commits were made in the final 24 hours before the activity deadline, **When** the worker runs heuristics, **Then** the report includes the flag `deadline_rush`.
3. **Given** a repository with commits attributed to more than one distinct author name, **When** heuristics run, **Then** the report includes the flag `multiple_authors` with the detected names listed.
4. **Given** the heuristic analysis completes, **When** the LLM processes the commit data, **Then** an advisory text is stored summarising the findings in natural language — visible only to the docente.
5. **Given** the submitted Git repository URL is unreachable or the clone fails, **When** the worker attempts analysis, **Then** a failed analysis record is stored with `fail_reason = clone_error`; the submission itself is unaffected and the docente sees the failure reason in the report.
6. **Given** a docente views the analysis report, **When** they decide to assign a grade, **Then** they can record a final grade (0–100) with an optional override reason if the grade differs significantly from the LLM advisory range.
7. **Given** a student JWT is used to request a git analysis detection report, **Then** a `403 Forbidden` is returned (Principle IV).

---

### User Story 3 — Oral Defense Scheduling and Grading (Priority: P3)

For `defensa_oral` activities, the docente sets a **scheduled date and time** for each student's defense — this replaces the generic deadline with a specific appointment. No file upload or session tracing is required. The defense is a human-to-human interaction; the system's role is to record when it is scheduled and to allow the docente to enter the final grade after the defense takes place. The student can view when their defense is scheduled.

**Why this priority**: Oral defense scheduling is the lightest of the three features — it extends an existing activity type with minimal new behaviour. It is also the most human-driven: the system records outcomes rather than capturing events.

**Independent Test**: Create a `defensa_oral` activity. Assign a scheduled date to a specific student. Verify the student can see their scheduled date. After the scheduled time passes, the docente enters a grade for that student's defense. Verify the grade is stored and retrievable.

**Acceptance Scenarios**:

1. **Given** a `defensa_oral` activity exists, **When** the docente assigns a defense date and time to a student, **Then** the student can see their scheduled defense appointment in their activity view.
2. **Given** a defense appointment has been scheduled, **When** the docente records a grade (0–100) for that student's oral defense, **Then** the grade is stored and visible in the docente's submission review.
3. **Given** a student has no scheduled defense appointment yet, **When** they view the `defensa_oral` activity, **Then** they see the activity as pending with no date shown.
4. **Given** a docente tries to record a grade before the scheduled defense time has passed, **When** they submit the grade, **Then** the system accepts it (the docente is authoritative — Principle III).

---

### Edge Cases

- What if a student submits to a later stage without submitting to an earlier one? → Accepted; stages are independent. The skipped stage remains `pending` and is visible to the docente.
- What if the Git repository is empty (no commits)? → Analysis stored with `commit_count = 0`, `consistency_score = null`, no flags raised; LLM advisory notes the empty history.
- What if the LLM call fails during Git analysis? → Heuristic result is stored independently; LLM status stored as `failed`; docente sees the heuristic flags even without the advisory text (same as M8 behaviour).
- What if the same student submits a new Git URL to replace a prior submission? → A new analysis job is enqueued for the updated URL; the old analysis record is superseded by the new one.
- What if a docente schedules two oral defense appointments for the same student in the same activity? → Only the latest scheduled appointment is active; the prior one is replaced.
- What if the `investigacion` activity has no stages and a student tries to submit? → The activity is not yet published (FR-007 blocks publication without stages), so the student cannot see it.

## Requirements *(mandatory)*

### Functional Requirements

**Multi-Stage Research Report (US1)**

- **FR-001**: System MUST support a new `investigacion` activity type in which the docente defines one or more named stages, each with its own deadline and optional description.
- **FR-002**: Students MUST be able to submit a file to each stage independently, regardless of the status of other stages.
- **FR-003**: System MUST mark a stage submission as `submitted_late` when received after that stage's deadline.
- **FR-004**: Docente MUST be able to view a per-student stage progress overview showing each stage status: `pending`, `submitted`, or `submitted_late`.
- **FR-005**: Docente MUST be able to assign a numeric grade (0–100) and optional note to each stage submission independently.
- **FR-006**: System MUST reject deletion of a stage when at least one submission exists for it.
- **FR-007**: System MUST reject publication of an `investigacion` activity that has zero stages defined.

**Git Repository Analysis (US2)**

- **FR-008**: System MUST support a new `git_analysis` activity type where the student's submission is a Git repository URL rather than a file upload.
- **FR-009**: When a student submits a Git URL, the async worker MUST clone the repository and extract: total commit count, list of distinct commit author names, date of first commit, date of last commit, and the percentage of commits made within 24 hours of the activity deadline (Principle II).
- **FR-010**: Worker MUST compute a `consistency_score` (0–100) and flag `deadline_rush` when more than 70% of commits fall within 24 hours of the deadline, and flag `multiple_authors` when more than one distinct author name is detected.
- **FR-011**: After heuristic extraction, the worker MUST send the commit statistics to the LLM for an advisory assessment — same pipeline pattern as M8.
- **FR-012**: System MUST store a failed analysis record with `fail_reason = clone_error` when the repository is unreachable, without affecting the submission record.
- **FR-013**: Git analysis report MUST be accessible to docentes via the existing detection report endpoint; students MUST NOT access it (Principle IV).
- **FR-014**: Docente MUST be able to assign a final grade after reviewing the Git analysis report, with an optional override reason when the grade differs from the LLM advisory range.

**Oral Defense (US3)**

- **FR-015**: Docente MUST be able to assign a specific scheduled date and time to a student's `defensa_oral` activity.
- **FR-016**: The scheduled defense date MUST be visible to the student in their activity list.
- **FR-017**: Docente MUST be able to record a final grade (0–100) for a student's oral defense at any time after the defense is scheduled.
- **FR-018**: System MUST NOT require a file submission or session trace for `defensa_oral` grading (the defense is a human interaction).

### Key Entities

- **InvestigacionStage**: A named checkpoint within an `investigacion` activity; has name, order, deadline, and optional description; belongs to one actividad.
- **StageSubmission**: A student's file upload for a specific `InvestigacionStage`; carries `is_late` flag and an optional docente-assigned grade and note; uniquely keyed by `(student_id, stage_id)`.
- **GitSubmission**: A student's Git repository URL submitted for a `git_analysis` activity; replaces file upload for this activity type.
- **GitAnalysis**: Async analysis record for a `GitSubmission`; stores commit count, author list, first/last commit dates, deadline-rush percentage, consistency score, heuristic flags, LLM advisory text, and status (`success` / `failed`).
- **DefenseAppointment**: Scheduled date/time for a specific student's `defensa_oral`; belongs to one actividad and one student; carries the docente-assigned grade once recorded.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Git analysis report (heuristic + LLM advisory) is available within 3 minutes of submission for 95% of cases with a reachable repository (Principle II SLA).
- **SC-002**: A docente can view the full stage-progress overview for a 30-student `investigacion` activity in a single response with no pagination.
- **SC-003**: `deadline_rush` and `multiple_authors` flags each have a false-positive rate below 5% on a test set of known-clean repositories.
- **SC-004**: Zero submissions are rejected due to Git clone failure, LLM failure, or any system error (Principle VII).
- **SC-005**: Docente can schedule, view, and grade all oral defenses for a 30-student activity without leaving the existing activity management interface.

## Assumptions

- `investigacion` and `git_analysis` are added to the existing `actividades` tipo enum; all prior tipos remain valid and unchanged.
- Git analysis uses a shallow clone (last N commits sufficient for heuristic analysis) to bound worker memory and execution time.
- Only the default branch of the submitted Git repository is analyzed.
- Private repositories are supported only if the student provides a personal access token alongside the URL; the token is used only during the clone and is not stored permanently.
- Oral defense grading does not go through the LLM or heuristic pipeline — the docente's grade is final and direct (Principle III).
- The existing grade decision endpoint from M8 is reused for `git_analysis` grading; a new lightweight grade endpoint is added for `defensa_oral` and stage-level grading.
- Stage ordering within an `investigacion` activity is cosmetic (display order set by the docente) and does not enforce sequential submission.

## Clarifications

### Session 2026-05-22

- Q: Are stages strictly sequential or free-order? → A: Free-order (cosmetic ordering only; students submit to any stage independently).
- Q: Is Git analysis a separate activity type or tied to project stages? → A: Separate activity type, like a tarea/homework, independent of the investigacion stages.
- Q: Does oral defense require session tracing or a detection pipeline? → A: No — it is a human-driven event; system only schedules appointments and records grades.
- Q: Does Git analysis include an LLM step? → A: Yes — same pipeline pattern as M8 (heuristic extraction → LLM advisory).
