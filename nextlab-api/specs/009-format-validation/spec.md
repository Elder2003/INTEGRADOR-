# Feature Specification: M9 — Format Validation

**Feature Branch**: `009-student-portal`

**Created**: 2026-05-22

**Status**: Draft

**Input**: User description: "Lets do the milestone 9"

## Clarifications

### Session 2026-05-22

- Q: Should format validation run at file upload or at session close? → A: Triggered at session close via the `session.closed` event — same unified analysis job as M8 detection.
- Q: Should format rules be a new endpoint or extend the existing actividad endpoint? → A: Extend the existing actividad create/update endpoint with an optional `format_rules` field — no new URL surface.
- Q: Should students be able to see their own format compliance report? → A: Docente-only — same 403 restriction as the M8 detection report; Principle IV bars students from any analysis output on their own sessions.
- Q: One format report per submission file or per session? → A: One per submission file — each upload (including versioned re-uploads) gets its own independent report; append-only per Principle I.
- Q: Should `require_title_section` be included in M9 or deferred to M12? → A: Deferred to M12 — M9 ships with four mechanical rules only: `min_pages`, `max_pages`, `min_word_count`, `required_types`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Worker Validates Format of a Submitted File (Priority: P1)

After a student submits an exam file (PDF or Word document), the async worker picks up the queued job and runs format analysis on it. The validator inspects the file's structure — page count, word count, detected sections, font usage, and metadata — and produces a format compliance report. This report is stored regardless of whether the file meets any configured rules; the submission is never blocked (Principle VII).

**Why this priority**: Format analysis is the fastest, most reliable signal in the pipeline. It requires no external AI call and is always deterministic. It must run before the LLM step and its result must be independently available to the docente even if later analysis fails.

**Independent Test**: Can be fully tested by uploading a known PDF or DOCX, waiting up to 3 minutes, and verifying that a format report record exists with: detected file type, page count, word count, and a pass/fail list for each configured rule — independently of any detection engine result.

**Acceptance Scenarios**:

1. **Given** a student submits a PDF exam file and the session closes, **When** the async worker processes the job, **Then** within 3 minutes a format report is stored with: detected file type (`pdf`), page count, word count, character count, and metadata fields (author, creation date if present).
2. **Given** a student submits a DOCX file, **When** the worker validates it, **Then** the same metadata fields are extracted and stored; no error is produced due to the different file format.
3. **Given** a submission file is corrupt or unreadable, **When** the worker tries to parse it, **Then** a format report is still stored with `parse_status = 'failed'` and the reason — the submission itself is never rolled back.
4. **Given** a file with no textual content (e.g., a scanned image PDF with no OCR layer), **When** the worker extracts text, **Then** `word_count = 0` is recorded and a `low_text_content` flag is set in the report — not an error, just a finding.

---

### User Story 2 — Format Report Checked Against Configurable Rules (Priority: P2)

Each activity can have a set of format rules defined by the docente: minimum page count, maximum page count, minimum word count, required file type (PDF only, or PDF and DOCX), and whether a title section is required. The format validator checks the extracted metadata against these rules and records each rule's pass/fail outcome in the report.

**Why this priority**: Rule evaluation is what turns raw metadata into actionable feedback for the docente. Without it, the format report is only informational; with it, the docente sees immediately whether the student's submission met the stated requirements.

**Independent Test**: Can be fully tested by creating an activity with explicit rules (e.g., min 3 pages, PDF only), submitting a DOCX of 1 page, and verifying the format report shows the `file_type` rule as failed and the `min_pages` rule as failed, while the submission itself is accepted and accessible.

**Acceptance Scenarios**:

1. **Given** an activity has a rule `min_pages = 3` and the submitted PDF has 2 pages, **When** the worker evaluates the report, **Then** the stored report includes a rule-check entry for `min_pages` with status `fail` and the actual value (`2`).
2. **Given** an activity has `required_type = pdf` and a DOCX is submitted, **When** the rule is checked, **Then** the report includes a `file_type` rule entry with status `fail` — but the submission is not rejected.
3. **Given** all configured rules pass, **When** the report is stored, **Then** the overall `compliant` flag is `true`.
4. **Given** an activity has no format rules configured, **When** validation runs, **Then** a report is still generated with raw metadata only and `compliant = true` (no rules means no violations).

---

### User Story 3 — Docente Views the Format Compliance Report (Priority: P3)

A docente opens a submission and sees the format report alongside the detection report. The format report shows: file type detected, page count, word count, creation timestamp, and the rule-by-rule pass/fail breakdown. The docente uses this as additional context before assigning the final grade.

**Why this priority**: The docente-facing report is the primary deliverable of format validation. The format report must be readable alongside the detection report without requiring the docente to navigate to a separate view.

**Independent Test**: Can be fully tested by requesting the detection report endpoint as a docente after a submission has been processed, confirming a `format_report` sub-object is present with `compliant` flag, `page_count`, `word_count`, and at least one rule result entry.

**Acceptance Scenarios**:

1. **Given** a submission has been fully processed, **When** a docente requests the detection report, **Then** the response includes a `format_report` field containing: `compliant` (bool), `file_type`, `page_count`, `word_count`, and `rule_results` (array of rule name + status + actual value).
2. **Given** format validation failed (parse error), **When** a docente requests the report, **Then** the `format_report` field shows `parse_status: "failed"` and a human-readable reason — the rest of the report (heuristic, LLM) is still visible.
3. **Given** format validation has not yet completed, **When** a docente requests the report, **Then** the `format_report` field is absent or shows `status: "pending"` — the endpoint does not block.

---

### User Story 4 — Docente Configures Format Rules for an Activity (Priority: P4)

When creating or editing an exam activity, the docente can set format rules: minimum pages, maximum pages, minimum word count, required file types, and whether a title is required. These rules are stored with the activity and applied automatically when any submission for that activity is validated.

**Why this priority**: Without rule configuration, format validation only produces raw metadata. The docente must be able to define expectations upfront so the compliance report is meaningful at grading time.

**Independent Test**: Can be fully tested by creating an activity with rules via the activity API, submitting a file, and verifying the format report evaluates each configured rule independently.

**Acceptance Scenarios**:

1. **Given** a docente creates an activity with `min_pages = 2`, `max_pages = 10`, `required_types = ["pdf"]`, **When** the activity is saved, **Then** the rules are retrievable via the activity detail endpoint.
2. **Given** a docente updates an activity's format rules, **When** a new submission is processed after the update, **Then** the new rules are applied — existing reports from prior submissions are not retroactively changed.
3. **Given** a docente sets `min_word_count = 500`, **When** a submission with 300 words is validated, **Then** the rule result shows `min_word_count: fail, actual: 300, required: 500`.

---

### Edge Cases

- What happens when a PDF is password-protected? → Report stored with `parse_status = 'failed'`, reason = `"password_protected"`.
- What happens if the worker receives a file that is not PDF or DOCX (e.g., a ZIP or image)? → Report stored with `parse_status = 'failed'`, `detected_type = 'unsupported'`, reason = `"unsupported_file_type"`.
- What if the same submission is enqueued twice? → Format validation is idempotent; a second run returns the existing report without inserting a duplicate row.
- What if the file is empty (0 bytes)? → Report stored with `parse_status = 'failed'`, reason = `"empty_file"`.
- What if MinIO is unavailable when the worker tries to fetch the file? → Job retried via Redis Streams PEL; report not stored until the file is successfully retrieved.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Format validation MUST be triggered by the `session.closed` event (same as M8 detection) — not at file upload time. The async worker MUST produce a format report within 3 minutes of session close (Principle II SLA).
- **FR-002**: Format reports MUST be stored as append-only records; no `UPDATE` or `DELETE` after creation (Principle I).
- **FR-003**: The format validator MUST extract and store: file type, page count, word count, character count, and available document metadata (author, creation date).
- **FR-004**: The format validator MUST support PDF and DOCX file types. Unsupported types MUST produce a `parse_status = 'failed'` record, not an error response.
- **FR-005**: There is exactly one format report per submission file. The report is idempotent — if the validation job runs twice for the same submission, the second run MUST return the existing record without creating a duplicate. Versioned re-uploads each produce their own report (one per `submission_id`).
- **FR-006**: Activity format rules MUST be configurable by the docente via the existing actividad create/update endpoint using an optional `format_rules` field containing four mechanical rules: `min_pages` (int), `max_pages` (int), `min_word_count` (int), `required_types` (string array, e.g. `["pdf"]` or `["pdf","docx"]`). No new endpoint is introduced. The `require_title_section` rule is out of scope for M9 (deferred to M12).
- **FR-007**: The format validator MUST evaluate each configured rule independently and store each rule result (name, status, actual value, required value) in the format report.
- **FR-008**: A submission file MUST always be accepted regardless of format rule violations (Principle VII). The format report is evidence for the docente, not a gate.
- **FR-009**: The `GET /sessions/:id/detection-report` endpoint MUST include the format report as a sub-field (`format_report`) when available. Access is restricted to docentes only — students receive 403 (Principle IV).
- **FR-010**: If format validation fails (parse error, unsupported type), the `format_report` field MUST still be present with a machine-readable `parse_status` and a human-readable reason.
- **FR-011**: Activities with no format rules configured MUST still receive a format report with raw metadata; `compliant` defaults to `true` when no rules are defined.
- **FR-012**: The format rule configuration MUST be scoped to the docente's own activities — a docente MUST NOT be able to read or modify another docente's rules (Principle IV).

### Key Entities

- **FormatReport**: Stores the outcome of one format validation run. Fields: `id`, `submission_id` (FK), `session_id` (FK), `parse_status` (`success`/`failed`), `file_type`, `page_count`, `word_count`, `char_count`, `doc_metadata` (JSONB), `compliant` (bool), `rule_results` (JSONB array), `created_at`.
- **ActivityFormatRules**: Per-activity rule set managed by the docente. Fields: `id`, `actividad_id` (FK, UNIQUE), `min_pages` (nullable int), `max_pages` (nullable int), `min_word_count` (nullable int), `required_types` (TEXT[], nullable), `created_by` (docente FK), `created_at`, `updated_at`. (`require_title_section` deferred to M12.)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Format reports are available within 3 minutes of session close for 95% of submissions under normal load.
- **SC-002**: 100% of submitted files — regardless of format compliance — receive a stored format report (no silent failures).
- **SC-003**: Docentes can view format rule results per submission without navigating away from the detection report screen.
- **SC-004**: Format rule violations are surfaced per-rule, not as a single pass/fail, allowing the docente to understand exactly which requirements were missed.
- **SC-005**: Format validation parse failures are self-descriptive: the reason is human-readable and actionable for the docente within 5 seconds of opening the report.

## Assumptions

- The M5 async worker infrastructure (Redis Streams, job dispatch) is already in place and will be extended — no new worker binary is needed.
- MinIO already stores submitted files via the M4 storage pipeline; format validation reads from MinIO using the existing `storage_key` pattern.
- The M8 detection report endpoint (`GET /sessions/:id/detection-report`) will be extended to include the `format_report` sub-field — no new endpoint is needed for basic report access.
- PDF parsing uses the `pdfcpu` library (already a dependency from M7 RAG text extraction); DOCX parsing will use `unidoc` or equivalent Go library.
- Format rule configuration is a docente-controlled property stored with the activity; admin-managed templates are out of scope for M9 (deferred to M12 Admin Modules).
- `require_title_section` rule is out of scope for M9 and deferred to M12 (Admin Modules / format templates).
- OCR is out of scope: image-only PDFs produce `word_count = 0` and a `low_text_content` flag; the system does not attempt optical character recognition.
