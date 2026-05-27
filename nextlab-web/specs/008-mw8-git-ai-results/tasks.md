# Tasks: MW8 — Git & AI Results

**Input**: Design documents from `specs/008-mw8-git-ai-results/`

**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/git-ai.md ✓, quickstart.md ✓

**Tests**: Not requested — no test tasks generated.

**Organization**: Tasks grouped by user story (US1 → US2 → US3 → US4) with shared foundational types/API first.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no upstream dependencies)
- **[Story]**: Which user story (US1–US4) this task belongs to

---

## Phase 1: Foundational — Types & API Module

**Purpose**: Shared types and API functions required by ALL user stories. No user story work begins until this phase is complete.

**⚠️ CRITICAL**: T001 and T002 block every subsequent task.

- [X] T001 Extend `src/types/academic.ts` — add `git_enabled: boolean`, `ai_enabled: boolean`, `detection_enabled: boolean` to `StudentActividadEntry` and `Actividad`; add `repo_url: string | null` to `SubmissionResponse` and `SubmissionListItem`; append interfaces: `GitAnalysisReport`, `GitFileEntry`, `AIReport`, `AIReportSection`, `DetectionReport`, `FlaggedSegment`, and `ActividadFlags = { git_enabled: boolean; ai_enabled: boolean; detection_enabled: boolean }`
- [X] T002 Create `src/api/gitAnalysis.ts` — local `toApiError` (matching grading.ts pattern); export `submitGitRepo(actividadId, repoUrl): Promise<SubmissionResponse>` (POST `/actividades/:aid/git-submission`); `updateGitRepo(actividadId, repoUrl): Promise<SubmissionResponse>` (PATCH `/actividades/:aid/git-submission`); `getGitAnalysis(submissionId): Promise<GitAnalysisReport>` (GET `/submissions/:sid/git-analysis`); `getAIReport(submissionId): Promise<AIReport>` (GET `/submissions/:sid/ai-report`); `getDetectionReport(submissionId): Promise<DetectionReport>` (GET `/submissions/:sid/detection-report`)

**Checkpoint**: Types compile cleanly (`pnpm tsc --noEmit`). Five API functions exported with correct signatures.

---

## Phase 2: User Story 1 — Student Submits Git Repo (Priority: P1) 🎯 MVP

**Goal**: Student on a git-enabled activity sees a URL input form instead of file upload. Can submit, view confirmation, and edit within the window.

**Independent Test**: Sign in as student → open a git-enabled activity → git URL form shown (not file upload) → enter `https://github.com/test/repo` → click "Enviar repositorio" → confirmation view shows URL and timestamp → reload → same confirmation shown.

- [X] T003 [US1] Create `src/components/student/GitSubmissionForm.tsx` — props: `{ actividadId: string; existingSubmission: SubmissionResponse | null; isPastDeadline: boolean; onSuccess: (s: SubmissionResponse) => void }`; local state `mode: 'form' | 'submitting' | 'confirmed' | 'editing' | 'updating' | 'closed'` and `repoUrl: string`; mode derived at render (`existingSubmission && mode !== 'editing'` → confirmed; `isPastDeadline && !existingSubmission` → closed; else form); instructional note showing `import.meta.env.VITE_GIT_SYSTEM_ACCOUNT ?? 'nextlab-bot'`; inline validation (empty or no `://` → error, no API call); calls `submitGitRepo`/`updateGitRepo` from `src/api/gitAnalysis.ts`; confirmed view shows repo_url + submitted_at + "Editar" button (hidden when is_locked); "Editar" sets mode to 'editing' and pre-fills repoUrl input; closed view shows "El período de entrega ha cerrado." with no submit button
- [X] T004 [US1] Update `src/pages/student/ActividadDetailPage.tsx` — import `GitSubmissionForm` from `../../components/student/GitSubmissionForm`; in the submission section, wrap the existing file-upload/confirmation block in `actividad.tipo !== 'examen' && !actividad.git_enabled` condition; add a new branch `actividad.git_enabled` that renders `<GitSubmissionForm actividadId={aid!} existingSubmission={submission} isPastDeadline={isPast} onSuccess={setSubmission} />`; the git branch MUST NOT show file upload controls, the replace/delete buttons, or the filename/size display

**Checkpoint**: Student portal correctly branches on `actividad.git_enabled`. TypeScript clean.

---

## Phase 3: User Story 2 — Docente Views Git Analysis (Priority: P2)

**Goal**: Docente sees a "Ver análisis Git" link on submission detail pages for git submissions. Clicking opens a page showing the full git analysis report (commit stats, authors, file table) or a pending/error state.

**Independent Test**: Sign in as docente → submission detail page for a git submission → "Ver análisis Git" link present → click → git analysis page loads → shows commit count, author list, date range, file change table (or pending/error state if applicable).

- [X] T005 [P] [US2] Update `src/pages/docente/SubmissionsPage.tsx` — import `ActividadFlags` from `../../types/academic`; after calling `getActividad`, extract `actividadFlags: ActividadFlags = { git_enabled: actividad.git_enabled, ai_enabled: actividad.ai_enabled, detection_enabled: actividad.detection_enabled }`; pass `actividadFlags` as a new prop to `<SubmissionTable>`
- [X] T006 [P] [US2] Update `src/components/docente/SubmissionTable.tsx` — add `actividadFlags?: ActividadFlags` to props interface; in the `navigate` call for each row, change state from `{ item, actividadTipo }` to `{ item, actividadTipo, actividadFlags }` so SubmissionDetailPage can read the flags
- [X] T007 [US2] Update `src/pages/docente/SubmissionDetailPage.tsx` — extend location.state parsing: extract `actividadFlags` (type `ActividadFlags | undefined`) from `rawState`; derive `routePrefix` from `useLocation().pathname` (starts with `/codocente` → `'/codocente'`, else `'/docente'`); after the existing button row, add a new "Análisis y Reportes" section showing a "Ver análisis Git" `<Link>` to `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${sid}/git-analysis` — only rendered when `item?.repo_url !== null && item?.repo_url !== undefined`
- [X] T008 [P] [US2] Create `src/pages/docente/GitAnalysisPage.tsx` — `useParams<{ cid: string; aid: string; subid: string }>()`; `useLocation().pathname` for back-link prefix; on mount call `getGitAnalysis(subid)` with loading/error state; render: `<LoadingSpinner />` while loading; `<ErrorMessage>` on fetch error; when data arrives: pending state (`status === 'pending'`) → info banner "Análisis en curso. Vuelve a revisar en unos minutos."; error state (`status === 'error'`) → error banner with `data.error_message`; done state → summary row (commit count, date range as formatted dates), authors list, file change table (`path | additions | deletions` columns with green/red text for additions/deletions); back link `← Entrega` pointing to `${prefix}/cursos/${cid}/actividades/${aid}/submissions/${subid}` (derive prefix from pathname); h1 "Análisis Git"
- [X] T009 [US2] Update `src/router/index.tsx` — import `GitAnalysisPage` from `../pages/docente/GitAnalysisPage`; add route `{ path: '/docente/cursos/:cid/actividades/:aid/submissions/:subid/git-analysis', element: <GitAnalysisPage /> }` inside the docente ProtectedRoute block (after the existing defensa-slots route); add the same path pattern under the codocente ProtectedRoute block (`/codocente/...`) pointing to the same `<GitAnalysisPage />`

**Checkpoint**: Docente can navigate to git analysis page and see at least the pending/loading state. TypeScript clean.

---

## Phase 4: User Story 3 — Docente Views AI Report (Priority: P3)

**Goal**: Docente sees "Ver informe IA" link when AI analysis is available. Clicking opens a page showing all AI-generated structured sections or a pending/error state.

**Independent Test**: Sign in as docente → submission detail for a submission on an AI-enabled activity → "Ver informe IA" link present → click → AI report page shows sections list (or pending banner).

- [X] T010 [P] [US3] Create `src/pages/docente/AIReportPage.tsx` — same structure as `GitAnalysisPage`; `useParams<{ cid: string; aid: string; subid: string }>()`; call `getAIReport(subid)` on mount; pending → "Análisis IA en curso. Vuelve a revisar en unos minutos."; error → error banner with `data.error_message`; done → render each section in `data.sections` as a card with `section.title` as heading and `section.content` in a `<pre className="whitespace-pre-wrap ...">` block; h1 "Informe IA"; back link to submission detail
- [X] T011 [US3] Update `src/pages/docente/SubmissionDetailPage.tsx` — in the "Análisis y Reportes" section added in T007, add a "Ver informe IA" `<Link>` to `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${sid}/ai-report` — only rendered when `actividadFlags?.ai_enabled === true`
- [X] T012 [US3] Update `src/router/index.tsx` — import `AIReportPage` from `../pages/docente/AIReportPage`; add route for `/docente/cursos/:cid/actividades/:aid/submissions/:subid/ai-report` inside docente block; add same under codocente block

**Checkpoint**: Docente can navigate to AI report page and see pending or completed report. TypeScript clean.

---

## Phase 5: User Story 4 — Docente Views Detection Report (Priority: P3)

**Goal**: Docente sees "Ver informe de detección" link when detection is available. Clicking opens a page showing similarity score and flagged segments or a pending/error state.

**Independent Test**: Sign in as docente → submission detail for a detection-enabled activity → "Ver informe de detección" link present → click → detection report page shows similarity score and flagged segments table (or pending banner).

- [X] T013 [P] [US4] Create `src/pages/docente/DetectionReportPage.tsx` — same structure as `AIReportPage`; call `getDetectionReport(subid)` on mount; pending → "Detección en curso. Vuelve a revisar en unos minutos."; error → error banner with `data.error_message`; done → similarity score displayed prominently (e.g., large number with "% similitud" label, color-coded: ≤20 green, 21–50 yellow, >50 red using Tailwind classes); flagged segments table with columns "Fuente", "Coincidencia %", "Fragmento" (content is optional — skip column if all segments lack it); empty flagged_segments → "Sin segmentos detectados."; h1 "Informe de Detección"; back link to submission detail
- [X] T014 [US4] Update `src/pages/docente/SubmissionDetailPage.tsx` — in the "Análisis y Reportes" section, add a "Ver informe de detección" `<Link>` to `${routePrefix}/cursos/${cid}/actividades/${aid}/submissions/${sid}/detection-report` — only rendered when `actividadFlags?.detection_enabled === true`
- [X] T015 [US4] Update `src/router/index.tsx` — import `DetectionReportPage` from `../pages/docente/DetectionReportPage`; add route for `/docente/cursos/:cid/actividades/:aid/submissions/:subid/detection-report` inside docente block; add same under codocente block

**Checkpoint**: All four user stories functional. TypeScript clean. Three analysis report links appear/hide correctly based on actividadFlags.

---

## Phase 6: Polish & Quality Gate

**Purpose**: Confirm zero blocking issues across the full src/ directory.

- [X] T016 Run React Doctor quality gate on `src/` — resolve any ERROR severity findings; for any WARNING findings, add a one-line comment at the affected site with justification if deferred; confirm score ≥80 and 0 ERRORs

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No external dependencies — start immediately
- **US1 (Phase 2)**: Depends on T001, T002 (Phase 1 complete)
- **US2 (Phase 3)**: Depends on T001, T002 (Phase 1 complete). T005 and T006 [P] can run right after Phase 1; T007 waits for T005+T006; T008 [P] can run alongside T007; T009 waits for T008
- **US3 (Phase 4)**: Depends on T007 (SubmissionDetailPage section exists). T010 [P] can start immediately after Phase 1; T011 waits for T010 (page must exist before linking to it); T012 waits for T010
- **US4 (Phase 5)**: Same pattern as US3. T013 [P] can start after Phase 1; T014 waits for T013; T015 waits for T013
- **Polish (Phase 6)**: Depends on all user story phases complete

### User Story Dependencies

- **US1 (P1)**: Independent — depends only on Phase 1 (T001, T002)
- **US2 (P2)**: Independent — depends only on Phase 1; no dependency on US1
- **US3 (P3)**: Depends on T007 (US2) for the section in SubmissionDetailPage where the AI link is added
- **US4 (P3)**: Depends on T007 (US2) for the same section; can proceed in parallel with US3

### Within Each User Story

- New page components (T008, T010, T013) can be created in parallel since they're new files
- Router updates (T009, T012, T015) must happen after the page they import is created
- SubmissionDetailPage link additions (T007, T011, T014) are sequential edits to the same file

---

## Parallel Opportunities

```
After Phase 1 completes (T001, T002):

  Parallel group A:
    T003 [US1] GitSubmissionForm.tsx  (new file)
    T005 [US2] SubmissionsPage.tsx    (new prop)
    T006 [US2] SubmissionTable.tsx    (new prop)
    T008 [US2] GitAnalysisPage.tsx    (new file)
    T010 [US3] AIReportPage.tsx       (new file)
    T013 [US4] DetectionReportPage.tsx (new file)

  Then sequentially per story:
    T004 after T003 (uses GitSubmissionForm)
    T007 after T005+T006 (reads actividadFlags from state)
    T009 after T008 (imports GitAnalysisPage)
    T011 after T010 (links to AIReportPage)
    T012 after T010
    T014 after T013 (links to DetectionReportPage)
    T015 after T013
    T016 after all above
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (T001, T002)
2. Complete Phase 2/US1 (T003, T004)
3. **STOP and VALIDATE**: Git form shows for git-enabled activities; file upload shows for others; submit and confirm flow works

### Incremental Delivery

1. Phase 1 → Foundation ready
2. US1 (T003–T004) → Students can submit git repos ✓
3. US2 (T005–T009) → Docente sees git analysis report ✓
4. US3 (T010–T012) → Docente sees AI report ✓
5. US4 (T013–T015) → Docente sees detection report ✓
6. Polish (T016) → Quality gate passes ✓

---

## Notes

- `ActividadFlags` type added in T001 — used in T005, T006, T007, T011, T014
- `routePrefix` in SubmissionDetailPage (T007): `location.pathname.startsWith('/codocente') ? '/codocente' : '/docente'`
- `params.cid`, `params.aid`, `params.sid` must be extracted in SubmissionDetailPage from `useParams` for link construction — confirm they're already available from existing `useParams` call
- All three report pages (T008, T010, T013) share the same loading/pending/error/done pattern — implement them consistently
- Detection score color coding (T013) is cosmetic display logic only — constitution VII compliant
- `import.meta.env.VITE_GIT_SYSTEM_ACCOUNT` in GitSubmissionForm — falls back to `'nextlab-bot'` if env var not set
