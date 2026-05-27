# Implementation Plan: MW7 — Grading

**Branch**: `013-appeals-final-audit` | **Date**: 2026-05-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/007-mw7-grading/spec.md`
**Research**: `specs/007-mw7-grading/research.md`
**Data Model**: `specs/007-mw7-grading/data-model.md`
**Contracts**: `specs/007-mw7-grading/contracts/grading.md`

---

## Summary

Build the grading layer across all activity types. Docente uses a `GradePanel` on the existing submission detail page to assign a score (0–100) + feedback; the grade is immediately visible to the student. Activity-type-specific features are rendered conditionally: `ExamForm` (textarea) for `examen`, `StageSelector` for `informe`, `DefensaSlotDisplay` / `DefensaSchedulingPage` for `defensa_oral`. Co-docente sees grades read-only. No new pages are required except `DefensaSchedulingPage`.

Key decisions from research:
- Exam submit: `POST /actividades/:aid/submission/exam` with JSON body — separate from FormData file upload endpoint
- Grade save: `createGrade` (POST 201) vs `updateGrade` (PATCH 200); `GradePanel` inspects `item.grade !== null` to decide
- Stage: `PATCH /submissions/:sid/stage` — minimal sub-resource, local state update on success
- Defensa scheduling: separate page reusing `listSubmissions` (MW5) as roster; `PUT /submissions/:sid/defensa-slot`
- Three nullable fields added to `SubmissionResponse` and `SubmissionListItem`: `grade`, `stage`, `exam_answer`

---

## Technical Context

**Language/Version**: TypeScript 5.9 (strict mode)

**Primary Dependencies**: React 19, React Router v7, axios, Tailwind CSS v4, Lucide React

**Storage**: N/A — frontend only; grade data stored by nextlab-api

**Testing**: Manual smoke tests via `quickstart.md`. No automated tests for MW7.

**Target Platform**: Browser (React SPA at `http://localhost:5173`)

**Performance Goals**: Grade save < 30 s end-to-end (SC-001); student sees grade on next page load < 1 s (SC-002).

**Constraints**: No external UI component library. No business logic on the frontend. `GradePanel` must be absent from student views — role-gated rendering enforced at component level.

**Scale/Scope**: 1 new API module, 5 new types + 2 type updates, 5 new components, 1 new page, 4 updated pages, 1 new route.

---

## Constitution Check

*GATE: Must pass before implementation.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Role-Gated Rendering | ✅ Pass | `GradePanel` rendered only inside docente SubmissionDetailPage — student page shows `GradeDisplay` (read-only). `StageSelector` shown only to docente. `DefensaSchedulingPage` under docente ProtectedRoute. |
| II. API Client Isolation | ✅ Pass | All calls through `src/api/grading.ts` → `api` from `../lib/api`. No raw `fetch()`. |
| III. React Quality Gate | ✅ Pass | React Doctor scan required after all MW7 components complete. |
| IV. Token Security | ✅ Pass | Bearer header via axios interceptor. No WebSocket in MW7. |
| V. Read-Only Evidence | ✅ Pass | `GradeDisplay` (student) and co-docente view are purely read-only. `ExamForm` becomes read-only after submit (`submitted` mode). |
| VI. Brand Baseline | ✅ Pass | All new components use Tailwind CSS v4 + dark palette. Named exports only. |
| VII. Thin Client | ✅ Pass | `GradePanel` decides POST vs PATCH by inspecting `item.grade` from location state — this is routing, not computation. No pass/fail logic on frontend. Stage label displayed as returned by API. |

No violations. All gates pass.

---

## Project Structure

### Documentation (this feature)

```text
specs/007-mw7-grading/
├── plan.md            ← this file
├── research.md        ← Phase 0 output (6 decisions)
├── data-model.md      ← Phase 1 output
├── quickstart.md      ← Phase 1 output
├── contracts/
│   └── grading.md     ← Phase 1 output
└── tasks.md           ← Phase 2 output (speckit-tasks)
```

### Source Code Changes

Files to **create** (new):

```text
src/
├── api/
│   └── grading.ts                         # 7 functions (createGrade, updateGrade,
│                                          #   submitExamAnswer, updateStage,
│                                          #   listDefensaSlots, assignDefensaSlot,
│                                          #   getMyDefensaSlot)
│
├── components/
│   ├── docente/
│   │   ├── GradePanel.tsx                 # Score input + feedback textarea + submit
│   │   │                                  # Modes: editing / submitting / success / error
│   │   └── StageSelector.tsx              # Dropdown + "Actualizar etapa" button for informe
│   └── student/
│       ├── GradeDisplay.tsx               # Read-only score + feedback (or "Pendiente")
│       ├── ExamForm.tsx                   # Textarea + submit for examen activities
│       └── DefensaSlotDisplay.tsx         # Assigned slot or "Defensa aún no programada"
│
└── pages/
    └── docente/
        └── DefensaSchedulingPage.tsx      # Scheduling table for defensa_oral activities
```

Files to **update** (existing):

```text
src/
├── types/
│   └── academic.ts        # Add 5 types (GradeDecision, InvestigaciónStage, DefensaSlot,
│                          #              DefensaSlotsResponse, DefensaSlotEntry)
│                          # Update SubmissionResponse: add grade, stage, exam_answer
│                          # Update SubmissionListItem: add grade, stage
│
├── pages/
│   ├── docente/
│   │   ├── SubmissionDetailPage.tsx   # Add GradePanel + StageSelector (conditional on tipo)
│   │   └── ActividadDetailPage.tsx    # Add "Programar Defensas" link (only for defensa_oral)
│   └── student/
│       └── ActividadDetailPage.tsx    # Add GradeDisplay, ExamForm (replaces upload for examen),
│                                      #   DefensaSlotDisplay (only for defensa_oral)
│
└── router/
    └── index.tsx          # Add DefensaSchedulingPage route
```

---

## Key Implementation Details

### `src/api/grading.ts`

```ts
import { api } from '../lib/api'
import type { GradeDecision, InvestigaciónStage, SubmissionResponse,
              DefensaSlotsResponse, DefensaSlot } from '../types/academic'

function toApiError(err: unknown): never { /* same pattern as student.ts */ }

export async function createGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>
  // POST /submissions/${sid}/grade  → 201

export async function updateGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>
  // PATCH /submissions/${sid}/grade → 200

export async function submitExamAnswer(aid: string, answer: string): Promise<SubmissionResponse>
  // POST /actividades/${aid}/submission/exam → 201

export async function updateStage(sid: string, stage: InvestigaciónStage): Promise<{ stage: InvestigaciónStage }>
  // PATCH /submissions/${sid}/stage → 200

export async function listDefensaSlots(cid: string, aid: string): Promise<DefensaSlotsResponse>
  // GET /cursos/${cid}/actividades/${aid}/defensa-slots → 200

export async function assignDefensaSlot(sid: string, scheduledAt: string, location: string | null): Promise<DefensaSlot>
  // PUT /submissions/${sid}/defensa-slot → 200

export async function getMyDefensaSlot(aid: string): Promise<DefensaSlot | null>
  // GET /actividades/${aid}/defensa-slot/me → 200 (body may be null)
```

All functions use the same `toApiError` pattern defined locally (not imported from lib/api).

---

### `src/components/docente/GradePanel.tsx`

Props:
```ts
interface GradePanelProps {
  submissionId: string
  existingGrade: GradeDecision | null  // null → POST, non-null → PATCH
  readOnly?: boolean                   // true for co-docente
}
```

Render logic:
- `readOnly={true}`: show score + feedback with no input fields (co-docente view)
- `existingGrade !== null` and not in editing mode: show score + feedback + "Editar calificación" button
- Editing mode (`existingGrade === null` or after clicking edit): show score `<input type="number" min="0" max="100">` + feedback `<textarea>` + "Calificar" button
- Submitting: button disabled, shows "Calificando..."
- Success: inline "Calificación guardada ✓" message visible for 3 seconds, then reverts to read-only view
- Error: inline error message; form remains editable

Decision logic (no business computation — just routing):
```ts
const save = existingGrade === null ? createGrade : updateGrade
```

---

### `src/components/student/GradeDisplay.tsx`

Props:
```ts
interface GradeDisplayProps {
  grade: GradeDecision | null
  hasSubmission: boolean
}
```

- `!hasSubmission`: render nothing (no grade section shown)
- `hasSubmission && grade === null`: render "Pendiente de calificación"
- `hasSubmission && grade !== null`: render score (`85 / 100`) + feedback text (read-only)

---

### `src/components/student/ExamForm.tsx`

Props:
```ts
interface ExamFormProps {
  actividadId: string
  existingAnswer: string | null  // from SubmissionResponse.exam_answer
  submittedAt: string | null     // from SubmissionResponse.submitted_at
  deadline: string               // ISO 8601 — for "closed" detection
  sessionGate: boolean           // if trace_enabled && session not active → block
}
```

Mode derivation:
- `existingAnswer !== null` → `submitted` (read-only, show answer + timestamp)
- `new Date() > new Date(deadline)` and `existingAnswer === null` → `closed` ("Examen cerrado")
- `sessionGate === true` → blocked by session (show "Inicia la sesión de monitoreo...")
- Otherwise → `draft`

On submit:
1. Set mode to `submitting`
2. Call `submitExamAnswer(actividadId, answerText)`
3. On success: set mode to `submitted`, store returned `exam_answer` + `submitted_at`
4. On error: set mode to `draft`, show inline error

---

### `src/components/docente/StageSelector.tsx`

Props:
```ts
interface StageSelectorProps {
  submissionId: string
  currentStage: InvestigaciónStage | null  // null → default to 'propuesta' in selector
}
```

Renders a `<select>` with options `propuesta | desarrollo | revisión | aprobado` + "Actualizar etapa" button. On save: call `updateStage(submissionId, selectedStage)` → update local state on success. No page reload.

---

### `src/components/student/DefensaSlotDisplay.tsx`

Props:
```ts
interface DefensaSlotDisplayProps {
  slot: DefensaSlot | null
}
```

- `slot === null`: "Defensa aún no programada"
- `slot !== null`: formatted datetime + optional location string

---

### `src/pages/docente/DefensaSchedulingPage.tsx`

Params: `cid`, `aid` from `useParams`.

On mount:
1. `listDefensaSlots(cid, aid)` — get submissions + existing slots.

Renders a table:
- Columns: Estudiante, Email, Fecha/Hora, Ubicación (opcional), Acción
- Each row has `<input type="datetime-local">` and `<input type="text" placeholder="Aula / Link">` pre-filled if slot exists
- "Guardar" button per row — calls `assignDefensaSlot(submission.id, scheduledAt, location)`
- Inline error on 409 (conflict)
- Back link: `← Actividad` → `/docente/cursos/${cid}/actividades/${aid}`

---

### Docente `SubmissionDetailPage` updates

The page already receives `SubmissionListItem` via `useLocation().state` (which now includes `grade` and `stage` from MW7 type updates). The docente is identified by their role from `useAuth()`.

Add below the existing download button area:

```tsx
{/* Stage selector — only for informe */}
{actividad?.tipo === 'informe' && (
  <StageSelector submissionId={sid!} currentStage={item.stage} />
)}

{/* Grade panel — docente editable, co-docente read-only */}
<GradePanel
  submissionId={sid!}
  existingGrade={item.grade}
  readOnly={user?.role === 'co_docente'}
/>
```

Note: `actividad` must be available on the page — currently the page receives the submission via state but may need to also receive the actividad (already passed in MW5's navigation from the list page) or derive tipo from state.

---

### Student `ActividadDetailPage` updates

The page already fetches `getSubmission(aid)` (returns `SubmissionResponse` with new `grade`, `stage`, `exam_answer` fields from MW7).

**Exam form** — replaces upload for `examen` tipo:

```tsx
{actividad.tipo === 'examen' ? (
  <ExamForm
    actividadId={aid!}
    existingAnswer={submission?.exam_answer ?? null}
    submittedAt={submission?.submitted_at ?? null}
    deadline={actividad.deadline}
    sessionGate={actividad.trace_enabled && sessionStatus !== 'active'}
  />
) : (
  /* existing file upload form */
)}
```

**Grade display** — below submission card:

```tsx
<GradeDisplay
  grade={submission?.grade ?? null}
  hasSubmission={submission !== null}
/>
```

**Defensa slot** — only for `defensa_oral`:

```tsx
{actividad.tipo === 'defensa_oral' && (
  <DefensaSlotDisplay slot={defensaSlot} />
)}
```

`defensaSlot` fetched on mount via `getMyDefensaSlot(aid)` (only when `actividad.tipo === 'defensa_oral'`).

---

### Docente `ActividadDetailPage` updates

Add "Programar Defensas" link alongside "Ver Entregas":

```tsx
{actividad.tipo === 'defensa_oral' && (
  <Link
    to={`/docente/cursos/${cursoId}/actividades/${aid}/defensa-slots`}
    className="..."
  >
    Programar Defensas
  </Link>
)}
```

---

### Router addition

```ts
// Inside ProtectedRoute allowedRoles={['docente']}
{ path: '/docente/cursos/:cid/actividades/:aid/defensa-slots', element: <DefensaSchedulingPage /> }
```

---

## Scope Adjustments (vs. spec)

| Spec claim | Actual behavior | Source |
|---|---|---|
| GradePanel shows grade immediately in list | `SubmissionListItem.grade` added — docente list can show a "Calificado" badge | Research Decision 5 |
| Co-docente sees grade read-only | Same `GradePanel` component with `readOnly={true}` prop — no separate page needed | Research Decision 2 |
| DefensaSchedulingPage uses course roster | Uses `listSubmissions` (MW5) → only students with submissions appear | Research Decision 3 |
| Exam form has per-question rendering | Single free-form textarea — backend handles structure (clarification Q1) | Spec clarification |

---

## Complexity Tracking

No constitution violations. All new components are thin display/input wrappers — zero business logic. `GradePanel` POST vs PATCH decision is routing (data-driven conditional), not computation.
