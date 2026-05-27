# Data Model: MW7 — Grading

**Feature**: MW7 — Grading
**Date**: 2026-05-26

---

## New Types (add to `src/types/academic.ts`)

### `GradeDecision`
A grade assigned by a docente to a submission. Linked 1-to-1 with a submission.

```ts
export interface GradeDecision {
  id: string
  submission_id: string
  score: number           // integer 0–100
  feedback: string
  graded_at: string       // ISO 8601
  graded_by: string       // docente user_id
}
```

---

### `InvestigaciónStage`
String union representing the four linear stages of an `informe` activity submission.

```ts
export type InvestigaciónStage = 'propuesta' | 'desarrollo' | 'revisión' | 'aprobado'
```

---

### `DefensaSlot`
A date/time slot assigned by a docente to a student for a `defensa_oral` activity.

```ts
export interface DefensaSlot {
  submission_id: string
  scheduled_at: string    // ISO 8601 — the assigned datetime
  location: string | null // optional room/link descriptor
}
```

---

### `DefensaSlotsResponse`
Returned by `GET /cursos/:cid/actividades/:aid/defensa-slots`. Pairs each submission with its slot.

```ts
export interface DefensaSlotsResponse {
  actividad_id: string
  entries: DefensaSlotEntry[]
}

export interface DefensaSlotEntry {
  submission: SubmissionListItem   // re-uses existing type from MW5
  slot: DefensaSlot | null         // null = no slot assigned yet
}
```

---

## Modified Types (update in `src/types/academic.ts`)

### `SubmissionResponse` — add `grade`, `stage`, `exam_answer`

```ts
// Before (MW6):
export interface SubmissionResponse {
  id: string
  actividad_id: string
  original_filename: string
  file_size_bytes: number
  mime_type: string
  submitted_at: string
  is_late: boolean
  edit_window_expires_at: string | null
  is_locked: boolean
  last_replaced_at: string | null
}

// After (MW7 — add three fields):
export interface SubmissionResponse {
  // ...all existing fields...
  grade: GradeDecision | null       // null = not yet graded
  stage: InvestigaciónStage | null  // null = no stage set (non-informe or unset)
  exam_answer: string | null        // null = non-exam activity or no answer yet
}
```

---

### `SubmissionListItem` — add `grade`, `stage`

```ts
// Before (MW6):
export interface SubmissionListItem {
  id: string
  student: StudentInfo
  original_filename: string
  file_size_bytes: number
  mime_type: string
  submitted_at: string
  is_late: boolean
  is_locked: boolean
  last_replaced_at: string | null
  edit_window_expires_at: string | null
  session_id: string | null
}

// After (MW7 — add two fields):
export interface SubmissionListItem {
  // ...all existing fields...
  grade: GradeDecision | null       // null = not yet graded (enables "graded" badge in list)
  stage: InvestigaciónStage | null  // null = no stage set
}
```

---

## Exam Submission Shape

The exam answer is not stored as a separate frontend type — the student's `SubmissionResponse` includes `exam_answer: string | null` after a successful exam submit. The POST body sent to the backend is:

```ts
// Body for POST /actividades/:aid/submission/exam (Content-Type: application/json)
interface ExamSubmitBody {
  answer: string  // the full free-form text
}
```

The response from `POST /actividades/:aid/submission/exam` is `SubmissionResponse` with `exam_answer` populated.

---

## Client-Side Component State (not backend types)

### GradePanel local state

```ts
// Internal to GradePanel.tsx
type GradePanelMode = 'editing' | 'submitting' | 'success' | 'error'

interface GradePanelState {
  mode: GradePanelMode
  score: string        // controlled input — string to allow intermediate "8" before "85"
  feedback: string     // controlled textarea
  errorMessage: string | null
}
```

### ExamForm local state

```ts
// Internal to ExamForm.tsx
type ExamFormMode = 'draft' | 'submitting' | 'submitted' | 'closed'
```

| Mode | Condition |
|------|-----------|
| `draft` | No prior exam answer; deadline not passed |
| `submitting` | POST in-flight |
| `submitted` | Successful POST; shows answer read-only + timestamp |
| `closed` | Deadline passed with or without prior answer |
