# Research: MW7 — Grading

**Date**: 2026-05-26
**Feature**: MW7 — Grading
**Input**: spec.md + codebase analysis (MW1–MW6 patterns)

---

## Decision 1: Exam answer endpoint — separate from file upload

**Decision**: `POST /actividades/:aid/submission/exam` with `Content-Type: application/json` and body `{ "answer": "..." }` — a distinct endpoint from the file upload (`POST /actividades/:aid/submission` with FormData).

**Rationale**: The existing upload endpoint uses multipart FormData, which is unsuitable for a plain text field. A dedicated `/exam` sub-path cleanly separates the two submission types, avoids content-type confusion in the interceptor, and gives the backend a clear signal to store the answer as text rather than a file blob. The student's `getSubmission(aid)` still reads from `GET /actividades/:aid/submission` — the same read endpoint returns the exam submission with an `exam_answer` field populated.

**Alternatives considered**:
- Same endpoint with JSON body: ambiguous to the backend; complicates file-vs-text branching.
- Separate page/route for exam: overkill — the exam is one section within the existing ActivityDetailPage.

---

## Decision 2: Grade save — two API functions, GradePanel decides

**Decision**: Two functions `createGrade(sid, score, feedback)` → `POST /submissions/:sid/grade` (201) and `updateGrade(sid, score, feedback)` → `PATCH /submissions/:sid/grade` (200). The `GradePanel` component inspects `existingGrade !== null` and calls the appropriate function.

**Rationale**: REST semantics are cleaner with explicit POST/PATCH. The GradePanel already knows whether a grade exists (it receives `item.grade` from the submission list state passed via `useLocation().state`). No need for backend upsert — the frontend can make the distinction without business logic (it's just routing, not computation).

**Alternatives considered**:
- Single function always calling POST (backend upsert): less RESTful; depends on backend supporting idempotent POST.
- Try POST, catch 409 and retry PATCH: fragile and adds unnecessary round trips.

---

## Decision 3: DefensaScheduling data source — `listSubmissions` (existing) + `listDefensaSlots`

**Decision**: The `DefensaSchedulingPage` fetches `listSubmissions(cid, aid)` (already implemented in MW5) to get the list of students with submissions, then fetches `listDefensaSlots(cid, aid)` → `GET /cursos/:cid/actividades/:aid/defensa-slots` for existing slots. For each submission in the list, the docente can assign/update a slot via `PUT /submissions/:sid/defensa-slot`.

**Rationale**: Using `listSubmissions` avoids a new roster endpoint. Defensa oral requires a submission to exist before scheduling — a student without a submission doesn't get a slot. Slot is keyed per submission (one submission per student per activity → one slot per student).

**Alternatives considered**:
- Full course roster (GET /cursos/:cid/roster): shows all enrolled students, even those who haven't submitted. Adds complexity for a P3 feature.
- Embedded scheduling panel on ActividadDetailPage: makes the page too complex and inconsistent with MW5/MW6 pattern of separate sub-pages.

---

## Decision 4: InvestigaciónStage — `PATCH /submissions/:sid/stage`

**Decision**: `PATCH /submissions/:sid/stage` with body `{ "stage": "<value>" }` → returns `{ "stage": "<value>" }` (200). Stage is stored on the submission record. No dedicated stage-history resource for MW7.

**Rationale**: Minimal surface area for a P3 feature. The stage is a single field on the submission. A PATCH to the stage sub-resource avoids polluting the main submission PATCH endpoint. Frontend updates `item.stage` in local state after a successful save (no reload).

**Alternatives considered**:
- Stage history log: out of scope for MW7, could be MW9/Analytics.
- Field on main submission PATCH: mixes grading concerns with file replacement concerns.

---

## Decision 5: Grade + stage embedded in SubmissionResponse and SubmissionListItem

**Decision**: Both `SubmissionResponse` (student read endpoint) and `SubmissionListItem` (docente list endpoint) gain three nullable fields: `grade: GradeDecision | null`, `stage: InvestigaciónStage | null`, `exam_answer: string | null`.

**Rationale**: The student's `ActividadDetailPage` already calls `getSubmission(aid)` on mount. Adding `grade` and `exam_answer` to that response lets the student see their grade and exam answer without an additional API call. Similarly, adding `grade` to `SubmissionListItem` lets the docente see which submissions are graded from the list (e.g., "graded" badge on the row) without opening each one.

**Alternatives considered**:
- Separate `GET /submissions/:sid/grade` on the detail page: requires an extra round-trip every time the page loads. Adds complexity to both pages for no benefit if the backend can include it cheaply.

---

## Decision 6: DefensaSchedulingPage — separate page, consistent with MW5/MW6 pattern

**Decision**: New page at `src/pages/docente/DefensaSchedulingPage.tsx` with route `/docente/cursos/:cid/actividades/:aid/defensa-slots`. Linked from docente `ActividadDetailPage` with a "Programar Defensas" button that is shown only when `actividad.tipo === 'defensa_oral'`. No co-docente wrapper needed (co-docente reads slots from the SubmissionDetailPage grade area).

**Rationale**: Consistent with how "Ver Entregas" (MW5) and "Ver Sesiones en Vivo" (MW6) are separate pages. Keeps `ActividadDetailPage` from becoming a monolith. The scheduling page fetches its own data and can be bookmarked directly.

**Alternatives considered**:
- Embedded panel on ActivityDetailPage: inconsistent with established navigation pattern; makes the page unwieldy for large classes.
- Co-docente wrapper page: overkill for P3 — co-docente doesn't need to schedule, only view grades.
