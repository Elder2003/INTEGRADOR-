# Feature Specification: MW8 — Git & AI Results

**Feature Branch**: `008-mw8-git-ai-results`

**Created**: 2026-05-27

**Status**: Draft

**Input**: User description: "lets do the mw8"

---

## Overview

MW8 adds two complementary surfaces to the platform:

1. **Git submission** — students submit a git repository URL as an alternative to file upload for activities that require it. Both public and private repositories are supported: for private repos the student adds the platform's system git account as a collaborator before submitting. No credential fields appear on the form — the form is always a single URL input.
2. **Analysis reports** — docente views three auto-generated reports per submission: a git analysis (commit history, file breakdown), an AI code-quality report, and a similarity/detection report. The frontend displays exactly what the backend provides; no business logic runs on the client.

---

## Clarifications

### Session 2026-05-27

- Q: When a student's repository is private, what credential mechanism does the form support? → A: Single URL field only. Private repos supported by student adding the platform's system git account (e.g., `nextlab-bot`, configured server-side) as a collaborator. The frontend displays the system account username in an instructional note on the form. No credential fields. The backend manages collaborator lifecycle (removes access when semester closes via GitHub API).
- Q: When a student updates their git URL within the edit window, what happens to existing analysis results? → A: Backend automatically invalidates all analyses when URL is updated; frontend shows all three reports as pending after a successful edit.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Student Submits a Git Repository (Priority: P1)

A student navigates to an activity that requires a git submission. They provide their repository URL and submit. On success the form switches to a read-only confirmation view showing the submitted URL and timestamp. They cannot change the URL once submitted unless the submission window is still open (same edit-window rules as file submissions).

**Why this priority**: The git submission form is the entry point for all downstream analysis — no analysis exists without it.

**Independent Test**: Sign in as student → open a `git`-enabled activity → git URL form is shown instead of file upload → enter a valid URL → click "Enviar repositorio" → confirmation view shows URL and timestamp → reload page → same confirmation shown.

**Acceptance Scenarios**:

1. **Given** a student is on a git-enabled activity with no prior submission, **When** they enter a valid git URL and click submit, **Then** the form shows a loading state, then switches to a read-only confirmation with the submitted URL and timestamp.
2. **Given** a student has already submitted, **When** they reload the page, **Then** the confirmation view is shown (not the form) with the existing URL.
3. **Given** the activity deadline has passed and no submission exists, **When** the student views the page, **Then** the git form shows a "closed" state with no submit button.
4. **Given** the submission window is still open, **When** the student clicks "Editar" and submits a new URL, **Then** the URL is updated and all three analysis reports reset to pending state.
5. **Given** a student submits an empty or obviously malformed URL (no `://`), **When** they click submit, **Then** an inline validation error appears and no API call is made.

---

### User Story 2 — Docente Views Git Analysis Report (Priority: P2)

From the submission detail page, docente can navigate to the git analysis report for a student's submission. The report shows commit metadata (count, authors, date range), a file-change summary, and any repository-level statistics returned by the backend.

**Why this priority**: Git analysis is the core value of git-mode submissions — without it, the docente has no visibility into student work.

**Independent Test**: Sign in as docente → open a submission detail page for a git submission → "Ver análisis Git" link visible → click → git analysis page loads with commit history, stats, and file breakdown as returned by the API.

**Acceptance Scenarios**:

1. **Given** a submission has a completed git analysis, **When** docente clicks "Ver análisis Git", **Then** a page shows the commit count, contributor list, date range, and file change table.
2. **Given** the analysis is still processing (pending state), **When** docente views the page, **Then** a "Análisis en curso..." message is shown with a note to check back later.
3. **Given** the analysis failed (error state), **When** docente views the page, **Then** an error message is shown with the reason returned by the API.
4. **Given** no git submission exists for the student, **When** docente views the submission detail page, **Then** the "Ver análisis Git" link is absent.

---

### User Story 3 — Docente Views AI Analysis Report (Priority: P3)

Docente accesses the AI-generated analysis report for a student's submission. The report contains structured feedback text produced by the AI backend. The docente reads the report; no editing is possible.

**Why this priority**: AI report is a key integrity signal for the docente's grading decision, but only available after the analysis pipeline completes.

**Independent Test**: Sign in as docente → open a submission with a completed AI analysis → "Ver informe IA" link visible → click → AI report page shows the full AI-generated text and any structured sections returned by the API.

**Acceptance Scenarios**:

1. **Given** a completed AI analysis, **When** docente opens the AI report page, **Then** all structured sections (e.g., summary, findings, risk level) are displayed exactly as returned by the API.
2. **Given** the AI analysis is pending, **When** docente opens the report link, **Then** a "Análisis IA en curso..." message is shown.
3. **Given** AI analysis is not available for the activity (disabled or not triggered), **When** docente views the submission detail page, **Then** no AI report link is shown.

---

### User Story 4 — Docente Views Detection Report (Priority: P3)

Docente accesses the similarity/detection report for a student's submission. The report contains similarity scores and flagged segments. All data is read-only and sourced from the backend detection pipeline.

**Why this priority**: Detection data is parallel to AI analysis in value and implementation complexity. Both are P3 display-only features.

**Independent Test**: Sign in as docente → open a submission with a completed detection report → "Ver informe de detección" link visible → click → detection report shows similarity score and any flagged segments.

**Acceptance Scenarios**:

1. **Given** a completed detection report, **When** docente opens the page, **Then** the similarity score and flagged segments are displayed read-only.
2. **Given** detection is pending, **When** docente opens the report link, **Then** a "Detección en curso..." message is shown.
3. **Given** detection is disabled for the activity, **When** docente views the submission detail page, **Then** no detection report link is shown.

---

### Edge Cases

- Student submits a git URL that the backend cannot clone (private repo, invalid URL, auth failure) — the backend returns an error after the submission is created; the frontend shows "submitted" confirmation immediately and the git analysis will show the failure state.
- Git analysis, AI report, and detection report may each be in a different state (one pending, one done, one error) — each section is independent.
- Co-docente sees all analysis reports read-only (no differences from docente view beyond role-gating of edit controls, which do not exist on report pages).
- Student navigates to a non-git-enabled activity — git form is not shown; standard file upload applies.
- Analysis reports contain empty data (no commits, empty similarity list) — empty states are displayed inline.
- Very long AI-generated text — displayed with scrollable pre-formatted area; no truncation.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: For activities with git submission enabled, the student portal MUST show a git URL input form (single URL field) instead of the file upload panel. No credential or token fields are shown.
- **FR-001a**: The git submission form MUST display an informational note showing the platform's system git account username (e.g., `nextlab-bot`, read from a frontend environment variable) and instructing the student to add it as a collaborator before submitting a private repository.
- **FR-002**: The git submission form MUST validate that the URL field is non-empty and contains `://` before allowing submit.
- **FR-003**: On successful git submission, the form MUST switch to a read-only confirmation view showing the submitted URL and timestamp.
- **FR-004**: If the submission edit window is still open, students MUST be able to update their git URL via an "Editar" button. After a successful update, all three analysis report sections MUST be shown as pending.
- **FR-005**: The docente submission detail page MUST show a "Ver análisis Git" link when a git submission exists.
- **FR-006**: The git analysis report page MUST display the data returned by the analysis API (commit count, authors, date range, file breakdown) or an appropriate pending/error state.
- **FR-007**: The docente submission detail page MUST show a "Ver informe IA" link when AI analysis is available for the activity.
- **FR-008**: The AI report page MUST display all structured sections returned by the API, read-only.
- **FR-009**: The docente submission detail page MUST show a "Ver informe de detección" link when detection is available for the activity.
- **FR-010**: The detection report page MUST display similarity score and flagged segments returned by the API, read-only.
- **FR-011**: All three report pages (git analysis, AI report, detection) MUST handle pending and error states with inline messages.
- **FR-012**: Co-docente MUST have read-only access to all three report pages (same content as docente, no edit controls).
- **FR-013**: Analysis report links MUST be absent from pages where the corresponding analysis is not configured or not triggered.

### Key Entities

- **GitSubmission**: A student's submitted git repository URL (public or private via collaborator access), tied to an actividad; has `submitted_at`, `repo_url`, `is_locked`, `edit_window_expires_at`. No credential fields stored on the client.
- **GitAnalysisReport**: Backend-generated git statistics for a submission; status (pending / done / error), commit count, author list, date range, file change entries.
- **AIReport**: AI-generated text analysis; status (pending / done / error), structured sections (summary, findings, risk level or similar as returned by API).
- **DetectionReport**: Similarity detection result; status (pending / done / error), overall similarity score, flagged segments with match source and percentage.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Student can submit a git URL in under 30 seconds from opening the activity page.
- **SC-002**: Docente can open and read a completed git analysis report within 3 seconds of clicking the link.
- **SC-003**: All three analysis report pages correctly display "pending", "error", and "complete" states — confirmed by manual smoke test across all three states.
- **SC-004**: No analysis-report link is visible on submission detail pages where the corresponding analysis is not available — role-guard and availability checks pass 100% of test cases.
- **SC-005**: Co-docente can access all three read-only report pages without being redirected or shown an error.

---

## Assumptions

- Git submission is an activity-level flag (`git_enabled`) set by the docente when creating/editing the activity — the student sees the git form only for activities with this flag.
- The git submission form contains a single URL field only. Private repository access is achieved by the student adding the platform's system git account (username read from `VITE_GIT_SYSTEM_ACCOUNT` env var, default `nextlab-bot`) as a collaborator — no credentials are collected by the frontend. The backend manages collaborator lifecycle via GitHub API.
- Analysis reports are triggered automatically by the backend when a git submission is received — the frontend never triggers analysis manually.
- The three report pages (git analysis, AI, detection) each have their own URL and are navigated to from the submission detail page via links, not shown inline.
- Co-docente access to report pages uses the same `/codocente/...` route prefix pattern established in MW5 and MW6.
- Activity-level flags indicating which analyses are enabled (`ai_enabled`, `detection_enabled`) are fields on the `Actividad` / `StudentActividadEntry` object returned by the existing activity APIs — no new API calls needed to determine availability.
- The `SubmissionResponse` for git submissions may differ from file submissions (no `original_filename`, `file_size_bytes`, `mime_type`) — the frontend adapts based on submission type.
