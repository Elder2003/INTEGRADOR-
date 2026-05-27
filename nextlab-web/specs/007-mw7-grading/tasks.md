# Tasks: MW7 — Grading

**Feature**: MW7 — Grading
**Plan**: [plan.md](plan.md) | **Spec**: [spec.md](spec.md) | **Contracts**: [contracts/grading.md](contracts/grading.md)
**Total tasks**: 19 | **Last updated**: 2026-05-26

---

## Phase 1 — Setup (Shared Foundation)

**Goal**: Add all new types, create stubs for every new file, and create the `grading.ts` API module. All tasks are independent — run in parallel.

- [ ] T001 [P] Add 5 new types to `src/types/academic.ts` and update 2 existing interfaces. Append after the last existing type: `GradeDecision { id: string; submission_id: string; score: number; feedback: string; graded_at: string; graded_by: string }`. Then add: `export type InvestigaciónStage = 'propuesta' | 'desarrollo' | 'revisión' | 'aprobado'`. Then add: `DefensaSlot { submission_id: string; scheduled_at: string; location: string | null }`. Then add: `DefensaSlotEntry { submission: SubmissionListItem; slot: DefensaSlot | null }`. Then add: `DefensaSlotsResponse { actividad_id: string; entries: DefensaSlotEntry[] }`. Update the existing `SubmissionResponse` interface — add three fields after `last_replaced_at`: `grade: GradeDecision | null`, `stage: InvestigaciónStage | null`, `exam_answer: string | null`. Update the existing `SubmissionListItem` interface — add two fields after `session_id`: `grade: GradeDecision | null`, `stage: InvestigaciónStage | null`. All new interfaces need `export` keyword.

- [ ] T002 [P] Create minimal named-export stubs for 6 new component/page files. Each file exports a named component returning `<p>ComponentName</p>` (no logic). Create: `src/components/docente/GradePanel.tsx` (export named `GradePanel`), `src/components/docente/StageSelector.tsx` (export named `StageSelector`), `src/components/student/GradeDisplay.tsx` (export named `GradeDisplay`), `src/components/student/ExamForm.tsx` (export named `ExamForm`), `src/components/student/DefensaSlotDisplay.tsx` (export named `DefensaSlotDisplay`), `src/pages/docente/DefensaSchedulingPage.tsx` (export named `DefensaSchedulingPage`). These stubs are replaced in later phases. Use `export function ComponentName() { return <p>ComponentName</p>; }` format.

- [ ] T003 [P] Create `src/api/grading.ts` — implement 7 functions using the same `toApiError` helper pattern as `src/api/student.ts`. Import `api` from `../lib/api`, `ApiError` from `../types`, `AxiosError` from `axios`, and types from `../types/academic`. Define a local `toApiError` function (same pattern: `function toApiError(err: unknown): never { if (err instanceof AxiosError && err.response?.data?.error) { throw new ApiError(err.response.data.error, err.response.status) } throw err }`). Functions: `export async function createGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>` → `POST /submissions/${sid}/grade` with body `{ score, feedback }`; `export async function updateGrade(sid: string, score: number, feedback: string): Promise<GradeDecision>` → `PATCH /submissions/${sid}/grade` with body `{ score, feedback }`; `export async function submitExamAnswer(aid: string, answer: string): Promise<SubmissionResponse>` → `POST /actividades/${aid}/submission/exam` with body `{ answer }`; `export async function updateStage(sid: string, stage: InvestigaciónStage): Promise<{ stage: InvestigaciónStage }>` → `PATCH /submissions/${sid}/stage` with body `{ stage }`; `export async function listDefensaSlots(cid: string, aid: string): Promise<DefensaSlotsResponse>` → `GET /cursos/${cid}/actividades/${aid}/defensa-slots`; `export async function assignDefensaSlot(sid: string, scheduledAt: string, location: string | null): Promise<DefensaSlot>` → `PUT /submissions/${sid}/defensa-slot` with body `{ scheduled_at: scheduledAt, location }`; `export async function getMyDefensaSlot(aid: string): Promise<DefensaSlot | null>` → `GET /actividades/${aid}/defensa-slot/me` — response body may be null (check `res.data !== null && res.data !== ''`). All catch blocks call `toApiError(err)`.

---

## Phase 2 — Foundational

**Goal**: Wire the new `DefensaSchedulingPage` route so the router compiles. Sequential — depends on T002.

- [ ] T004 Update `src/router/index.tsx` — import `DefensaSchedulingPage` from `../pages/docente/DefensaSchedulingPage`. Inside the existing `ProtectedRoute allowedRoles={['docente']}` children array, add one new route after the existing defensa/submissions routes: `{ path: '/docente/cursos/:cid/actividades/:aid/defensa-slots', element: <DefensaSchedulingPage /> }`. No co-docente route needed for the scheduling page. Depends on T002.

---

## Phase 3 — US1: Docente Grades a Submission (Priority: P1) 🎯 MVP

**Goal**: Docente can assign a score (0–100) + feedback from the submission detail page. Grade is immediately shown read-only after save. Co-docente sees it read-only with no input fields.

**Independent test criteria**: Sign in as docente → open a submission detail page → GradePanel visible with score input and feedback textarea → enter 85 + feedback → click "Calificar" → button disables ("Calificando...") → on success, "Calificación guardada" message for 3 s → GradePanel shows 85 + feedback read-only. Sign in as co-docente → same page → grade visible read-only, no input or submit button.

- [ ] T005 [P] [US1] Implement `src/components/docente/GradePanel.tsx` (replace stub) — named export `GradePanel`. Props interface: `{ submissionId: string; existingGrade: GradeDecision | null; readOnly?: boolean }`. Import `createGrade`, `updateGrade` from `../../api/grading`. State: `type GradePanelMode = 'view' | 'editing' | 'submitting' | 'success' | 'error'`. Initialize: `mode = existingGrade ? 'view' : 'editing'`, `score = existingGrade ? String(existingGrade.score) : ''`, `feedback = existingGrade?.feedback ?? ''`, `errorMsg: string | null = null`, `grade: GradeDecision | null = existingGrade`. **Read-only mode** (`readOnly === true` or `mode === 'view'` with no edit button for co-docente): render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2">` with heading "Calificación", score `<span className="text-2xl font-bold text-[#E85D24]">{grade.score} / 100</span>`, feedback `<p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade.feedback}</p>`. If `!readOnly && mode === 'view'` also render `<button onClick={() => setMode('editing')} className="text-xs text-[#888888] hover:text-[#E85D24] transition-colors mt-2">Editar calificación</button>`. **Editing mode** (`mode === 'editing'` and `!readOnly`): render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">` with heading "Calificar entrega", score input `<input type="number" min="0" max="100" value={score} onChange={e => setScore(e.target.value)} className="w-24 bg-[#111111] border border-[#2e2e2e] rounded px-3 py-1.5 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none" placeholder="0–100" />`, feedback textarea `<textarea value={feedback} onChange={e => setFeedback(e.target.value)} rows={4} className="w-full bg-[#111111] border border-[#2e2e2e] rounded px-3 py-2 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none resize-none" placeholder="Comentarios para el estudiante..." />`, submit button `<button onClick={handleSubmit} disabled={mode === 'submitting'} className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 disabled:cursor-not-allowed transition-colors">{mode === 'submitting' ? 'Calificando...' : 'Calificar'}</button>`. If `errorMsg`: render `<p className="text-red-400 text-sm">{errorMsg}</p>`. **`handleSubmit`**: validate `n = parseInt(score)` → if `isNaN(n) || n < 0 || n > 100` set `errorMsg = 'El puntaje debe ser un número entre 0 y 100'` and return; set `mode = 'submitting'`; call `(grade ? updateGrade : createGrade)(submissionId, n, feedback)` → on success: update `grade` state, set `mode = 'success'` (show "Calificación guardada ✓" badge for 3 s via `setTimeout(() => setMode('view'), 3000)`) → on error: set `mode = 'editing'`, `errorMsg = err.message`. **No grade yet + readOnly**: render `<p className="text-[#888888] text-sm">Sin calificación.</p>`. Depends on T001, T003.

- [ ] T006 [US1] Update `src/pages/docente/SubmissionDetailPage.tsx` — add the `GradePanel` below the existing download / "Ver trazas" button row. The page receives `item: SubmissionListItem` from `useLocation().state`; the page URL contains `cid`, `aid`, `sid` params. Add import: `GradePanel` from `../../components/docente/GradePanel`. Add import: `useAuth` from `../../context/AuthContext` (already imported). After the download button `<div>` block, add: `<div className="border-t border-[#2e2e2e] pt-4 mt-4"><h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Calificación</h2><GradePanel submissionId={sid!} existingGrade={item.grade} readOnly={user?.role === 'co_docente'} /></div>`. Note: `item.grade` comes from the updated `SubmissionListItem` type (T001). If `user` is not yet imported from `useAuth()`, add `const { user } = useAuth()` after existing hook calls. Depends on T001, T003, T005.

---

## Phase 4 — US2: Student Views Their Grade (Priority: P2)

**Goal**: Student sees their grade (score + feedback) or "Pendiente de calificación" on the activity detail page. No edit controls shown.

**Independent test criteria**: Sign in as student → navigate to a graded activity → "Mi Calificación" section shows score and feedback read-only. Navigate to an ungraded activity (submission exists) → "Pendiente de calificación" shown. Navigate to an activity with no submission → grade section absent.

- [ ] T007 [P] [US2] Implement `src/components/student/GradeDisplay.tsx` (replace stub) — named export `GradeDisplay`. Props: `{ grade: GradeDecision | null; hasSubmission: boolean }`. If `!hasSubmission`: return `null` (render nothing). If `hasSubmission && grade === null`: render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4"><p className="text-[#888888] text-sm">Pendiente de calificación</p></div>`. If `hasSubmission && grade !== null`: render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2"><div className="flex items-center gap-2"><span className="text-2xl font-bold text-[#E85D24]">{grade.score} / 100</span></div><p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{grade.feedback}</p><p className="text-[#888888] text-xs">Calificado el {new Date(grade.graded_at).toLocaleDateString('es-BO')}</p></div>`. Depends on T001.

- [ ] T008 [US2] Update `src/pages/student/ActividadDetailPage.tsx` — add the `GradeDisplay` component below the "Mi Entrega" submission card. The page already calls `getSubmission(aid)` on mount and stores `submission: SubmissionResponse | null`. After the submission card block (and before any session/modal code), add: `import { GradeDisplay } from '../../components/student/GradeDisplay'`. In the JSX, after the submission card `<div>`: `<div className="border-t border-[#2e2e2e] pt-4 mt-4"><h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Mi Calificación</h2><GradeDisplay grade={submission?.grade ?? null} hasSubmission={submission !== null} /></div>`. This section is always rendered when the actividad is shown (the component itself returns null when `!hasSubmission`). Depends on T001, T007.

---

## Phase 5 — US3: Exam Session Form for Student (Priority: P2)

**Goal**: For `examen`-type activities, replace the file upload panel with a free-form text area. After submit, the form becomes read-only and shows the timestamp.

**Independent test criteria**: Sign in as student → open an `examen`-type activity (trace_enabled = false) → text area visible instead of file upload → type answers → click "Enviar respuesta" → loading state → on success, answer shown read-only + submission timestamp. Reload page → same read-only view. Deadline-passed activity → "Examen cerrado" shown.

- [ ] T009 [P] [US3] Implement `src/components/student/ExamForm.tsx` (replace stub) — named export `ExamForm`. Props: `{ actividadId: string; existingAnswer: string | null; submittedAt: string | null; deadline: string; sessionGate: boolean }`. Import `submitExamAnswer` from `../../api/grading`. Derive mode at render time: if `existingAnswer !== null` → `'submitted'`; else if `new Date() > new Date(deadline)` → `'closed'`; else if `sessionGate` → `'gated'`; else → `'draft'`. State: `answer = ''`, `submitting = false`, `localAnswer: string | null = existingAnswer`, `localSubmittedAt: string | null = submittedAt`, `errorMsg: string | null = null`. Recompute mode inside render using `localAnswer` instead of `existingAnswer`. **`'gated'` mode**: `<p className="text-[#888888] text-sm">Inicia la sesión de monitoreo para poder responder el examen.</p>`. **`'closed'` mode**: `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4"><p className="text-sm font-medium text-[#888888] mb-2">Examen cerrado</p>{localAnswer ? <p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{localAnswer}</p> : <p className="text-[#888888] text-xs">Sin respuesta registrada.</p>}</div>`. **`'submitted'` mode**: `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-2"><p className="text-green-400 text-xs">Enviado el {new Date(localSubmittedAt!).toLocaleString('es-BO')}</p><p className="text-[#f0f0f0] text-sm whitespace-pre-wrap">{localAnswer}</p></div>`. **`'draft'` mode**: `<div className="space-y-3"><textarea value={answer} onChange={e => setAnswer(e.target.value)} rows={12} disabled={submitting} className="w-full bg-[#111111] border border-[#2e2e2e] rounded px-3 py-2 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none resize-none disabled:opacity-50" placeholder="Escribe tus respuestas aquí..." />{errorMsg && <p className="text-red-400 text-sm">{errorMsg}</p>}<button onClick={handleSubmit} disabled={submitting || !answer.trim()} className="px-4 py-2 bg-[#E85D24] text-white text-sm rounded-lg hover:bg-[#d4521f] disabled:opacity-50 disabled:cursor-not-allowed transition-colors">{submitting ? 'Enviando...' : 'Enviar respuesta'}</button></div>`. **`handleSubmit`**: set `submitting = true`; call `submitExamAnswer(actividadId, answer)` → on success: set `localAnswer = res.exam_answer`, `localSubmittedAt = res.submitted_at` (mode auto-switches to `'submitted'`) → on error: set `errorMsg = err.message`; always set `submitting = false`. Depends on T001, T003.

- [ ] T010 [US3] Update `src/pages/student/ActividadDetailPage.tsx` — replace the file upload section with `ExamForm` for `examen`-type activities. Import `ExamForm` from `../../components/student/ExamForm`. The page already has `sessionStatus` and `actividad.trace_enabled` from MW6. Locate the block that renders the upload form (the file input, integrity checkbox, submit button). Wrap it with a conditional: if `actividad.tipo === 'examen'`, render `<ExamForm actividadId={aid!} existingAnswer={submission?.exam_answer ?? null} submittedAt={submission?.submitted_at ?? null} deadline={actividad.deadline} sessionGate={actividad.trace_enabled && sessionStatus !== 'active'} />` in place of the upload form. The session gate in ExamForm handles the "start session first" UX for traced exam activities — the existing session gate paragraph shown for non-exam uploads is unchanged. Ensure `ExamForm` is rendered inside the "Mi Entrega" section, above the `GradeDisplay` block added in T008. Depends on T001, T003, T008, T009.

---

## Phase 6 — US4: Investigación Stage Tracking (Priority: P3)

**Goal**: Docente can set/advance the stage of an `informe` submission. Student sees the current stage read-only.

**Independent test criteria**: Docente opens an `informe` submission detail → stage selector dropdown visible, current stage shown → select "desarrollo" → click "Actualizar etapa" → label updates immediately. Navigate to a `practica` submission detail → stage selector NOT shown. Sign in as student → view `informe` activity → "Etapa: Desarrollo" shown.

- [ ] T011 [P] [US4] Implement `src/components/docente/StageSelector.tsx` (replace stub) — named export `StageSelector`. Props: `{ submissionId: string; currentStage: InvestigaciónStage | null }`. Import `updateStage` from `../../api/grading`. Import `InvestigaciónStage` from `../../types/academic`. State: `selected: InvestigaciónStage = currentStage ?? 'propuesta'`, `saving = false`, `saved = false`, `errorMsg: string | null = null`. Render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-3">` with label "Etapa de investigación". Select: `<select value={selected} onChange={e => setSelected(e.target.value as InvestigaciónStage)} disabled={saving} className="bg-[#111111] border border-[#2e2e2e] rounded px-3 py-1.5 text-[#f0f0f0] text-sm focus:border-[#E85D24] focus:outline-none">` with options: `propuesta → Propuesta`, `desarrollo → Desarrollo`, `revisión → Revisión`, `aprobado → Aprobado`. Button: `<button onClick={handleSave} disabled={saving} className="px-4 py-2 bg-[#1a1a1a] border border-[#2e2e2e] text-[#f0f0f0] text-sm rounded-lg hover:border-[#E85D24] disabled:opacity-50 transition-colors">{saving ? 'Guardando...' : 'Actualizar etapa'}</button>`. If `saved`: show `<span className="text-green-400 text-xs">Etapa actualizada ✓</span>` (hide after 3 s via `setTimeout`). If `errorMsg`: `<p className="text-red-400 text-xs">{errorMsg}</p>`. **`handleSave`**: set `saving = true`; call `updateStage(submissionId, selected)` → on success: set `saved = true`, schedule `setTimeout(() => setSaved(false), 3000)` → on error: set `errorMsg = err.message`; always set `saving = false`. Depends on T001, T003.

- [ ] T012 [US4] Update `src/pages/docente/SubmissionDetailPage.tsx` — add conditional `StageSelector` and pass `actividadTipo` via navigation state. **Step 1**: In the submissions list page (`src/pages/docente/SubmissionsPage.tsx`), find where it navigates to the submission detail (the `onClick` or `Link` on each row). Currently it passes `item` (SubmissionListItem) via state. Update the navigate call to also pass `actividadTipo: actividad.tipo` (the actividad object is already available in that page's data). The state becomes `{ item, actividadTipo }`. **Step 2**: In `src/pages/docente/SubmissionDetailPage.tsx`, update `useLocation().state` destructuring to also extract `actividadTipo: ActividadTipo`. Add import: `ActividadTipo` from `../../types/academic`; `StageSelector` from `../../components/docente/StageSelector`. After the `GradePanel` block added in T006, add: `{actividadTipo === 'informe' && (<div className="border-t border-[#2e2e2e] pt-4 mt-4"><h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Etapa de investigación</h2><StageSelector submissionId={sid!} currentStage={item.stage} /></div>)}`. Note: `item.stage` is available from the updated `SubmissionListItem` type (T001). Depends on T001, T003, T006, T011.

---

## Phase 7 — US5: Defensa Oral Scheduling (Priority: P3)

**Goal**: Docente can assign date/time slots to students from a dedicated scheduling page. Student sees their assigned slot on the activity detail page.

**Independent test criteria**: Docente opens a `defensa_oral` activity → "Programar Defensas" button visible (absent for other types) → click → scheduling page shows student rows with datetime inputs → enter datetime + location → save → slot shown for that student. Sign in as student → `defensa_oral` activity → "Tu defensa: [datetime] — [location]". No slot assigned → "Defensa aún no programada".

- [ ] T013 [P] [US5] Implement `src/components/student/DefensaSlotDisplay.tsx` (replace stub) — named export `DefensaSlotDisplay`. Props: `{ slot: DefensaSlot | null }`. If `slot === null`: render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4"><p className="text-[#888888] text-sm">Defensa aún no programada.</p></div>`. Otherwise: render `<div className="bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg p-4 space-y-1"><p className="text-[#f0f0f0] text-sm font-medium">Tu defensa está programada</p><p className="text-[#f0f0f0]">{new Date(slot.scheduled_at).toLocaleString('es-BO', { dateStyle: 'full', timeStyle: 'short' })}</p>{slot.location && <p className="text-[#888888] text-sm">{slot.location}</p>}</div>`. Depends on T001.

- [ ] T014 [P] [US5] Implement `src/pages/docente/DefensaSchedulingPage.tsx` (replace stub) — named export `DefensaSchedulingPage`. Params: `{ cid, aid }` from `useParams<{ cid: string; aid: string }>()`. Import `listDefensaSlots`, `assignDefensaSlot` from `../../api/grading`. Import `DefensaSlotsResponse`, `DefensaSlot` from `../../types/academic`. Import `Link`, `useParams` from `react-router-dom`. State: `data: DefensaSlotsResponse | null`, `loading = true`, `error: string | null = null`. Also a per-row state map: `rowState: Record<string, { scheduledAt: string; location: string; saving: boolean; saved: boolean; errorMsg: string | null }>` initialized from `data.entries` on load (key = `entry.submission.id`, value = `{ scheduledAt: entry.slot?.scheduled_at ?? '', location: entry.slot?.location ?? '', saving: false, saved: false, errorMsg: null }`). On mount: `listDefensaSlots(cid!, aid!)` → set `data`, initialize `rowState`. Back link: `<Link to={\`/docente/cursos/${cid}/actividades/${aid}\`} className="text-[#888888] hover:text-[#E85D24] text-sm transition-colors">← Actividad</Link>`. Heading: "Programar Defensas". Show `<LoadingSpinner />` while loading; `<ErrorMessage />` on error. Table columns: Estudiante, Email, Fecha y Hora, Ubicación, Acción. Per row: student name + email (from `entry.submission.student`), `<input type="datetime-local" value={rowState[id].scheduledAt} onChange={...}` pre-filled from existing slot, `<input type="text" placeholder="Aula / Link (opcional)" value={rowState[id].location} onChange={...}`, `<button onClick={() => handleSave(entry.submission.id)}` disabled while saving, shows "Guardando..." → on save: calls `assignDefensaSlot(entry.submission.id, rowState[id].scheduledAt, rowState[id].location || null)` → on success: set `rowState[id].saved = true`, setTimeout 3 s to clear. If `rowState[id].errorMsg`: show red inline error below the row. Empty state (no entries): `<p className="text-[#888888]">No hay entregas registradas para esta actividad.</p>`. Depends on T001, T003, T004.

- [ ] T015 [US5] Update `src/pages/docente/ActividadDetailPage.tsx` — add "Programar Defensas" link for `defensa_oral` activities. The page already has `actividad.tipo` available from its fetched data. In the button row where "Ver Entregas" and "Ver Sesiones en Vivo" links appear, add: `{actividad.tipo === 'defensa_oral' && (<Link to={\`/docente/cursos/${cursoId}/actividades/${aid}/defensa-slots\`} className="inline-flex items-center gap-2 bg-[#1a1a1a] border border-[#2e2e2e] rounded-lg px-4 py-2 text-sm text-[#f0f0f0] hover:border-[#E85D24] transition-colors">Programar Defensas</Link>)}`. `Link` import already exists from MW5. Depends on T004.

- [ ] T016 [US5] Update `src/pages/student/ActividadDetailPage.tsx` — add `DefensaSlotDisplay` and fetch the student's assigned slot for `defensa_oral` activities. Import `DefensaSlotDisplay` from `../../components/student/DefensaSlotDisplay`. Import `getMyDefensaSlot` from `../../api/grading`. Import `DefensaSlot` from `../../types/academic`. Add state: `defensaSlot: DefensaSlot | null = null`. In the mount effect (alongside existing `getSubmission` call), add: `if (actividad.tipo === 'defensa_oral') { getMyDefensaSlot(aid!).then(setDefensaSlot).catch(() => {}) }`. In JSX, after the `ExamForm`/upload block and before the `GradeDisplay` block: `{actividad.tipo === 'defensa_oral' && (<div className="border-t border-[#2e2e2e] pt-4 mt-4"><h2 className="text-base font-semibold text-[#f0f0f0] mb-3">Mi Defensa</h2><DefensaSlotDisplay slot={defensaSlot} /></div>)}`. The slot fetch is fire-and-forget — errors are swallowed and the component shows "Defensa aún no programada" when slot is null. Depends on T001, T003, T010, T013.

---

## Phase 8 — Polish & Quality Gate

**Goal**: Ensure quality before marking MW7 complete.

- [ ] T017 Run React Doctor scan from repo root (`npx react-doctor@latest .`) — resolve ALL findings with ERROR severity before proceeding. Pay attention to: `GradePanel` controlled inputs (score is a `string` state — must stay string to allow partial input like "8" before "85"; do not parse in onChange); `ExamForm` mode derivation must not be a side effect (must be computed during render, not in a useEffect); `StageSelector` `useEffect` cleanup for the success timeout. Add a one-line justification comment for any WARNING-severity findings that are deferred.

- [ ] T018 Verify TypeScript compiles clean (`npx tsc --noEmit`) with 0 errors — confirm in particular: `InvestigaciónStage` type is correctly used in `StageSelector` select onChange cast; `DefensaSlot | null` return from `getMyDefensaSlot` is correctly typed (API function uses explicit `Promise<DefensaSlot | null>`); `SubmissionListItem.grade` and `.stage` do not break existing MW5 pages that spread or destructure the interface; `SubmissionResponse.exam_answer` does not break existing `getSubmission` call in student `ActividadDetailPage`.

- [ ] T019 Run smoke tests from `quickstart.md` — test scenarios 1–8 (US1 + US2 grading flow), 9–13 (US3 exam form), 14–17 (US4 stage tracking), 18–24 (US5 defensa scheduling). Confirm role guard tests 23–24 pass (student cannot see GradePanel inputs; admin redirected from docente routes). Confirm "Programar Defensas" button is absent from `practica` and `tarea` activity pages.

---

## Dependencies

```
Phase 1 (T001, T002, T003 all parallel) → Phase 2 (T004) → Phases 3–7

T001 [P] ─────────────────────────────────────────────────────────────┐
T002 [P] → T004 (router)                                               │
T003 [P] ──────────────────────────────────────────────────────────── │
                                                                        │
T004 → T014 [US5] (DefensaSchedulingPage)                             │
                                                                        │
T001, T003, T005 [P] → T006 [US1] (SubmissionDetailPage grade)        │
T001, T007 [P]        → T008 [US2] (student page grade section)       ├→ Phase 8
T001, T003, T008, T009 [P] → T010 [US3] (student page exam form)      │
T001, T003, T006, T011 [P] → T012 [US4] (SubmissionDetailPage stage)  │
T001, T003, T010, T013 [P] → T016 [US5] (student page defensa slot)   │
T004 → T015 [US5] (docente ActividadDetailPage "Programar Defensas")   │
```

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 1. No dependencies on other stories. T005 and T006.
- **US2 (P2)**: Can start after Phase 1. Independent. T007 and T008.
- **US3 (P2)**: Depends on US2 complete (T008) because both touch student ActividadDetailPage. T009 (component) can run in parallel with T007.
- **US4 (P3)**: Depends on US1 complete (T006) because both touch docente SubmissionDetailPage. T011 (component) can run in parallel with earlier phases.
- **US5 (P3)**: T013, T014, T015 can run in parallel after Phase 2. T016 depends on T010 (US3 complete).

---

## Parallel Execution Examples

**Phase 1 (fully independent — run together):**
- T001 (types), T002 (stubs), T003 (grading.ts API)

**After Phase 2, parallel start:**
- T005 (GradePanel component) + T007 (GradeDisplay component) + T009 (ExamForm component) + T011 (StageSelector component) + T013 (DefensaSlotDisplay component)
- T014 (DefensaSchedulingPage) — after T004 (router)

**Sequential within US1 → US4 on SubmissionDetailPage:**
```
T005 → T006 [US1] → T012 [US4]
```

**Sequential within US2 → US3 → US5 on student ActividadDetailPage:**
```
T007 → T008 [US2] → T010 [US3] → T016 [US5]
```

---

## Implementation Strategy

### MVP First (US1 Only — P1 Grading)

1. Complete Phase 1 (T001–T003) — types, stubs, API module
2. Complete Phase 2 (T004) — router
3. Complete Phase 3 (T005–T006) — GradePanel + docente SubmissionDetailPage
4. **STOP and VALIDATE**: Docente can grade submissions; co-docente sees read-only grade
5. Student sees `submission.grade` in the updated `SubmissionResponse` — GradeDisplay (US2) closes the loop

### Incremental Delivery

1. Foundation (T001–T004) — TypeScript clean, stubs in place
2. US1 Docente Grades (T005–T006) — core grading loop → validate
3. US2 Student Sees Grade (T007–T008) — student feedback loop → validate
4. US3 Exam Form (T009–T010) — exam activity support → validate
5. US4 Stage Tracking (T011–T012) — informe progress → validate
6. US5 Defensa Scheduling (T013–T016) — oral defense coordination → validate
7. Polish (T017–T019) — quality gate before marking MW7 done

---

## Notes

- `GradePanel` controls POST vs PATCH by checking `existingGrade !== null` — no try/catch retry needed
- `ExamForm` mode is derived at render time (not stored in state) using `localAnswer` to immediately switch view after successful submit
- `StageSelector` passes `actividadTipo` only if it's 'informe' — the SubmissionsPage must pass this via navigate state (see T012)
- `DefensaSchedulingPage` uses `listSubmissions`-equivalent data from `listDefensaSlots` — only students who have submitted appear
- `getMyDefensaSlot` may return null (no slot) or a `DefensaSlot` object — catch errors silently (404 = no submission yet)
- [P] tasks = different files, no conflicts
- Each user story phase is independently testable once complete
